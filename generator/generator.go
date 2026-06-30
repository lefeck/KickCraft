package generator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kickcraft/config"
	"github.com/kickcraft/logger"
	"github.com/kickcraft/utils"
)

// ProgressCallback is called to report build progress
type ProgressCallback func(step, status, message string)

// Options holds generator configuration
type Options struct {
	TempDir string
}

// Generator handles ISO generation
type Generator struct {
	options Options
	Path    *utils.Path
}

// Result represents the result of ISO generation
type Result struct {
	OutputPath   string
	BuildDir     string
	VolumeLabel  string
	InstallMedia string
	Size         int64
	Duration     time.Duration
}

// New creates a new generator
func New(opts Options) *Generator {
	return &Generator{
		options: opts,
		Path:    utils.NewPath(opts.TempDir),
	}
}

// InitDirs creates all required directories for the build system
func (g *Generator) InitDirs() error {
	dirs := []string{
		g.options.TempDir,
		g.Path.DownloadDir(),
		g.Path.BuildDir(),
		g.Path.Mount(),
		g.Path.Packages(),
		g.Path.Scripts(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	logger.Info("Build directories initialized at: %s", g.options.TempDir)
	return nil
}

// Generate generates an ISO with the given Kickstart configuration.
// installMedia is "cdrom" (default) or "harddrive". It selects the
// inst.ks= boot param format. extraBootParams (e.g. "biosdevname=0
// net.ifnames=0 console=tty0 console=ttyS0,115200n8") is appended
// after inst.ks= in both grub.cfg and isolinux.cfg.
// additionalPackages is a list of RPM package names to download and
// bundle into build/mnt/packages/ for offline installation.
func (g *Generator) Generate(distro string, cfg *config.KickstartConfig, sourceISO string, destinationISO string, installMedia string, extraBootParams string, additionalPackages []string, progress ProgressCallback) (*Result, error) {
	start := time.Now()
	workDir := g.Path.BuildDir()

	if installMedia == "" {
		installMedia = "cdrom"
	}

	report := func(step, status, msg string) {
		if progress != nil {
			progress(step, status, msg)
		}
	}

	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	logger.Info("Starting ISO generation for %s (installMedia=%s)", distro, installMedia)
	logger.Info("Work directory: %s", workDir)

	// Verify required tools
	report("prepare", "running", "Ensuring required tools are installed...")
	if err := g.ensureTools(); err != nil {
		report("prepare", "failed", err.Error())
		return nil, err
	}
	report("prepare", "completed", "Installation environment ready")

	// Step 1: Acquire source ISO
	report("source", "running", "Using local ISO: "+filepath.Base(sourceISO))
	if err := g.acquireSource(sourceISO, workDir, func(msg string) {
		report("source", "running", msg)
	}); err != nil {
		report("source", "failed", err.Error())
		return nil, fmt.Errorf("failed to acquire source: %w", err)
	}
	report("source", "completed", "Local ISO ready")

	// Step 2: Generate kickstart file
	report("kickstart", "running", "Generating kickstart configuration...")
	ksPath := g.Path.KickstartFile()
	if err := os.WriteFile(ksPath, []byte(cfg.ToString()), 0644); err != nil {
		report("kickstart", "failed", err.Error())
		return nil, fmt.Errorf("failed to write kickstart file: %w", err)
	}
	logger.Info("Kickstart file written: %s", ksPath)
	report("kickstart", "completed", fmt.Sprintf("Kickstart file ready: %s", filepath.Base(ksPath)))

	// Step 3: Modify boot configuration
	// Read the volume label from the source ISO so the hd: boot param
	// is correct (matches xorriso's -V value used during repackaging).
	volumeLabel := ""
	if sourceISO != "" {
		if label, err := utils.ReadISOVolumeID(sourceISO); err == nil {
			volumeLabel = label
			logger.Info("Using volume label for boot param: %s", volumeLabel)
		}
	}
	report("boot", "running", "Patching boot configuration...")
	if err := g.patchBootConfig(workDir, installMedia, volumeLabel, extraBootParams); err != nil {
		report("boot", "failed", err.Error())
		return nil, fmt.Errorf("failed to patch boot config: %w", err)
	}
	report("boot", "completed", "Boot configuration patched")

	// Step 3b: Download additional packages into build/mnt/packages/
	if len(additionalPackages) > 0 {
		report("packages", "running", fmt.Sprintf("Downloading %d additional package(s)...", len(additionalPackages)))
		if err := g.DownloadAdditionalPackages(additionalPackages, distro, func(step, status, msg string) {
			report("packages", status, msg)
		}); err != nil {
			report("packages", "failed", err.Error())
			// Non-fatal: warn but continue — ISO can still be built.
			logger.Warn("Additional packages download failed, continuing: " + err.Error())
		}
		report("packages", "completed", "Additional packages ready")
	}

	// Step 3c: Write config.sh into build/mnt/script/ so it ends up inside
	// the ISO. This script sets up /mnt/packages/ as a local yum/dnf repo
	// and is sourced by Anaconda during %pre/%post so offline RPMs can be
	// resolved without network access. The script also contains
	// `yum/dnf install` commands for each downloaded package.
	report("repo", "running", "Writing offline repo script (config.sh)...")
	confScriptPath := g.Path.ScriptFile(ConfScriptName)
	if err := WriteConfScript(confScriptPath, additionalPackages); err != nil {
		report("repo", "failed", err.Error())
		return nil, fmt.Errorf("failed to write config.sh: %w", err)
	}
	logger.Info("config.sh written to: %s", confScriptPath)
	report("repo", "completed", "Offline repo script ready")

	// Step 4: Repackage ISO
	report("repackage", "running", "Repackaging ISO image...")
	outputDir := g.Path.DownloadDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use destinationISO if provided, otherwise fallback to distro-based name
	outputFilename := destinationISO
	if outputFilename == "" {
		outputFilename = fmt.Sprintf("%s-%s", distro, time.Now().Format("2006-01-02"))
	}
	// Strip any existing .iso suffix to avoid double extension
	outputFilename = strings.TrimSuffix(outputFilename, ".iso")
	outputFilename += ".iso"
	outputPath := filepath.Join(outputDir, outputFilename)
	logger.Info("Output ISO path: %s", outputPath)

	volumeLabelUsed, err := g.repackageISO(workDir, outputPath, sourceISO)
	if err != nil {
		report("repackage", "failed", err.Error())
		return nil, fmt.Errorf("failed to repackage ISO: %w", err)
	}
	logger.Info("Volume label used in ISO: %s (installMedia=%s)", volumeLabelUsed, installMedia)
	report("repackage", "completed", "ISO repackaged successfully")

	// Step 5: Embed MD5 into ISO using implantisomd5
	report("checksum", "running", "Embedding MD5 checksum into ISO...")
	if err := embedISO_md5(outputPath, func(msg string) {
		report("checksum", "running", msg)
	}); err != nil {
		report("checksum", "failed", err.Error())
		return nil, fmt.Errorf("failed to embed MD5: %w", err)
	}
	logger.Info("MD5 embedded into ISO via implantisomd5")
	report("checksum", "completed", "MD5 embedded via implantisomd5")

	// Step 6: Verify ISO integrity with checkisomd5
	report("verify", "running", "Verifying ISO integrity...")
	if err := verifyISO_md5(outputPath, func(msg string) {
		report("verify", "running", msg)
	}); err != nil {
		report("verify", "failed", err.Error())
		return nil, fmt.Errorf("ISO integrity check failed: %w", err)
	}
	logger.Info("ISO integrity verified via checkisomd5")
	report("verify", "completed", "ISO integrity verified")

	// Get output file info
	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat output file: %w", err)
	}

	return &Result{
		OutputPath:   outputPath,
		BuildDir:     workDir,
		VolumeLabel:  volumeLabelUsed,
		InstallMedia: installMedia,
		Size:         info.Size(),
		Duration:     time.Since(start),
	}, nil
}

// acquireSource extracts the source ISO or copies from local file
func (g *Generator) acquireSource(sourceISO, destDir string, logFn func(string)) error {
	// Check if source ISO is a local file
	if sourceISO != "" && !strings.HasPrefix(sourceISO, "http") {
		if _, err := os.Stat(sourceISO); os.IsNotExist(err) {
			return fmt.Errorf("source ISO not found: %s", sourceISO)
		}
		logger.Info("Extracting local source ISO: %s", sourceISO)
		if logFn != nil {
			logFn(fmt.Sprintf("Extracting ISO: %s", filepath.Base(sourceISO)))
		}
		return g.extractISO(sourceISO, destDir, logFn)
	}

	// For HTTP sources, download the ISO file
	if strings.HasPrefix(sourceISO, "http") {
		logger.Info("Downloading source ISO from: %s", sourceISO)
		if logFn != nil {
			logFn(fmt.Sprintf("Downloading ISO from: %s", sourceISO))
		}
		// Download to a temporary file
		tmpDir := filepath.Dir(destDir)
		tmpISO := filepath.Join(tmpDir, "source-"+filepath.Base(sourceISO))
		if err := g.downloadFile(sourceISO, tmpISO, logFn); err != nil {
			return fmt.Errorf("failed to download ISO: %w", err)
		}
		// Extract the downloaded ISO
		logger.Info("Extracting downloaded source ISO: %s", tmpISO)
		if logFn != nil {
			logFn(fmt.Sprintf("Extracting ISO: %s", filepath.Base(tmpISO)))
		}
		return g.extractISO(tmpISO, destDir, logFn)
	}

	// No source provided - need to use default
	return fmt.Errorf("source ISO required")
}

// downloadFile downloads a file from a URL to a local path
func (g *Generator) downloadFile(url, destPath string, logFn func(string)) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP download failed with status: %d", resp.StatusCode)
	}

	// Create destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Get content length for progress
	contentLength := resp.ContentLength
	downloaded := int64(0)
	buffer := make([]byte, 32*1024)
	reader := resp.Body
	lastLoggedPercent := 0

	if logFn != nil {
		logFn("Download started...")
	}
	logger.Info("Download started: %s", url)

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			// Write to file
			written, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("write error: %w", writeErr)
			}
			if written != n {
				return fmt.Errorf("incomplete write")
			}
			downloaded += int64(n)

			// Update progress every 5%
			if contentLength > 0 {
				percent := int(float64(downloaded) / float64(contentLength) * 100)
				if percent >= lastLoggedPercent+5 || percent == 100 {
					progressMsg := fmt.Sprintf("Download progress: %.1f%%", float64(percent))
					logger.Info(progressMsg)
					if logFn != nil {
						logFn(progressMsg)
					}
					lastLoggedPercent = percent
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error: %w", err)
		}
	}

	logger.Info("Download completed: %s", destPath)
	if logFn != nil {
		logFn("Download completed")
	}
	return nil
}

// extractISO extracts an ISO file using xorriso or 7z
func (g *Generator) extractISO(isoPath, destDir string, logFn func(string)) error {
	isoSize, _ := os.Stat(isoPath)
	isoSizeMB := float64(0)
	if isoSize != nil {
		isoSizeMB = float64(isoSize.Size()) / (1024 * 1024)
	}
	if logFn != nil {
		logFn(fmt.Sprintf("Extracting %s (%.1f MB)...", filepath.Base(isoPath), isoSizeMB))
	}

	// Try xorriso first
	if g.isCommandAvailable("xorriso") {
		if logFn != nil {
			logFn("Using xorriso for extraction...")
		}
		cmd := exec.Command("xorriso", "-osirrox", "on", "-indev", isoPath, "-extract", "/", destDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Warn("xorriso extraction failed: %s", string(output))
			if logFn != nil {
				logFn(fmt.Sprintf("xorriso failed, trying 7z fallback..."))
			}
		} else {
			logger.Info("ISO extracted with xorriso")
			if logFn != nil {
				logFn(fmt.Sprintf("xorriso extraction completed"))
			}
			return nil
		}
	}

	// Fallback to 7z
	if g.isCommandAvailable("7z") {
		if logFn != nil {
			logFn("Using 7z for extraction...")
		}
		cmd := exec.Command("7z", "x", "-y", "-o"+destDir, isoPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("7z extraction failed: %s", string(output))
		}
		logger.Info("ISO extracted with 7z")
		if logFn != nil {
			logFn("7z extraction completed")
		}
		return nil
	}

	return fmt.Errorf("no suitable extraction tool available (xorriso or 7z required)")
}

// patchBootConfig patches the boot configuration to include kickstart
// and any extra boot params. installMedia is "cdrom" (default) or
// "harddrive" — it selects the inst.ks= format. volumeLabel is needed
// only for the harddrive case. extraBootParams is appended after
// inst.ks= (e.g. "biosdevname=0 net.ifnames=0 console=tty0 ...").
func (g *Generator) patchBootConfig(workDir string, installMedia string, volumeLabel string, extraBootParams string) error {
	// Determine boot param based on install media type
	var ksParam string
	switch installMedia {
	case "harddrive":
		if volumeLabel == "" {
			volumeLabel = "KickCraft-KS"
		}
		ksParam = fmt.Sprintf("inst.ks=hd:LABEL=%s:/ks.cfg", volumeLabel)
	default: // cdrom
		ksParam = "inst.ks=cdrom:/ks.cfg"
	}
	if extraBootParams != "" {
		ksParam = ksParam + " " + extraBootParams
	}
	logger.Info("Install media: %s, boot param: %s", installMedia, ksParam)

	// Patch UEFI grub.cfg
	grubCfg := g.Path.GrubConfigFile(filepath.Join("EFI", "BOOT", "grub.cfg"))
	if _, err := os.Stat(grubCfg); err == nil {
		logger.Info("Patching UEFI grub.cfg with kickstart parameter: %s", ksParam)
		if err := g.patchGrubCfg(grubCfg, ksParam); err != nil {
			return err
		}
		logger.Info("UEFI grub.cfg patched successfully")
	}

	// Patch BIOS isolinux.cfg
	isolinuxCfg := g.Path.IsolinuxConfigFile(filepath.Join("isolinux", "isolinux.cfg"))
	if _, err := os.Stat(isolinuxCfg); err == nil {
		logger.Info("Patching BIOS isolinux.cfg with kickstart parameter: %s", ksParam)
		if err := g.patchIsolinuxCfg(isolinuxCfg, ksParam); err != nil {
			return err
		}
		logger.Info("BIOS isolinux.cfg patched successfully")
	}

	return nil
}

// patchGrubCfg adds kickstart parameter to a grub.cfg file. grub.cfg
// uses linuxefi/linux directive names (not "append").
func (g *Generator) patchGrubCfg(path, ksParam string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// grub.cfg boot lines start with "linuxefi" or "linux"
		if strings.HasPrefix(trimmed, "linuxefi") || strings.HasPrefix(trimmed, "linux ") || trimmed == "linux" {
			if !strings.Contains(lines[i], "inst.ks=") {
				lines[i] = line + " " + ksParam
			}
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// patchIsolinuxCfg adds kickstart parameter to an isolinux.cfg file.
// isolinux.cfg uses the "append" directive.
func (g *Generator) patchIsolinuxCfg(path, ksParam string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "append ") || trimmed == "append" {
			if !strings.Contains(lines[i], "inst.ks=") {
				lines[i] = line + " " + ksParam
			}
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// repackageISO creates the final ISO. If sourceISO is non-empty, the
// volume label is dynamically extracted from it; otherwise a fallback
// label is used. Returns the volume label that was used.
func (g *Generator) repackageISO(sourceDir, outputPath, sourceISO string) (string, error) {
	if !g.isCommandAvailable("xorriso") {
		return "", fmt.Errorf("xorriso is required for ISO generation")
	}

	// Dynamically detect the volume label from the source ISO. Fall back
	// to a generic KickCraft label if detection fails.
	volumeLabel := "KickCraft-KS"
	if sourceISO != "" {
		if label, err := utils.ReadISOVolumeID(sourceISO); err != nil {
			logger.Warn("Could not read source ISO volume label (%s): %v — using fallback", sourceISO, err)
		} else {
			volumeLabel = label
			logger.Info("Detected source ISO volume label: %s", volumeLabel)
		}
	}

	// Build xorriso command
	logger.Info("Building xorriso command with volume label: %s", volumeLabel)
	args := []string{
		"-as", "mkisofs",
		"-r",
		"-V", volumeLabel,
		"-J",
		"-joliet-long",
		"-pad",
		"-no-emul-boot",
		"-boot-load-size", "4",
		"-boot-info-table",
		"-o", outputPath,
		sourceDir,
	}

	// Add boot options after the mkisofs args (xorriso -as mkisofs expects these after -as mkisofs)
	isolinuxBin := g.Path.IsolinuxBin()
	if _, err := os.Stat(isolinuxBin); err == nil {
		logger.Info("Bootable ISO detected, adding isolinux boot options")
		bootIdx := len(args) - 1 // insert before outputPath and sourceDir
		newArgs := make([]string, 0, len(args)+2)
		newArgs = append(newArgs, args[:bootIdx]...)
		newArgs = append(newArgs, "-b", "isolinux/isolinux.bin", "-c", "isolinux/boot.cat")
		newArgs = append(newArgs, args[bootIdx:]...)
		args = newArgs
	}

	logger.Info("Running xorriso mkisofs emulation...")
	cmd := exec.Command("xorriso", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return volumeLabel, fmt.Errorf("xorriso failed: %s", string(output))
	}

	outputSize, _ := os.Stat(outputPath)
	if outputSize != nil {
		logger.Info("ISO created: %s (%.1f MB)", outputPath, float64(outputSize.Size())/(1024*1024))
	} else {
		logger.Info("ISO created: %s", outputPath)
	}
	return volumeLabel, nil
}

// isCommandAvailable checks if a command is available
func (g *Generator) isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ensureTools installs required tools if missing (isomd5sum for implantisomd5/checkisomd5)
func (g *Generator) ensureTools() error {
	if g.isCommandAvailable("implantisomd5") {
		return nil
	}
	logger.Info("implantisomd5 not found, installing isomd5sum...")
	cmd := exec.Command("yum", "install", "-y", "isomd5sum")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install isomd5sum: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	if !g.isCommandAvailable("implantisomd5") {
		return fmt.Errorf("implantisomd5 still not available after install")
	}
	logger.Info("isomd5sum installed successfully")
	return nil
}

// DownloadAdditionalPackages downloads RPM packages and their dependencies
// into build/mnt/packages/. It mirrors the logic of build_offline_packages
// in centos-autoinstall-generator-tools.sh: uses yumdownloader --resolve
// to fetch packages, and falls back to dnf download if yumdownloader is
// unavailable. A local "extra" repo pointing to build/mnt/packages/extra/
// is registered so that dependencies can be resolved offline.
func (g *Generator) DownloadAdditionalPackages(packages []string, distro string, progress ProgressCallback) error {
	if len(packages) == 0 {
		return nil
	}
	report := func(msg string) {
		if progress != nil {
			progress("packages", "running", msg)
		}
	}
	report("Starting additional packages download...")

	pkgDir := g.Path.Packages()

	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("failed to create packages directory: %w", err)
	}

	// Normalize: trim whitespace and remove blank/comment lines
	var clean []string
	for _, p := range packages {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return nil
	}

	// Try yumdownloader first (EL8/EL9), then dnf download as fallback
	downloader := "yumdownloader"
	if !g.isCommandAvailable(downloader) {
		downloader = "dnf"
	}
	report(fmt.Sprintf("Using downloader: %s", downloader))

	// Download packages one-by-one so unavailable ones can be skipped gracefully
	var downloaded, skipped []string
	for i, pkg := range clean {
		report(fmt.Sprintf("Downloading [%d/%d]: %s", i+1, len(clean), pkg))

		var args []string
		if downloader == "yumdownloader" {
			args = []string{"--resolve", pkg, "--destdir", pkgDir}
		} else {
			args = []string{"download", "--resolve", "--destdir", pkgDir, pkg}
		}

		cmd := exec.Command(downloader, args...)
		cmd.Env = append(cmd.Env, "LC_ALL=C")
		out, err := cmd.CombinedOutput()

		if err != nil {
			// Check if the package is simply not available in the repos
			outStr := strings.ToLower(string(out))
			notFound := strings.Contains(outStr, "no package") ||
				strings.Contains(outStr, "error") ||
				strings.Contains(outStr, "no matches")

			if notFound {
				skipped = append(skipped, pkg)
				report(fmt.Sprintf("Skipping (not available): %s", pkg))
				logger.Warn("Package not available: %s", pkg)
			} else {
				// Real error — still skip but warn
				skipped = append(skipped, pkg)
				report(fmt.Sprintf("Skipping (error): %s — %v", pkg, err))
				logger.Warn("Package download error: %s — %v", pkg, err)
			}
			continue
		}

		downloaded = append(downloaded, pkg)
		report(fmt.Sprintf("Downloaded: %s", pkg))
	}

	// Summary
	if len(downloaded) > 0 {
		report(fmt.Sprintf("Downloaded %d package(s): %v", len(downloaded), downloaded))
		logger.Info("Downloaded packages: %v", downloaded)
	}
	if len(skipped) > 0 {
		report(fmt.Sprintf("Skipped %d package(s) (not in repos): %v", len(skipped), skipped))
		logger.Warn("Skipped packages: %v", skipped)
	}

	if len(downloaded) == 0 && len(skipped) > 0 {
		report("Warning: all packages were skipped — no RPMs will be included in the ISO")
	}

	// Generate repo metadata so packages can be installed offline via a yum repo.
	report("Generating repo metadata with createrepo...")
	if err := createRepoMetadata(pkgDir, func(msg string) {
		report(msg)
	}); err != nil {
		return fmt.Errorf("createrepo failed: %w", err)
	}

	if progress != nil {
		progress("packages", "completed",
			fmt.Sprintf("Packages: %d downloaded, %d skipped", len(downloaded), len(skipped)))
	}
	return nil
}

// createRepoMetadata runs createrepo_c (or createrepo fallback) on dir.
func createRepoMetadata(dir string, progress func(string)) error {
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("package directory not accessible: %w", err)
	}

	var tool string
	var args []string
	if _, err := exec.LookPath("createrepo_c"); err == nil {
		tool = "createrepo_c"
		args = []string{dir}
	} else if _, err := exec.LookPath("createrepo"); err == nil {
		tool = "createrepo"
		args = []string{dir}
	} else {
		progress("createrepo not installed, attempting yum install -y createrepo_c...")
		cmd := exec.Command("yum", "install", "-y", "createrepo_c")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("createrepo not installed and yum install failed: %s (%w)", strings.TrimSpace(string(out)), err)
		}
		if _, err := exec.LookPath("createrepo_c"); err == nil {
			tool = "createrepo_c"
			args = []string{dir}
		} else {
			tool = "createrepo"
			args = []string{dir}
		}
	}

	progress(fmt.Sprintf("Running %s on %s", tool, dir))
	cmd := exec.Command(tool, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s (%w)", tool, strings.TrimSpace(string(out)), err)
	}
	return nil
}

// embedISO_md5 embeds an MD5 checksum into the ISO using implantisomd5.
func embedISO_md5(isoPath string, progress func(string)) error {
	progress("Running implantisomd5...")
	cmd := exec.Command("implantisomd5", isoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("implantisomd5 failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// verifyISO_md5 verifies the ISO integrity using checkisomd5.
func verifyISO_md5(isoPath string, progress func(string)) error {
	progress("Running checkisomd5...")
	cmd := exec.Command("checkisomd5", isoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("checkisomd5 failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

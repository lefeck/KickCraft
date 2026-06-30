package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Path defines the directory structure for build operations
type Path struct {
	RootDir string
}

// NewPath constructs a Path instance
func NewPath(rootDir string) *Path {
	return &Path{RootDir: rootDir}
}

// Top-level directories under RootDir
func (p *Path) BuildDir() string    { return filepath.Join(p.RootDir, "build") }
func (p *Path) DownloadDir() string { return filepath.Join(p.RootDir, "download") }
func (p *Path) ConfigDir() string   { return filepath.Join(p.RootDir, "config") }

// Subdirectories under build/
func (p *Path) Mount() string    { return filepath.Join(p.BuildDir(), "mnt") }
func (p *Path) Packages() string { return filepath.Join(p.Mount(), "packages") }
func (p *Path) Scripts() string  { return filepath.Join(p.Mount(), "script") }

// File path helpers

// DownloadFile returns the path of a file inside the download directory
func (p *Path) DownloadFile(fileName string) string {
	return filepath.Join(p.DownloadDir(), fileName)
}

// KickstartFile returns the path where the generated ks.cfg is written
// directly under the ISO build root so it can be loaded via cdrom:/ks.cfg
// (the same convention used by Anaconda's inst.ks=cdrom:/ks.cfg boot param).
func (p *Path) KickstartFile() string {
	return filepath.Join(p.BuildDir(), "ks.cfg")
}

// GrubConfigFile returns the path of a UEFI GRUB config file under the
// extracted ISO tree (e.g. "EFI/BOOT/grub.cfg")
func (p *Path) GrubConfigFile(grub string) string {
	return filepath.Join(p.BuildDir(), grub)
}

// IsolinuxConfigFile returns the path of a BIOS ISOLINUX config file under
// the extracted ISO tree (e.g. "isolinux/isolinux.cfg")
func (p *Path) IsolinuxConfigFile(cfg string) string {
	return filepath.Join(p.BuildDir(), cfg)
}

// IsolinuxBin returns the path of the ISOLINUX boot loader binary
func (p *Path) IsolinuxBin() string {
	return filepath.Join(p.BuildDir(), "isolinux", "isolinux.bin")
}

// ScriptFile returns the path of a file inside the script/ directory
// (e.g. "config.sh" -> build/mnt/script/config.sh)
func (p *Path) ScriptFile(fileName string) string {
	return filepath.Join(p.Scripts(), fileName)
}

// IsolinuxCat returns the path of the ISOLINUX boot catalog
func (p *Path) IsolinuxCat() string {
	return filepath.Join(p.BuildDir(), "isolinux", "boot.cat")
}

// MD5SumFile returns the path of an MD5 checksum file under the build dir
func (p *Path) MD5SumFile(name string) string {
	return filepath.Join(p.BuildDir(), name)
}

// OutputISO returns the path of the generated output ISO
func (p *Path) OutputISO(distro string) string {
	return filepath.Join(p.DownloadDir(), distro+"-ks.iso")
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetAbsPath returns the absolute path for a given path
func GetAbsPath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return abs, nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// CleanPath cleans a file path
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// ReadISOVolumeID extracts the volume label (Volume ID) of an ISO file
// using xorriso's -pvd_info command. Returns the trimmed volume ID
// (without surrounding quotes) or an error if the label cannot be read.
func ReadISOVolumeID(isoPath string) (string, error) {
	if _, err := exec.LookPath("xorriso"); err != nil {
		return "", fmt.Errorf("xorriso not found: %w", err)
	}

	cmd := exec.Command("xorriso", "-indev", isoPath, "-pvd_info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to inspect ISO volume ID: %w: %s", err, strings.TrimSpace(string(output)))
	}

	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "volume id") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}
			volumeID := strings.TrimSpace(parts[1])
			volumeID = strings.Trim(volumeID, "'\"")
			if volumeID != "" {
				return volumeID, nil
			}
		}
	}

	return "", fmt.Errorf("volume ID not found in xorriso output")
}

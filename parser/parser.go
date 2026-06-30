package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/kickcraft/config"
)

// Parser handles Kickstart configuration parsing
type Parser struct {
	lineNum int
	line    string
}

// New creates a new Kickstart parser
func New() *Parser {
	return &Parser{}
}

// Parse parses a Kickstart configuration from a reader
func (p *Parser) Parse(r io.Reader) (*config.KickstartConfig, error) {
	cfg := config.NewKickstartConfig()
	scanner := bufio.NewScanner(r)

	p.lineNum = 0
	var inSection bool
	var currentSection string
	var sectionLines []string

	for scanner.Scan() {
		p.lineNum++
		line := strings.TrimRight(scanner.Text(), "\r")

		// Check for section start
		if strings.HasPrefix(line, "%") {
			// Process previous section if any
			if inSection && currentSection != "" {
				if err := p.processSection(cfg, currentSection, sectionLines); err != nil {
					return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
				}
				// Add section content to raw lines
				for _, sl := range sectionLines {
					cfg.RawLines = append(cfg.RawLines, sl)
				}
			}

			if strings.HasPrefix(line, "%end") {
				inSection = false
				currentSection = ""
				sectionLines = nil
				// Add %end to raw lines
				cfg.RawLines = append(cfg.RawLines, "%end")
				continue
			}

			// Start new section
			inSection = true
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "%"))
			sectionLines = nil
			// Add section header to raw lines
			cfg.RawLines = append(cfg.RawLines, "%"+currentSection)
			continue
		}

		if inSection {
			sectionLines = append(sectionLines, line)
		} else {
			// Skip empty lines and comments for parsing
			if line == "" || strings.HasPrefix(line, "#") {
				// Still add non-empty lines to raw lines
				if line != "" {
					cfg.RawLines = append(cfg.RawLines, line)
				}
				continue
			}
			// Add command to raw lines
			cfg.RawLines = append(cfg.RawLines, line)
			// Parse command for structured access
			if err := p.parseCommand(cfg, line); err != nil {
				return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
			}
		}
	}

	// Process last section if any
	if inSection && currentSection != "" {
		if err := p.processSection(cfg, currentSection, sectionLines); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return cfg, nil
}

// parseCommand parses a single Kickstart command
func (p *Parser) parseCommand(cfg *config.KickstartConfig, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	parts := splitCommand(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "network":
		return p.parseNetwork(cfg, args)
	case "lang":
		return p.parseLang(cfg, args)
	case "keyboard":
		return p.parseKeyboard(cfg, args)
	case "timezone":
		return p.parseTimezone(cfg, args)
	case "rootpw":
		return p.parseRootPw(cfg, args)
	case "user":
		return p.parseUser(cfg, args)
	case "authconfig":
		return p.parseAuthconfig(cfg, args)
	case "bootloader":
		return p.parseBootloader(cfg, args)
	case "firewall":
		return p.parseFirewall(cfg, args)
	case "selinux":
		return p.parseSELinux(cfg, args)
	case "services":
		return p.parseServices(cfg, args)
	case "repo":
		return p.parseRepo(cfg, args)
	case "cdrom":
		cfg.Method.Type = "cdrom"
		return nil
	case "nfs":
		return p.parseNFS(cfg, args)
	case "url":
		return p.parseURL(cfg, args)
	case "harddrive":
		return p.parseHarddrive(cfg, args)
	case "part", "partition":
		return p.parsePart(cfg, args)
	case "raid":
		return p.parseRaid(cfg, args)
	case "volgroup":
		return p.parseVolGroup(cfg, args)
	case "logvol":
		return p.parseLogVol(cfg, args)
	case "zerombr":
		cfg.Storage.Zerombr = true
	case "clearpart":
		return p.parseClearPart(cfg, args)
	case "ignoredisk":
		return p.parseIgnoredisk(cfg, args)
	case "autopart":
		cfg.Storage.AutoPart = true
		// Parse --type=<type> argument
		for _, arg := range args {
			if strings.HasPrefix(arg, "--type=") {
				cfg.Storage.AutoPartType = strings.TrimPrefix(arg, "--type=")
				break
			}
		}
		return nil
	case "skipx", "xconfig":
		return p.parseXConfig(cfg, cmd, args)
	case "firstboot":
		return p.parseFirstBoot(cfg, args)
	case "pwpolicy":
		return p.parsePwPolicy(cfg, args)
	case "eula":
		return p.parseEula(cfg, args)
	case "realm":
		return p.parseRealm(cfg, args)
	case "text":
		cfg.InstallMode = "text"
		return nil
	case "graphical":
		cfg.InstallMode = "graphical"
		return nil
	case "cmdline":
		cfg.InstallMode = "cmdline"
		return nil
	case "reboot", "poweroff", "shutdown", "halt":
		cfg.PowerAction = cmd
		return nil
	case "sshkey":
		return p.parseSSHKey(cfg, args)
	case "include":
		// %include path - store for later use after %pre
		if len(args) > 0 {
			cfg.PreInclude = args[0]
		}
		return nil
	default:
		// Unknown command - store as custom for later
		cfg.AddCustomCommand(cmd, strings.Join(parts, " "))
	}

	return nil
}

// splitCommand splits a command line into parts, handling quoted strings
func splitCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		c := line[i]

		if !inQuote && (c == '"' || c == '\'') {
			inQuote = true
			quoteChar = c
			continue
		}

		if inQuote && c == quoteChar {
			inQuote = false
			continue
		}

		if !inQuote && c == ' ' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// processSection processes a %section block
func (p *Parser) processSection(cfg *config.KickstartConfig, section string, lines []string) error {
	section = strings.TrimSpace(section)

	switch {
	case strings.HasPrefix(section, "packages") || section == "packages":
		return p.processPackages(cfg, lines)
	case strings.HasPrefix(section, "pre"):
		return p.processPreScript(cfg, section, lines)
	case strings.HasPrefix(section, "post"):
		// Check for nochroot BEFORE general post check
		if strings.Contains(section, "--nochroot") || strings.HasPrefix(section, "post --nochroot") {
			return p.processPostScriptNoChroot(cfg, section, lines)
		}
		return p.processPostScript(cfg, section, lines)
	case strings.HasPrefix(section, "onerror"):
		cfg.OnError = strings.Join(lines, "\n")
	default:
		cfg.AddCustomSection(section, strings.Join(lines, "\n"))
	}

	return nil
}

// --- URL Install Source ---

// --- Network Configuration ---

func (p *Parser) parseNetwork(cfg *config.KickstartConfig, args []string) error {
	net := config.NetworkConfig{}
	hasDevice := false

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--device="):
			net.Device = strings.TrimPrefix(arg, "--device=")
			hasDevice = true
		case strings.HasPrefix(arg, "--bootproto="):
			net.BootProto = strings.TrimPrefix(arg, "--bootproto=")
		case strings.HasPrefix(arg, "--ip="):
			net.IP = strings.TrimPrefix(arg, "--ip=")
		case strings.HasPrefix(arg, "--gateway="):
			net.Gateway = strings.TrimPrefix(arg, "--gateway=")
		case strings.HasPrefix(arg, "--nameserver="):
			net.Nameserver = strings.TrimPrefix(arg, "--nameserver=")
		case strings.HasPrefix(arg, "--netmask="):
			net.Netmask = strings.TrimPrefix(arg, "--netmask=")
		case strings.HasPrefix(arg, "--hostname="):
			// Store hostname in Locale (for network without device)
			cfg.Locale.Hostname = strings.TrimPrefix(arg, "--hostname=")
		case strings.HasPrefix(arg, "--onboot="):
			net.OnBoot = strings.TrimPrefix(arg, "--onboot=") == "yes"
		case strings.HasPrefix(arg, "--ipv6="):
			net.IPv6 = strings.TrimPrefix(arg, "--ipv6=")
		case strings.HasPrefix(arg, "--noipv6"):
			net.NoIPv6 = true
		case arg == "--activate":
			net.Activate = true
		case strings.HasPrefix(arg, "--mtu="):
			net.MTU = strings.TrimPrefix(arg, "--mtu=")
		}
	}

	// Only add to Networks if it has a device
	if hasDevice {
		cfg.Networks = append(cfg.Networks, net)
	}
	return nil
}

// --- Language ---

func (p *Parser) parseLang(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--addsupport=") {
			cfg.Locale.AddSupport = strings.TrimPrefix(arg, "--addsupport=")
		} else if !strings.HasPrefix(arg, "--") {
			cfg.Locale.Lang = arg
		}
	}
	return nil
}

// --- Keyboard ---

func (p *Parser) parseKeyboard(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--vckeymap=") {
			cfg.Locale.Keymap = strings.TrimPrefix(arg, "--vckeymap=")
		}
		if strings.HasPrefix(arg, "--xlayouts=") {
			cfg.Locale.XLayouts = strings.TrimPrefix(arg, "--xlayouts=")
		}
	}
	return nil
}

// --- Timezone ---

func (p *Parser) parseTimezone(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--utc") {
			cfg.Locale.UTC = true
		} else if strings.HasPrefix(arg, "--timezone=") {
			cfg.Locale.Timezone = strings.TrimPrefix(arg, "--timezone=")
		} else if strings.HasPrefix(arg, "--nontp") {
			cfg.Locale.NoNTP = true
		} else if strings.HasPrefix(arg, "--ntpservers=") {
			cfg.Locale.NTPServers = strings.TrimPrefix(arg, "--ntpservers=")
		} else if !strings.HasPrefix(arg, "--") {
			// Positional argument: timezone name (e.g., Asia/Shanghai)
			if cfg.Locale.Timezone == "" {
				cfg.Locale.Timezone = arg
			}
		}
	}
	return nil
}

// --- NFS Installation ---

func (p *Parser) parseNFS(cfg *config.KickstartConfig, args []string) error {
	cfg.Method.Type = "nfs"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--server=") {
			cfg.Method.Server = strings.TrimPrefix(arg, "--server=")
		} else if strings.HasPrefix(arg, "--dir=") {
			cfg.Method.Dir = strings.TrimPrefix(arg, "--dir=")
		} else if strings.HasPrefix(arg, "--opts=") {
			cfg.Method.Opts = strings.TrimPrefix(arg, "--opts=")
		}
	}
	return nil
}

// --- URL Installation ---

func (p *Parser) parseURL(cfg *config.KickstartConfig, args []string) error {
	cfg.Method.Type = "url"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--url=") {
			cfg.Method.URL = strings.TrimPrefix(arg, "--url=")
		} else if arg == "--noverifyssl" {
			cfg.Method.NoSSL = true
		} else if strings.HasPrefix(arg, "--proxy=") {
			cfg.Method.Proxy = strings.TrimPrefix(arg, "--proxy=")
		}
	}
	return nil
}

// --- Hard Drive Installation ---

func (p *Parser) parseHarddrive(cfg *config.KickstartConfig, args []string) error {
	cfg.Method.Type = "harddrive"

	for _, arg := range args {
		if strings.HasPrefix(arg, "--partition=") {
			cfg.Method.Partition = strings.TrimPrefix(arg, "--partition=")
		} else if strings.HasPrefix(arg, "--dir=") {
			cfg.Method.Dir = strings.TrimPrefix(arg, "--dir=")
		}
	}
	return nil
}

// --- Root Password ---

func (p *Parser) parseRootPw(cfg *config.KickstartConfig, args []string) error {
	cfg.RootPassword.IsSet = true

	for i, arg := range args {
		if arg == "--plaintext" {
			cfg.RootPassword.IsCrypted = false
		} else if strings.HasPrefix(arg, "--plaintext=") {
			cfg.RootPassword.IsCrypted = false
			cfg.RootPassword.Password = strings.TrimPrefix(arg, "--plaintext=")
		} else if arg == "--iscrypted" {
			cfg.RootPassword.IsCrypted = true
			// Next arg might be the password value
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				cfg.RootPassword.Password = args[i+1]
			}
		} else if strings.HasPrefix(arg, "--iscrypted=") {
			cfg.RootPassword.IsCrypted = true
			cfg.RootPassword.Password = strings.TrimPrefix(arg, "--iscrypted=")
		} else if arg == "--lock" {
			cfg.RootPassword.Lock = true
		} else if arg == "--allow-ssh" {
			cfg.RootPassword.AllowSsh = true
		} else if !strings.HasPrefix(arg, "--") {
			// Plain password argument (no flag prefix)
			if cfg.RootPassword.Password == "" {
				cfg.RootPassword.Password = arg
			}
		}
	}
	return nil
}

// --- User ---

func (p *Parser) parseUser(cfg *config.KickstartConfig, args []string) error {
	user := config.UserConfig{}

	for i, arg := range args {
		if strings.HasPrefix(arg, "--name=") {
			user.Name = strings.TrimPrefix(arg, "--name=")
		}
		if strings.HasPrefix(arg, "--password=") {
			user.Password = strings.TrimPrefix(arg, "--password=")
			user.IsPlaintext = false
			user.IsCrypted = true
		}
		if strings.HasPrefix(arg, "--plaintext=") {
			user.Password = strings.TrimPrefix(arg, "--plaintext=")
			user.IsPlaintext = true
		}
		if arg == "--plaintext" {
			user.IsPlaintext = true
			// Password might be in next arg
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				user.Password = args[i+1]
			}
		}
		if arg == "--iscrypted" {
			user.IsCrypted = true
			user.IsPlaintext = false
		}
		if strings.HasPrefix(arg, "--gecos=") {
			user.Gecos = strings.TrimPrefix(arg, "--gecos=")
		}
		if strings.HasPrefix(arg, "--groups=") {
			groupsStr := strings.TrimPrefix(arg, "--groups=")
			user.Groups = strings.Split(groupsStr, ",")
		}
		if strings.HasPrefix(arg, "--homedir=") {
			user.HomeDir = strings.TrimPrefix(arg, "--homedir=")
		}
		if strings.HasPrefix(arg, "--shell=") {
			user.Shell = strings.TrimPrefix(arg, "--shell=")
		}
		if strings.HasPrefix(arg, "--uid=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--uid="), "%d", &user.UID)
		}
		if strings.HasPrefix(arg, "--gid=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--gid="), "%d", &user.GID)
		}
		if arg == "--lock" {
			user.Lock = true
		}
		if strings.HasPrefix(arg, "--sshkey=") {
			key := strings.TrimPrefix(arg, "--sshkey=")
			user.SSHKeys = append(user.SSHKeys, key)
		}
	}

	cfg.Users = append(cfg.Users, user)
	return nil
}

// --- Authconfig ---

func (p *Parser) parseAuthconfig(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--enableshadow" || arg == "--enable shadow" {
			cfg.Auth.EnableShadow = true
		}
		if strings.HasPrefix(arg, "--passalgo=") {
			cfg.Auth.PasswordAlgorithm = strings.TrimPrefix(arg, "--passalgo=")
		}
	}
	return nil
}

// --- Bootloader ---

func (p *Parser) parseBootloader(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--location=") {
			cfg.Bootloader.Location = strings.TrimPrefix(arg, "--location=")
		}
		if strings.HasPrefix(arg, "--append=") {
			cfg.Bootloader.Append = strings.TrimPrefix(arg, "--append=")
		}
		if strings.HasPrefix(arg, "--boot-drive=") {
			cfg.Bootloader.BootDrive = strings.TrimPrefix(arg, "--boot-drive=")
		}
		if arg == "--driveorder" {
			// Next args will be drive order
		}
	}
	return nil
}

// --- Firewall ---

func (p *Parser) parseFirewall(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--enabled" || arg == "--enable" {
			cfg.Firewall.Enabled = true
		}
		if arg == "--disabled" || arg == "--disable" {
			cfg.Firewall.Enabled = false
		}
		if strings.HasPrefix(arg, "--service=") {
			cfg.Firewall.Services = append(cfg.Firewall.Services, strings.TrimPrefix(arg, "--service="))
		}
		if strings.HasPrefix(arg, "--port=") {
			cfg.Firewall.Ports = append(cfg.Firewall.Ports, strings.TrimPrefix(arg, "--port="))
		}
	}
	return nil
}

// --- SELinux ---

func (p *Parser) parseSELinux(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--enforcing" {
			cfg.SELinux.Mode = "enforcing"
		}
		if arg == "--permissive" {
			cfg.SELinux.Mode = "permissive"
		}
		if arg == "--disabled" {
			cfg.SELinux.Mode = "disabled"
		}
	}
	return nil
}

// --- Services ---

func (p *Parser) parseServices(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--enabled=") {
			cfg.Services.Enabled = strings.Split(strings.TrimPrefix(arg, "--enabled="), ",")
		}
		if strings.HasPrefix(arg, "--disabled=") {
			cfg.Services.Disabled = strings.Split(strings.TrimPrefix(arg, "--disabled="), ",")
		}
	}
	return nil
}

// --- Repo ---

func (p *Parser) parseRepo(cfg *config.KickstartConfig, args []string) error {
	repo := config.RepoConfig{}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--name=") {
			repo.Name = strings.TrimPrefix(arg, "--name=")
		}
		if strings.HasPrefix(arg, "--baseurl=") {
			repo.BaseURL = strings.TrimPrefix(arg, "--baseurl=")
		}
		if strings.HasPrefix(arg, "--mirrorlist=") {
			repo.MirrorList = strings.TrimPrefix(arg, "--mirrorlist=")
		}
		if strings.HasPrefix(arg, "--cost=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--cost="), "%d", &repo.Cost)
		}
	}

	cfg.Repos = append(cfg.Repos, repo)
	return nil
}

// --- Partition ---

func (p *Parser) parsePart(cfg *config.KickstartConfig, args []string) error {
	part := config.PartConfig{}

	// First positional arg is the mountpoint (e.g., "part /boot")
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		part.Mountpoint = args[0]
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--fstype=") {
			part.FSType = strings.TrimPrefix(arg, "--fstype=")
		}
		if strings.HasPrefix(arg, "--size=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--size="), "%d", &part.Size)
		}
		if strings.HasPrefix(arg, "--grow") {
			part.Grow = true
		}
		if strings.HasPrefix(arg, "--maxsize=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--maxsize="), "%d", &part.MaxSize)
		}
		if strings.HasPrefix(arg, "--ondisk=") {
			part.OnDisk = strings.TrimPrefix(arg, "--ondisk=")
		}
		if strings.HasPrefix(arg, "--asprimary") {
			part.AsPrimary = true
		}
		if strings.HasPrefix(arg, "--mountpoint=") {
			part.Mountpoint = strings.TrimPrefix(arg, "--mountpoint=")
		}
	}

	cfg.Storage.Partitions = append(cfg.Storage.Partitions, part)
	return nil
}

// --- RAID ---

func (p *Parser) parseRaid(cfg *config.KickstartConfig, args []string) error {
	raid := config.RaidConfig{}

	// First positional arg is the mountpoint (e.g., "raid / --level=1")
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		raid.Mountpoint = args[0]
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--level=") {
			raid.Level = strings.TrimPrefix(arg, "--level=")
		}
		if strings.HasPrefix(arg, "--device=") {
			raid.Device = strings.TrimPrefix(arg, "--device=")
		}
		if strings.HasPrefix(arg, "--fstype=") {
			raid.FSType = strings.TrimPrefix(arg, "--fstype=")
		}
		if strings.HasPrefix(arg, "--mountpoint=") {
			raid.Mountpoint = strings.TrimPrefix(arg, "--mountpoint=")
		}
		// Devices are captured from remaining args
	}

	// Capture remaining args as devices
	seenFlags := map[string]bool{
		"--level": true, "--device": true, "--fstype": true, "--mountpoint": true,
		"--spares": true, "--useexisting": true, "--noformat": true,
	}
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		prefix := parts[0]
		if !seenFlags[prefix] && !strings.HasPrefix(arg, "--") {
			raid.Devices = append(raid.Devices, arg)
		}
	}

	cfg.Storage.Raids = append(cfg.Storage.Raids, raid)
	return nil
}

// --- Volume Group ---

func (p *Parser) parseVolGroup(cfg *config.KickstartConfig, args []string) error {
	vg := config.VolGroupConfig{}

	knownFlags := map[string]bool{"--name": true, "--pesize": true, "--useexisting": true, "--noformat": true}
	nameCaptured := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--name=") {
			vg.Name = strings.TrimPrefix(arg, "--name=")
			nameCaptured = true
		} else if strings.HasPrefix(arg, "--pesize=") {
			vg.PESize = strings.TrimPrefix(arg, "--pesize=")
		} else if !strings.HasPrefix(arg, "--") {
			if !nameCaptured {
				vg.Name = arg
				nameCaptured = true
				continue
			}
			vg.PhysicalVolumes = append(vg.PhysicalVolumes, arg)
		} else {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 && !knownFlags[parts[0]] {
				vg.PhysicalVolumes = append(vg.PhysicalVolumes, parts[1])
			}
		}
	}

	cfg.Storage.VolGroups = append(cfg.Storage.VolGroups, vg)
	return nil
}

// --- Logical Volume ---

func (p *Parser) parseLogVol(cfg *config.KickstartConfig, args []string) error {
	lv := config.LogVolConfig{}

	// First positional arg is the mountpoint (e.g., "logvol / --vgname=vg0")
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		lv.Mountpoint = args[0]
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--vgname=") {
			lv.VGName = strings.TrimPrefix(arg, "--vgname=")
		}
		if strings.HasPrefix(arg, "--name=") {
			lv.Name = strings.TrimPrefix(arg, "--name=")
		}
		if strings.HasPrefix(arg, "--size=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--size="), "%d", &lv.Size)
		}
		if strings.HasPrefix(arg, "--grow") {
			lv.Grow = true
		}
		if strings.HasPrefix(arg, "--maxsize=") {
			fmt.Sscanf(strings.TrimPrefix(arg, "--maxsize="), "%d", &lv.MaxSize)
		}
		if strings.HasPrefix(arg, "--fstype=") {
			lv.FSType = strings.TrimPrefix(arg, "--fstype=")
		}
		if strings.HasPrefix(arg, "--mountpoint=") {
			lv.Mountpoint = strings.TrimPrefix(arg, "--mountpoint=")
		}
	}

	cfg.Storage.LogVols = append(cfg.Storage.LogVols, lv)
	return nil
}

// --- Clearpart ---

func (p *Parser) parseClearPart(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--all" {
			cfg.Storage.ClearAll = true
		}
		if arg == "--linux" {
			cfg.Storage.ClearLinux = true
		}
		if strings.HasPrefix(arg, "--drives=") {
			cfg.Storage.ClearDrives = strings.Split(strings.TrimPrefix(arg, "--drives="), ",")
		}
		if arg == "--initlabel" {
			cfg.Storage.InitLabel = true
		}
	}
	return nil
}

// --- Ignoredisk ---

func (p *Parser) parseIgnoredisk(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--interactive" {
			cfg.Storage.IgnoreDiskInteractive = true
		}
		if strings.HasPrefix(arg, "--drives=") {
			cfg.Storage.IgnoreDiskDrives = strings.Split(strings.TrimPrefix(arg, "--drives="), ",")
		}
		if arg == "--only-use" {
			cfg.Storage.IgnoreDiskOnlyUse = true
		}
		if strings.HasPrefix(arg, "--only-use=") {
			cfg.Storage.IgnoreDiskOnlyUse = true
			cfg.Storage.OnlyUseDrives = strings.Split(strings.TrimPrefix(arg, "--only-use="), ",")
		}
	}
	return nil
}

// --- XConfig (firstboot, skipx) ---

func (p *Parser) parseXConfig(cfg *config.KickstartConfig, cmd string, args []string) error {
	switch cmd {
	case "skipx":
		cfg.Graphics.SkipX = true
	case "firstboot":
		for _, arg := range args {
			if arg == "--disabled" || arg == "--disable" {
				cfg.Graphics.FirstBoot = "disabled"
			}
			if arg == "--enabled" || arg == "--enable" {
				cfg.Graphics.FirstBoot = "enabled"
			}
			if arg == "--reconfig" {
				cfg.Graphics.FirstBoot = "reconfig"
			}
		}
	}
	return nil
}

// --- Firstboot ---

func (p *Parser) parseFirstBoot(cfg *config.KickstartConfig, args []string) error {
	return p.parseXConfig(cfg, "firstboot", args)
}

// --- PwPolicy ---

func (p *Parser) parsePwPolicy(cfg *config.KickstartConfig, args []string) error {
	// Store as custom for now
	cfg.AddCustomCommand("pwpolicy", strings.Join(args, " "))
	return nil
}

// --- EULA ---

func (p *Parser) parseEula(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if arg == "--agreed" {
			cfg.EULA = "agreed"
		}
	}
	return nil
}

// --- Realm ---

func (p *Parser) parseRealm(cfg *config.KickstartConfig, args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--join=") {
			cfg.Realm.Join = strings.TrimPrefix(arg, "--join=")
		}
	}
	return nil
}

// --- SSH Key ---

func (p *Parser) parseSSHKey(cfg *config.KickstartConfig, args []string) error {
	if len(args) >= 1 {
		parts := strings.SplitN(args[0], ":", 2)
		if len(parts) == 2 {
			cfg.SSHKeys = append(cfg.SSHKeys, config.SSHKeyConfig{
				Username: parts[0],
				Key:      parts[1],
			})
		}
	}
	return nil
}

// --- Packages Section ---

func (p *Parser) processPackages(cfg *config.KickstartConfig, lines []string) error {
	cfg.Packages = config.PackagesConfig{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section options
		if strings.HasPrefix(line, "--") {
			if strings.HasPrefix(line, "--inst-langs=") {
				cfg.Packages.Languages = strings.Split(strings.TrimPrefix(line, "--inst-langs="), ",")
			}
			if line == "--default" {
				cfg.Packages.Default = true
			}
			if line == "--nobase" {
				cfg.Packages.NoBase = true
			}
			if strings.HasPrefix(line, "--excludedocs") {
				cfg.Packages.ExcludeDocs = true
			}
			if strings.HasPrefix(line, "--ignoremissing") {
				cfg.Packages.IgnoreMissing = true
			}
			continue
		}

		// Package or group
		if strings.HasPrefix(line, "@") {
			group := strings.TrimPrefix(line, "@")
			isOptional := !strings.HasPrefix(group, "^")
			group = strings.TrimPrefix(group, "^")
			cfg.Packages.Groups = append(cfg.Packages.Groups, config.PackageGroup{
				Name:     group,
				Optional: isOptional,
			})
		} else {
			cfg.Packages.Packages = append(cfg.Packages.Packages, line)
		}
	}

	return nil
}

// --- Pre Script ---

func (p *Parser) processPreScript(cfg *config.KickstartConfig, section string, lines []string) error {
	script := config.ScriptConfig{
		Type: "pre",
	}

	// Parse section options
	sectionParts := strings.Fields(section)
	for _, part := range sectionParts[1:] {
		if strings.HasPrefix(part, "--interpreter=") {
			script.Interpreter = strings.TrimPrefix(part, "--interpreter=")
		}
		if part == "--erroronfail" {
			script.ErrorOnFail = true
		}
		if part == "--log=" {
			// Log file specified
		}
	}

	script.Content = strings.Join(lines, "\n")
	cfg.PreScripts = append(cfg.PreScripts, script)
	return nil
}

// --- Post Script ---

func (p *Parser) processPostScript(cfg *config.KickstartConfig, section string, lines []string) error {
	script := config.ScriptConfig{
		Type: "post",
	}

	// Parse section options
	sectionParts := strings.Fields(section)
	for _, part := range sectionParts[1:] {
		if strings.HasPrefix(part, "--interpreter=") {
			script.Interpreter = strings.TrimPrefix(part, "--interpreter=")
		}
		if part == "--nochroot" {
			script.NoChroot = true
		}
		if part == "--erroronfail" {
			script.ErrorOnFail = true
		}
	}

	script.Content = strings.Join(lines, "\n")
	cfg.PostScripts = append(cfg.PostScripts, script)
	return nil
}

// --- Post Script No Chroot ---

func (p *Parser) processPostScriptNoChroot(cfg *config.KickstartConfig, section string, lines []string) error {
	script := config.ScriptConfig{
		Type:     "post",
		NoChroot: true,
	}

	// Parse section options
	sectionParts := strings.Fields(section)
	for _, part := range sectionParts[1:] {
		if strings.HasPrefix(part, "--interpreter=") {
			script.Interpreter = strings.TrimPrefix(part, "--interpreter=")
		}
		if part == "--erroronfail" {
			script.ErrorOnFail = true
		}
	}

	script.Content = strings.Join(lines, "\n")
	cfg.PostScriptsNoChroot = append(cfg.PostScriptsNoChroot, script)
	return nil
}

// ParseFromString parses Kickstart configuration from a string
func ParseFromString(s string) (*config.KickstartConfig, error) {
	return New().Parse(strings.NewReader(s))
}

// ParseFromFile parses Kickstart configuration from a file
func ParseFromFile(path string) (*config.KickstartConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return New().Parse(file)
}

// Compile regex patterns
var (
	argRegex = regexp.MustCompile(`--([a-zA-Z0-9-]+)(?:=(.*))?`)
)

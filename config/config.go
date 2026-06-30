package config

import (
	"fmt"
	"strings"
)

// ServerOptions holds server configuration
type ServerOptions struct {
	Port         string
	StaticDir    string
	TemplatesDir string
}

// KickstartConfig represents a complete Kickstart configuration
type KickstartConfig struct {
	// Basic settings
	Locale       LocaleConfig   `json:"locale"`
	RootPassword RootPwConfig   `json:"rootPassword"`
	Users        []UserConfig   `json:"users"`
	SSHKeys      []SSHKeyConfig `json:"sshKeys"`

	// Installation method (cdrom, nfs, url, harddrive, etc.)
	Method InstallMethodConfig `json:"method"`

	// Installation mode (graphical, text, cmdline)
	InstallMode string `json:"installMode"`

	// Boot and storage
	Bootloader BootloaderConfig `json:"bootloader"`
	Storage    StorageConfig    `json:"storage"`
	Graphics   GraphicsConfig   `json:"graphics"`

	// Network and security
	Networks []NetworkConfig `json:"networks"`
	Firewall FirewallConfig  `json:"firewall"`
	SELinux  SELinuxConfig   `json:"selinux"`
	Auth     AuthConfig      `json:"auth"`
	Realm    RealmConfig     `json:"realm"`

	// Packages and repos
	Packages           PackagesConfig `json:"packages"`
	Repos              []RepoConfig   `json:"repos"`
	AdditionalPackages []string       `json:"additionalPackages"` // Extra RPMs downloaded to build/mnt/packages/

	// Scripts
	PreScripts          []ScriptConfig `json:"preScripts"`
	PostScripts         []ScriptConfig `json:"postScripts"`
	PostScriptsNoChroot []ScriptConfig `json:"postScriptsNoChroot"`
	PreInclude          string         `json:"preInclude"` // Path for %include after %pre
	OnError             string         `json:"onError"`

	// Other settings
	EULA        string            `json:"eula"`
	Services    ServicesConfig    `json:"services"`
	PowerAction string            `json:"powerAction"` // reboot, poweroff, shutdown, halt
	FirstBoot   string            `json:"firstBoot"`   // enabled, disabled, reconfig
	CustomCmds  map[string]string `json:"customCommands"`
	CustomSecs  map[string]string `json:"customSections"`
	Kdump       KdumpConfig       `json:"kdump"`
	// Preserve original line order for non-section commands
	RawLines []string `json:"rawLines"`
}

// NewKickstartConfig creates a new default configuration
func NewKickstartConfig() *KickstartConfig {
	return &KickstartConfig{
		Locale:      NewLocaleConfig(),
		Method:      NewInstallMethodConfig(),
		Bootloader:  NewBootloaderConfig(),
		Storage:     NewStorageConfig(),
		Graphics:    NewGraphicsConfig(),
		Firewall:    NewFirewallConfig(),
		SELinux:     NewSELinuxConfig(),
		Auth:        NewAuthConfig(),
		Realm:       NewRealmConfig(),
		Services:    NewServicesConfig(),
		Kdump:       NewKdumpConfig(),
		Packages:    PackagesConfig{},
		Repos:       []RepoConfig{},
		Users:       []UserConfig{},
		Networks:    []NetworkConfig{},
		SSHKeys:     []SSHKeyConfig{},
		PreScripts:  []ScriptConfig{},
		PostScripts: []ScriptConfig{},
		CustomCmds:  make(map[string]string),
		CustomSecs:  make(map[string]string),
	}
}

// AddCustomCommand adds a custom command
func (c *KickstartConfig) AddCustomCommand(name, value string) {
	if c.CustomCmds == nil {
		c.CustomCmds = make(map[string]string)
	}
	c.CustomCmds[name] = value
}

// AddCustomSection adds a custom section
func (c *KickstartConfig) AddCustomSection(name, content string) {
	if c.CustomSecs == nil {
		c.CustomSecs = make(map[string]string)
	}
	c.CustomSecs[name] = content
}

// --- Locale Configuration ---

type LocaleConfig struct {
	Lang       string `json:"lang"`
	Keymap     string `json:"keymap,omitempty"`
	XLayouts   string `json:"xlayouts,omitempty"`
	AddSupport string `json:"addSupport,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
	UTC        bool   `json:"utc"`
	NoNTP      bool   `json:"noNtp"`
	NTPServers string `json:"ntpServers,omitempty"`
	Hostname   string `json:"hostname,omitempty"`
}

func NewLocaleConfig() LocaleConfig {
	return LocaleConfig{
		Lang: "en_US.UTF-8",
		UTC:  true,
	}
}

// --- Installation Method Configuration ---

// InstallMethodConfig represents the installation method (url, cdrom, nfs, harddrive, etc.)
type InstallMethodConfig struct {
	Type        string `json:"type"` // cdrom, nfs, url, harddrive
	Server      string `json:"server,omitempty"`
	Dir         string `json:"dir,omitempty"`
	URL         string `json:"url,omitempty"`
	Device      string `json:"device,omitempty"`
	Partition   string `json:"partition,omitempty"`
	BIOSDrive   string `json:"biosDrive,omitempty"`
	Opts        string `json:"opts,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	NoSSL       bool   `json:"noSSL"`
	Proxy       string `json:"proxy,omitempty"`
	Excluded    string `json:"excluded,omitempty"`
	Included    string `json:"included,omitempty"`
	InstallRepo string `json:"installRepo,omitempty"`
}

func NewInstallMethodConfig() InstallMethodConfig {
	return InstallMethodConfig{
		Type: "cdrom", // Default to cdrom
	}
}

// --- Root Password ---

type RootPwConfig struct {
	Password  string `json:"password,omitempty"`
	IsCrypted bool   `json:"isCrypted"`
	Lock      bool   `json:"lock"`
	AllowSsh  bool   `json:"allowSsh"`
	IsSet     bool   `json:"isSet"`
}

// --- User Configuration ---

type UserConfig struct {
	Name        string   `json:"name"`
	Password    string   `json:"password,omitempty"`
	UID         int      `json:"uid"`
	GID         int      `json:"gid"`
	HomeDir     string   `json:"homeDir,omitempty"`
	Shell       string   `json:"shell,omitempty"`
	Lock        bool     `json:"lock"`
	IsPlaintext bool     `json:"isPlaintext"`
	IsCrypted   bool     `json:"isCrypted"`
	Gecos       string   `json:"gecos,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	SSHKeys     []string `json:"sshKeys,omitempty"`
}

// --- SSH Key ---

type SSHKeyConfig struct {
	Username string `json:"username"`
	Key      string `json:"key"`
}

// --- Bootloader Configuration ---

type BootloaderConfig struct {
	Location   string   `json:"location,omitempty"` // mbr, partition, none
	Append     string   `json:"append,omitempty"`
	BootDrive  string   `json:"bootDrive,omitempty"`
	DriveOrder []string `json:"driveOrder,omitempty"`
}

func NewBootloaderConfig() BootloaderConfig {
	return BootloaderConfig{
		Location: "mbr",
	}
}

// --- Storage Configuration ---

type StorageConfig struct {
	// Disk management
	Zerombr               bool     `json:"zerombr"`
	ClearAll              bool     `json:"clearAll"`
	ClearLinux            bool     `json:"clearLinux"`
	ClearDrives           []string `json:"clearDrives,omitempty"`
	InitLabel             bool     `json:"initLabel"`
	IgnoreDiskDrives      []string `json:"ignoreDiskDrives,omitempty"`
	OnlyUseDrives         []string `json:"onlyUseDrives,omitempty"`
	IgnoreDiskInteractive bool     `json:"ignoreDiskInteractive"`
	IgnoreDiskOnlyUse     bool     `json:"ignoreDiskOnlyUse"`

	// Auto partitioning
	AutoPart     bool   `json:"autopart"`     // Enable autopart
	AutoPartType string `json:"autopartType"` // lvm, plain, thinp, btrfs

	// Btrfs
	Btrfs []BtrfsConfig `json:"btrfs"`

	// Partitioning
	Partitions []PartConfig     `json:"partitions"`
	Raids      []RaidConfig     `json:"raids"`
	VolGroups  []VolGroupConfig `json:"volGroups"`
	LogVols    []LogVolConfig   `json:"logVols"`
}

func NewStorageConfig() StorageConfig {
	return StorageConfig{
		Partitions: []PartConfig{},
		Raids:      []RaidConfig{},
		VolGroups:  []VolGroupConfig{},
		LogVols:    []LogVolConfig{},
		Btrfs:      []BtrfsConfig{},
	}
}

// Btrfs configuration
type BtrfsConfig struct {
	Subvol    string   `json:"subvol"`    // Mountpoint or "none" for metadata
	Name      string   `json:"name"`      // Subvol name
	Level     string   `json:"level"`     // Data level (raid0, raid1, raid10, dup)
	MetaLevel string   `json:"metaLevel"` // Metadata level
	Label     string   `json:"label"`     // Filesystem label
	Devices   []string `json:"devices"`   // Devices to use
}

// Partition configuration
type PartConfig struct {
	Mountpoint string `json:"mountpoint,omitempty"`
	FSType     string `json:"fstype,omitempty"`
	Size       int    `json:"size"`
	MaxSize    int    `json:"maxSize"`
	Grow       bool   `json:"grow"`
	OnDisk     string `json:"ondisk,omitempty"`
	AsPrimary  bool   `json:"asPrimary"`
	Encrypted  bool   `json:"encrypted"`
	Passphrase string `json:"passphrase,omitempty"`
}

// RAID configuration
type RaidConfig struct {
	Level      string   `json:"level"`
	Device     string   `json:"device"`
	FSType     string   `json:"fstype,omitempty"`
	Mountpoint string   `json:"mountpoint,omitempty"`
	Spares     int      `json:"spares"`
	Devices    []string `json:"devices"`
	Encrypted  bool     `json:"encrypted"`
	Passphrase string   `json:"passphrase,omitempty"`
}

// Volume group configuration
type VolGroupConfig struct {
	Name            string   `json:"name"`
	PESize          string   `json:"pesize,omitempty"`
	PhysicalVolumes []string `json:"physicalVolumes"`
}

// Logical volume configuration
type LogVolConfig struct {
	VGName     string `json:"vgname"`
	Name       string `json:"name"`
	Size       int    `json:"size"`
	MaxSize    int    `json:"maxSize"`
	Grow       bool   `json:"grow"`
	FSType     string `json:"fstype,omitempty"`
	Mountpoint string `json:"mountpoint,omitempty"`
	Encrypted  bool   `json:"encrypted"`
	Passphrase string `json:"passphrase,omitempty"`
}

// --- Graphics Configuration ---

type GraphicsConfig struct {
	SkipX     bool   `json:"skipX"`
	FirstBoot string `json:"firstBoot"` // enabled, disabled, reconfig
}

func NewGraphicsConfig() GraphicsConfig {
	return GraphicsConfig{
		FirstBoot: "enabled",
	}
}

// --- Network Configuration ---

type NetworkConfig struct {
	Device            string `json:"device"`
	BootProto         string `json:"bootProto"`
	Ethtool           string `json:"ethtool,omitempty"`
	Gateway           string `json:"gateway,omitempty"`
	Hostname          string `json:"hostname,omitempty"`
	IP                string `json:"ip,omitempty"`
	MTU               string `json:"mtu,omitempty"`
	Nameserver        string `json:"nameserver,omitempty"`
	Netmask           string `json:"netmask,omitempty"`
	NoDNS             bool   `json:"noDns"`
	OnBoot            bool   `json:"onBoot"`
	NoIPv4            bool   `json:"noIpv4"`
	NoIPv6            bool   `json:"noIpv6"`
	IPv6              string `json:"ipv6,omitempty"`
	IPv6Gateway       string `json:"ipv6Gateway,omitempty"`
	Activate          bool   `json:"activate"`
	NoDefaultRoute    bool   `json:"noDefaultRoute"`
	InterfaceName     string `json:"interfaceName,omitempty"`
	NoActivate        bool   `json:"noActivate"`
	IPv4DNSSearch     string `json:"ipv4DnsSearch,omitempty"`
	IPv6DNSSearch     string `json:"ipv6DnsSearch,omitempty"`
	IPv4IgnoreAutoDNS bool   `json:"ipv4IgnoreAutoDns"`
	IPv6IgnoreAutoDNS bool   `json:"ipv6IgnoreAutoDns"`
}

// --- Firewall Configuration ---

type FirewallConfig struct {
	Enabled  bool     `json:"enabled"`
	Services []string `json:"services,omitempty"`
	Ports    []string `json:"ports,omitempty"`
}

func NewFirewallConfig() FirewallConfig {
	return FirewallConfig{
		Enabled: true,
	}
}

// --- SELinux Configuration ---

type SELinuxConfig struct {
	Mode string `json:"mode"` // enforcing, permissive, disabled
}

func NewSELinuxConfig() SELinuxConfig {
	return SELinuxConfig{
		Mode: "enforcing",
	}
}

// --- Auth Configuration ---

type AuthConfig struct {
	EnableShadow      bool   `json:"enableShadow"`
	PasswordAlgorithm string `json:"passwordAlgorithm,omitempty"` // sha256, sha512
}

func NewAuthConfig() AuthConfig {
	return AuthConfig{
		EnableShadow:      true,
		PasswordAlgorithm: "sha512",
	}
}

// --- Realm Configuration ---

type RealmConfig struct {
	Join string `json:"join,omitempty"`
}

func NewRealmConfig() RealmConfig {
	return RealmConfig{}
}

// --- Services Configuration ---

type ServicesConfig struct {
	Enabled  []string `json:"enabled,omitempty"`
	Disabled []string `json:"disabled,omitempty"`
}

func NewServicesConfig() ServicesConfig {
	return ServicesConfig{}
}

// --- Kdump Configuration ---

type KdumpConfig struct {
	Enabled   bool   `json:"enabled"`
	ReserveMb string `json:"reserveMb,omitempty"` // auto, or numeric value in MB
}

func NewKdumpConfig() KdumpConfig {
	return KdumpConfig{Enabled: false}
}

// --- Repo Configuration ---

type RepoConfig struct {
	Name       string `json:"name"`
	BaseURL    string `json:"baseurl,omitempty"`
	MirrorList string `json:"mirrorlist,omitempty"`
	Cost       int    `json:"cost"`
}

// --- Packages Configuration ---

type PackagesConfig struct {
	Packages      []string       `json:"packages"`
	Groups        []PackageGroup `json:"groups"`
	Languages     []string       `json:"languages,omitempty"`
	Default       bool           `json:"default"`
	NoBase        bool           `json:"noBase"`
	ExcludeDocs   bool           `json:"excludeDocs"`
	IgnoreMissing bool           `json:"ignoreMissing"`
}

type PackageGroup struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"` // if true, group is @name; if false, group is @^name (mandatory)
}

// --- Script Configuration ---

type ScriptConfig struct {
	Type        string `json:"type"` // pre, post
	Interpreter string `json:"interpreter,omitempty"`
	Content     string `json:"content"`
	NoChroot    bool   `json:"noChroot"`
	ErrorOnFail bool   `json:"errorOnFail"`
	Log         string `json:"log,omitempty"` // --log= path for %post sections
}

// --- Serialization to ks.cfg format ---

// ToString generates a Kickstart configuration string
func (c *KickstartConfig) ToString() string {
	var sb strings.Builder

	// Track which structured commands have already been emitted (either
	// as a replacement of a matching RawLine, or as a new line appended
	// after RawLines). This lets us avoid duplicates.
	emitted := make(map[string]bool)

	// Commands that have a structured representation. We walk RawLines
	// once and, for each line that starts with one of these, emit the
	// structured field instead of the original line. This preserves
	// original order while reflecting any user edits to the form.
	structuredCmds := []string{
		"lang", "keyboard", "timezone", "rootpw", "eula", "authconfig",
		"selinux", "firewall", "services", "bootloader", "zerombr",
		"ignoredisk", "clearpart", "autopart", "part", "volgroup",
		"logvol", "raid", "btrfs", "network", "repo", "skipx",
		"firstboot", "reboot", "poweroff", "shutdown", "halt",
		"cdrom", "nfs", "url", "harddrive", "text", "cmdline", "graphical",
	}
	isStructured := make(map[string]bool, len(structuredCmds))
	for _, c := range structuredCmds {
		isStructured[c] = true
	}

	// Helper to emit the structured version of one command, returning true
	// if anything was written. Updates `emitted` as a side effect.
	emitStructured := func(cmd string) bool {
		switch cmd {
		case "lang":
			if c.Locale.Lang != "" {
				sb.WriteString(fmt.Sprintf("lang %s", c.Locale.Lang))
				if c.Locale.AddSupport != "" {
					sb.WriteString(fmt.Sprintf(" --addsupport=%s", c.Locale.AddSupport))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "keyboard":
			if c.Locale.Keymap != "" || c.Locale.XLayouts != "" {
				sb.WriteString("keyboard ")
				if c.Locale.Keymap != "" {
					sb.WriteString(fmt.Sprintf("--vckeymap=%s", c.Locale.Keymap))
				}
				if c.Locale.XLayouts != "" {
					sb.WriteString(fmt.Sprintf("--xlayouts=%s", c.Locale.XLayouts))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "timezone":
			if c.Locale.Timezone != "" {
				sb.WriteString(fmt.Sprintf("timezone %s", c.Locale.Timezone))
				if c.Locale.UTC {
					sb.WriteString(" --utc")
				}
				if c.Locale.NoNTP {
					sb.WriteString(" --nontp")
				}
				if c.Locale.NTPServers != "" {
					sb.WriteString(fmt.Sprintf(" --ntpservers=%s", c.Locale.NTPServers))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "rootpw":
			if c.RootPassword.IsSet {
				sb.WriteString("rootpw ")
				if c.RootPassword.IsCrypted {
					sb.WriteString("--iscrypted ")
					sb.WriteString(c.RootPassword.Password)
				} else {
					sb.WriteString("--plaintext ")
					sb.WriteString(c.RootPassword.Password)
				}
				if c.RootPassword.Lock {
					sb.WriteString(" --lock")
				}
				if c.RootPassword.AllowSsh {
					sb.WriteString(" --allow-ssh")
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "text", "cmdline", "graphical":
			if c.InstallMode == cmd {
				sb.WriteString(cmd + "\n")
				emitted[cmd] = true
				return true
			}
		case "cdrom", "nfs", "url", "harddrive":
			if c.Method.Type == cmd {
				if cmd == "cdrom" {
					sb.WriteString("cdrom\n")
				} else if cmd == "nfs" {
					sb.WriteString("nfs")
					if c.Method.Server != "" {
						sb.WriteString(fmt.Sprintf(" --server=%s", c.Method.Server))
					}
					if c.Method.Dir != "" {
						sb.WriteString(fmt.Sprintf(" --dir=%s", c.Method.Dir))
					}
					if c.Method.Opts != "" {
						sb.WriteString(fmt.Sprintf(" --opts=%s", c.Method.Opts))
					}
					sb.WriteString("\n")
				} else if cmd == "url" {
					sb.WriteString("url")
					if c.Method.URL != "" {
						sb.WriteString(fmt.Sprintf(" --url=%s", c.Method.URL))
					}
					if c.Method.NoSSL {
						sb.WriteString(" --noverifyssl")
					}
					if c.Method.Proxy != "" {
						sb.WriteString(fmt.Sprintf(" --proxy=%s", c.Method.Proxy))
					}
					sb.WriteString("\n")
				} else if cmd == "harddrive" {
					sb.WriteString("harddrive")
					if c.Method.Partition != "" {
						sb.WriteString(fmt.Sprintf(" --partition=%s", c.Method.Partition))
					}
					if c.Method.Dir != "" {
						sb.WriteString(fmt.Sprintf(" --dir=%s", c.Method.Dir))
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "bootloader":
			if c.Bootloader.Location != "" || c.Bootloader.Append != "" || c.Bootloader.BootDrive != "" {
				sb.WriteString("bootloader")
				if c.Bootloader.Location != "" {
					sb.WriteString(fmt.Sprintf(" --location=%s", c.Bootloader.Location))
				}
				if c.Bootloader.Append != "" {
					sb.WriteString(fmt.Sprintf(" --append=%s", c.Bootloader.Append))
				}
				if c.Bootloader.BootDrive != "" {
					sb.WriteString(fmt.Sprintf(" --boot-drive=%s", c.Bootloader.BootDrive))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "authconfig":
			// The web UI has no authconfig field. Match that
			// by NOT synthesising an authconfig line in the
			// preview — even when AuthConfig carries default
			// values from a parsed template. (authconfig has
			// been replaced by `authselect` in RHEL 9+, so
			// omitting it is safe for current targets.)
			return false
		case "selinux":
			if c.SELinux.Mode != "" {
				sb.WriteString(fmt.Sprintf("selinux --%s\n", c.SELinux.Mode))
				emitted[cmd] = true
				return true
			}
		case "firewall":
			sb.WriteString("firewall")
			if c.Firewall.Enabled {
				sb.WriteString(" --enabled")
			} else {
				sb.WriteString(" --disabled")
			}
			for _, svc := range c.Firewall.Services {
				sb.WriteString(fmt.Sprintf(" --service=%s", svc))
			}
			for _, port := range c.Firewall.Ports {
				sb.WriteString(fmt.Sprintf(" --port=%s", port))
			}
			sb.WriteString("\n")
			emitted[cmd] = true
			return true
		case "services":
			if len(c.Services.Enabled) > 0 || len(c.Services.Disabled) > 0 {
				sb.WriteString("services")
				if len(c.Services.Enabled) > 0 {
					sb.WriteString(fmt.Sprintf(" --enabled=%s", strings.Join(c.Services.Enabled, ",")))
				}
				if len(c.Services.Disabled) > 0 {
					sb.WriteString(fmt.Sprintf(" --disabled=%s", strings.Join(c.Services.Disabled, ",")))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "zerombr":
			if c.Storage.Zerombr {
				sb.WriteString("zerombr\n")
				emitted[cmd] = true
				return true
			}
		case "ignoredisk":
			if len(c.Storage.IgnoreDiskDrives) > 0 {
				sb.WriteString(fmt.Sprintf("ignoredisk --drives=%s\n", strings.Join(c.Storage.IgnoreDiskDrives, ",")))
				emitted[cmd] = true
				return true
			}
			if len(c.Storage.OnlyUseDrives) > 0 {
				sb.WriteString(fmt.Sprintf("ignoredisk --only-use=%s\n", strings.Join(c.Storage.OnlyUseDrives, ",")))
				emitted[cmd] = true
				return true
			}
		case "clearpart":
			if c.Storage.ClearAll || c.Storage.ClearLinux {
				sb.WriteString("clearpart --")
				if c.Storage.ClearAll {
					sb.WriteString("all")
				} else {
					sb.WriteString("linux")
				}
				if c.Storage.InitLabel {
					sb.WriteString(" --initlabel")
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "autopart":
			if c.Storage.AutoPart {
				sb.WriteString("autopart")
				if c.Storage.AutoPartType != "" {
					sb.WriteString(fmt.Sprintf(" --type=%s", c.Storage.AutoPartType))
				}
				sb.WriteString("\n")
				emitted[cmd] = true
				return true
			}
		case "part":
			if !c.Storage.AutoPart && len(c.Storage.Partitions) > 0 {
				for _, part := range c.Storage.Partitions {
					sb.WriteString("part ")
					if part.Mountpoint != "" {
						sb.WriteString(fmt.Sprintf("%s", part.Mountpoint))
					}
					if part.FSType != "" {
						sb.WriteString(fmt.Sprintf(" --fstype=%s", part.FSType))
					}
					if part.Size > 0 {
						sb.WriteString(fmt.Sprintf(" --size=%d", part.Size))
					}
					if part.MaxSize > 0 {
						sb.WriteString(fmt.Sprintf(" --maxsize=%d", part.MaxSize))
					}
					if part.Grow {
						sb.WriteString(" --grow")
					}
					if part.OnDisk != "" {
						sb.WriteString(fmt.Sprintf(" --ondisk=%s", part.OnDisk))
					}
					if part.AsPrimary {
						sb.WriteString(" --asprimary")
					}
					if part.Encrypted {
						sb.WriteString(" --encrypted")
						if part.Passphrase != "" {
							sb.WriteString(fmt.Sprintf(" --passphrase=%s", part.Passphrase))
						}
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "volgroup":
			if !c.Storage.AutoPart && len(c.Storage.VolGroups) > 0 {
				for _, vg := range c.Storage.VolGroups {
					sb.WriteString(fmt.Sprintf("volgroup %s", vg.Name))
					if vg.PESize != "" {
						sb.WriteString(fmt.Sprintf(" --pesize=%s", vg.PESize))
					}
					sb.WriteString(" ")
					if len(vg.PhysicalVolumes) > 0 {
						sb.WriteString(strings.Join(vg.PhysicalVolumes, " "))
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "logvol":
			if !c.Storage.AutoPart && len(c.Storage.LogVols) > 0 {
				for _, lv := range c.Storage.LogVols {
					sb.WriteString("logvol ")
					if lv.Mountpoint != "" {
						sb.WriteString(fmt.Sprintf("%s", lv.Mountpoint))
					}
					sb.WriteString(fmt.Sprintf(" --vgname=%s --name=%s", lv.VGName, lv.Name))
					if lv.Size > 0 {
						sb.WriteString(fmt.Sprintf(" --size=%d", lv.Size))
					}
					if lv.MaxSize > 0 {
						sb.WriteString(fmt.Sprintf(" --maxsize=%d", lv.MaxSize))
					}
					if lv.Grow {
						sb.WriteString(" --grow")
					}
					if lv.FSType != "" {
						sb.WriteString(fmt.Sprintf(" --fstype=%s", lv.FSType))
					}
					if lv.Encrypted {
						sb.WriteString(" --encrypted")
						if lv.Passphrase != "" {
							sb.WriteString(fmt.Sprintf(" --passphrase=%s", lv.Passphrase))
						}
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "raid":
			if !c.Storage.AutoPart && len(c.Storage.Raids) > 0 {
				for _, raid := range c.Storage.Raids {
					sb.WriteString("raid ")
					if raid.Mountpoint != "" {
						sb.WriteString(fmt.Sprintf("%s", raid.Mountpoint))
					}
					if raid.Level != "" {
						sb.WriteString(fmt.Sprintf(" --level=%s", raid.Level))
					}
					if raid.Device != "" {
						sb.WriteString(fmt.Sprintf(" --device=%s", raid.Device))
					}
					if raid.Spares > 0 {
						sb.WriteString(fmt.Sprintf(" --spares=%d", raid.Spares))
					}
					if raid.FSType != "" {
						sb.WriteString(fmt.Sprintf(" --fstype=%s", raid.FSType))
					}
					if raid.Encrypted {
						sb.WriteString(" --encrypted")
						if raid.Passphrase != "" {
							sb.WriteString(fmt.Sprintf(" --passphrase=%s", raid.Passphrase))
						}
					}
					sb.WriteString(" ")
					if len(raid.Devices) > 0 {
						sb.WriteString(strings.Join(raid.Devices, " "))
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "btrfs":
			if !c.Storage.AutoPart && len(c.Storage.Btrfs) > 0 {
				for _, btrfs := range c.Storage.Btrfs {
					sb.WriteString("btrfs ")
					if btrfs.Subvol != "" && btrfs.Subvol != "none" {
						sb.WriteString(fmt.Sprintf("%s", btrfs.Subvol))
					}
					sb.WriteString(" --name=" + btrfs.Name)
					if btrfs.Level != "" {
						sb.WriteString(fmt.Sprintf(" --data=%s", btrfs.Level))
					}
					if btrfs.MetaLevel != "" {
						sb.WriteString(fmt.Sprintf(" --metadata=%s", btrfs.MetaLevel))
					}
					if btrfs.Label != "" {
						sb.WriteString(fmt.Sprintf(" --label=%s", btrfs.Label))
					}
					sb.WriteString(" ")
					if btrfs.Subvol == "none" {
						sb.WriteString("none")
					} else if len(btrfs.Devices) > 0 {
						sb.WriteString(strings.Join(btrfs.Devices, " "))
					}
					sb.WriteString("\n")
				}
				emitted[cmd] = true
				return true
			}
		case "network":
			for i, net := range c.Networks {
				sb.WriteString("network ")
				// Apply hostname to the first network device only, so
				// the user's hostname edit in the form is reflected in
				// the output.
				if i == 0 && c.Locale.Hostname != "" {
					sb.WriteString(fmt.Sprintf("--hostname=%s ", c.Locale.Hostname))
				}
				if net.Device != "" {
					sb.WriteString(fmt.Sprintf("--device=%s ", net.Device))
				}
				if net.BootProto != "" {
					sb.WriteString(fmt.Sprintf("--bootproto=%s ", net.BootProto))
				}
				if net.IP != "" {
					sb.WriteString(fmt.Sprintf("--ip=%s ", net.IP))
				}
				if net.Gateway != "" {
					sb.WriteString(fmt.Sprintf("--gateway=%s ", net.Gateway))
				}
				if net.Nameserver != "" {
					sb.WriteString(fmt.Sprintf("--nameserver=%s ", net.Nameserver))
				}
				if net.Netmask != "" {
					sb.WriteString(fmt.Sprintf("--netmask=%s ", net.Netmask))
				}
				if net.OnBoot {
					sb.WriteString("--onboot=yes ")
				}
				if net.NoDNS {
					sb.WriteString("--nodns ")
				}
				if net.NoIPv4 {
					sb.WriteString("--noipv4 ")
				}
				if net.NoIPv6 {
					sb.WriteString("--noipv6 ")
				}
				if net.IPv6 != "" {
					sb.WriteString(fmt.Sprintf("--ipv6=%s ", net.IPv6))
				}
				if net.IPv6Gateway != "" {
					sb.WriteString(fmt.Sprintf("--ipv6gateway=%s ", net.IPv6Gateway))
				}
				if net.MTU != "" {
					sb.WriteString(fmt.Sprintf("--mtu=%s ", net.MTU))
				}
				if net.Activate {
					sb.WriteString("--activate ")
				}
				if net.NoDefaultRoute {
					sb.WriteString("--nodefroute ")
				}
				if net.InterfaceName != "" {
					sb.WriteString(fmt.Sprintf("--interfacename=%s ", net.InterfaceName))
				}
				if net.NoActivate {
					sb.WriteString("--no-activate ")
				}
				if net.IPv4DNSSearch != "" {
					sb.WriteString(fmt.Sprintf("--ipv4-dns-search=%s ", net.IPv4DNSSearch))
				}
				if net.IPv6DNSSearch != "" {
					sb.WriteString(fmt.Sprintf("--ipv6-dns-search=%s ", net.IPv6DNSSearch))
				}
				if net.IPv4IgnoreAutoDNS {
					sb.WriteString("--ipv4-ignore-auto-dns ")
				}
				if net.IPv6IgnoreAutoDNS {
					sb.WriteString("--ipv6-ignore-auto-dns ")
				}
				sb.WriteString("\n")
			}
			emitted[cmd] = true
			return true
		case "repo":
			for _, repo := range c.Repos {
				sb.WriteString(fmt.Sprintf("repo --name=%s", repo.Name))
				if repo.BaseURL != "" {
					sb.WriteString(fmt.Sprintf(" --baseurl=%s", repo.BaseURL))
				}
				if repo.MirrorList != "" {
					sb.WriteString(fmt.Sprintf(" --mirrorlist=%s", repo.MirrorList))
				}
				if repo.Cost > 0 {
					sb.WriteString(fmt.Sprintf(" --cost=%d", repo.Cost))
				}
				sb.WriteString("\n")
			}
			emitted[cmd] = true
			return true
		case "skipx":
			if c.Graphics.SkipX {
				sb.WriteString("skipx\n")
				emitted[cmd] = true
				return true
			}
		case "firstboot":
			if c.Graphics.FirstBoot != "" {
				sb.WriteString(fmt.Sprintf("firstboot --%s\n", c.Graphics.FirstBoot))
				emitted[cmd] = true
				return true
			}
		case "reboot", "poweroff", "shutdown", "halt":
			if c.PowerAction == cmd {
				sb.WriteString(cmd + "\n")
				emitted[cmd] = true
				return true
			}
		case "eula":
			if c.EULA == "agreed" {
				sb.WriteString("eula --agreed\n")
				emitted[cmd] = true
				return true
			}
		}
		return false
	}

	// Sections whose contents we drop in Pass 1 and re-emit in a
	// dedicated block at the end of ToString. For each known section we
	// either write the matching struct field (e.g. c.Packages for
	// %packages) or, if the struct field has no representation, the
	// original RawLines content for that section (re-emitting it
	// verbatim from the buffered block, deduplicated against any
	// user-driven changes).
	knownSections := map[string]bool{
		"packages": true, // -> c.Packages
		"pre":      true, // -> c.PreScripts
		"post":     true, // -> c.PostScripts / c.PostScriptsNoChroot
		"anaconda": true, // -> emitted from c.CustomCmds (pwpolicy ...)
		"addon":    true, // -> passthrough (no struct field)
	}
	// Drop everything inside %addon / %anaconda / etc — we re-emit
	// these from struct fields at the bottom.
	dropSectionContent := map[string]bool{
		"packages": true, "pre": true, "post": true,
		"anaconda": true, "addon": true,
	}
	// Per-section buffer to re-emit RawLines content for sections
	// that have no struct field (e.g. an unknown %addon or %anaconda
	// body when the user did not change it). Keyed by section name.
	sectionBuffer := make(map[string][]string)
	// Original opener line for each section (with flags, e.g.
	// "%addon com_redhat_kdump --enable --reserve-mb='auto'"). The
	// section name alone ("addon") is not enough to reproduce a valid
	// opener — we must remember the full original line.
	sectionOpener := make(map[string]string)
	// Track the open section ("packages" / "pre" / "post" / "anaconda" /
	// "addon" / etc). For known sections we drop the contents; for
	// unknown sections we copy them through.
	openSection := ""

	// Pass 1: walk RawLines. For each line, if it starts with a
	// structured command, emit the structured version in its place;
	// otherwise copy it through. This preserves order and handles
	// unknown commands (e.g. custom commands).
	for _, line := range c.RawLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Preserve blank lines only when not inside a structured
			// section (where the post-pass block will write the body).
			if openSection == "" || !knownSections[openSection] {
				sb.WriteString(line + "\n")
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			if openSection == "" || !knownSections[openSection] {
				sb.WriteString(line + "\n")
			}
			continue
		}
		if strings.HasPrefix(trimmed, "%") {
			if strings.HasPrefix(trimmed, "%end") {
				// End of current section.
				openSection = ""
				continue
			}
			// Section opener
			sectionName := strings.TrimSpace(strings.TrimPrefix(trimmed, "%"))
			if idx := strings.Index(sectionName, " "); idx >= 0 {
				sectionName = sectionName[:idx]
			}
			openSection = sectionName
			// Drop the opener and content for sections we will re-emit
			// at the bottom of ToString. Mark as seen so the post-pass
			// block can decide whether to write the structured body.
			if dropSectionContent[openSection] {
				// Remember the original opener line so the post-pass
				// block can write it verbatim — this is critical for
				// sections like %addon com_redhat_kdump that carry
				// flags after the section name.
				sectionOpener[openSection] = line
				emitted["__section_"+openSection+"__"] = true
				continue
			}
			// Unknown section: keep the opener verbatim and copy the
			// rest of the block through.
			sb.WriteString(line + "\n")
			continue
		}

		// If we are inside a section whose content is to be re-emitted
		// from a struct field / buffer, drop the line here and let the
		// bottom block handle it. For sections we are not tracking
		// (e.g. a fully custom section), copy through.
		if openSection != "" && dropSectionContent[openSection] {
			sectionBuffer[openSection] = append(sectionBuffer[openSection], line)
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) == 0 {
			sb.WriteString(line + "\n")
			continue
		}
		if isStructured[parts[0]] {
			// Skip if we have already emitted this structured command
			// (e.g. multiple "part" / "logvol" lines in the original
			// kickstart). The structured emitter outputs the whole list
			// in one call, so calling it more than once would duplicate
			// every entry.
			if emitted[parts[0]] {
				continue
			}
			// Some commands (e.g. authconfig) are intentionally NOT
			// emitted by emitStructured because the UI has no field
			// for them — but we must also skip the original line
			// here so it does not slip through as an "unknown
			// command". Detect that case by checking the return
			// value of emitStructured: false means "not emitted,
			// drop the line too".
			if !emitStructured(parts[0]) {
				continue
			}
			continue
		}
		// Unknown command (e.g. custom command) — copy through.
		sb.WriteString(line + "\n")
	}

	// Pass 2: for structured commands that did not appear in RawLines,
	// emit them now (so e.g. adding a new repo / firewall rule via the
	// form still appears in the output even if the original kickstart
	// did not have one).
	for _, cmd := range structuredCmds {
		if emitted[cmd] {
			continue
		}
		emitStructured(cmd)
	}

	// Hostname — usually rolled into `network --hostname=`, but emit a
	// standalone line as a fallback if the user provided one and there
	// is no network device entry that already covers it.
	if c.Locale.Hostname != "" && !emitted["network"] {
		sb.WriteString(fmt.Sprintf("network --hostname=%s\n", c.Locale.Hostname))
	}

	// Section markers and their contents. If the section appeared in
	// RawLines, its content was already dropped in Pass 1 and we re-emit
	// it here from the structured field (so any user edits in the form
	// win). If the section did NOT appear in RawLines, we only emit a
	// block when the struct field has data to add (e.g. user added a
	// repo / script via the form).
	sectionSeen := func(name string) bool {
		return emitted["__section_"+name+"__"]
	}

	// Pre scripts
	if sectionSeen("pre") || len(c.PreScripts) > 0 {
		for _, script := range c.PreScripts {
			sb.WriteString("%pre")
			if script.Interpreter != "" {
				sb.WriteString(fmt.Sprintf(" --interpreter=%s", script.Interpreter))
			}
			if script.ErrorOnFail {
				sb.WriteString(" --erroronfail")
			}
			sb.WriteString("\n")
			sb.WriteString(script.Content)
			sb.WriteString("\n%end\n")
		}
	}

	// Pre include (after %pre if present)
	if c.PreInclude != "" {
		sb.WriteString(fmt.Sprintf("%%include %s\n", c.PreInclude))
	}

	// Packages section
	if sectionSeen("packages") ||
		len(c.Packages.Packages) > 0 || len(c.Packages.Groups) > 0 ||
		c.Packages.Default || c.Packages.NoBase {
		sb.WriteString("%packages")
		if c.Packages.Default {
			sb.WriteString(" --default")
		}
		if c.Packages.NoBase {
			sb.WriteString(" --nobase")
		}
		if c.Packages.ExcludeDocs {
			sb.WriteString(" --excludedocs")
		}
		if c.Packages.IgnoreMissing {
			sb.WriteString(" --ignoremissing")
		}
		if len(c.Packages.Languages) > 0 {
			sb.WriteString(fmt.Sprintf(" --inst-langs=%s", strings.Join(c.Packages.Languages, ",")))
		}
		sb.WriteString("\n")

		for _, group := range c.Packages.Groups {
			if group.Optional {
				sb.WriteString(fmt.Sprintf("@%s\n", group.Name))
			} else {
				sb.WriteString(fmt.Sprintf("@^%s\n", group.Name))
			}
		}
		for _, pkg := range c.Packages.Packages {
			sb.WriteString(fmt.Sprintf("%s\n", pkg))
		}
		sb.WriteString("%end\n")
	}

	// Post scripts (nochroot first)
	if sectionSeen("post") || len(c.PostScriptsNoChroot) > 0 {
		for _, script := range c.PostScriptsNoChroot {
			sb.WriteString("%post")
			if script.Interpreter != "" {
				sb.WriteString(fmt.Sprintf(" --interpreter=%s", script.Interpreter))
			}
			if script.ErrorOnFail {
				sb.WriteString(" --erroronfail")
			}
			if script.NoChroot {
				sb.WriteString(" --nochroot")
			}
			if script.Log != "" {
				sb.WriteString(fmt.Sprintf(" --log=%s", script.Log))
			}
			sb.WriteString("\n")
			sb.WriteString(script.Content)
			sb.WriteString("\n%end\n")
		}
	}

	// Post scripts
	if sectionSeen("post") || len(c.PostScripts) > 0 {
		for _, script := range c.PostScripts {
			if script.NoChroot {
				continue // Already handled above
			}
			sb.WriteString("%post")
			if script.Interpreter != "" {
				sb.WriteString(fmt.Sprintf(" --interpreter=%s", script.Interpreter))
			}
			if script.ErrorOnFail {
				sb.WriteString(" --erroronfail")
			}
			if script.Log != "" {
				sb.WriteString(fmt.Sprintf(" --log=%s", script.Log))
			}
			sb.WriteString("\n")
			sb.WriteString(script.Content)
			sb.WriteString("\n%end\n")
		}
	}

	// Custom commands. Exclude entries that belong to a section we
	// have already re-emitted (e.g. `pwpolicy` lives in %anaconda and
	// is written as part of that block) to avoid duplication.
	customExclude := map[string]bool{}
	if sectionSeen("anaconda") {
		customExclude["pwpolicy"] = true
	}
	for name, value := range c.CustomCmds {
		if customExclude[name] {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", name, value))
	}

	// hasCustomSection reports whether c.CustomSecs has a key whose
	// first whitespace-delimited token matches `name`. Used to detect
	// sections that came in via JSON (frontend form) and therefore
	// have no RawLines entry to mark as "seen" in Pass 1.
	hasCustomSection := func(name string) bool {
		for k := range c.CustomSecs {
			head := k
			if idx := strings.IndexAny(k, " \t"); idx >= 0 {
				head = k[:idx]
			}
			if head == name {
				return true
			}
		}
		return false
	}
	// customSectionBody returns the joined content of every CustomSecs
	// entry whose first token matches `name`. When the user supplied
	// a section via the form we re-emit the body verbatim here.
	customSectionBody := func(name string) string {
		var parts []string
		for k, v := range c.CustomSecs {
			head := k
			if idx := strings.IndexAny(k, " \t"); idx >= 0 {
				head = k[:idx]
			}
			if head == name && v != "" {
				parts = append(parts, v)
			}
		}
		return strings.Join(parts, "\n")
	}
	// customSectionOpener returns the first CustomSecs key whose head
	// matches `name`. For "%addon com_redhat_kdump --enable ..." the
	// key is the full opener text and we use it directly as the
	// section opener line so flags survive.
	customSectionOpener := func(name string) string {
		for k := range c.CustomSecs {
			head := k
			if idx := strings.IndexAny(k, " \t"); idx >= 0 {
				head = k[:idx]
			}
			if head == name {
				return k
			}
		}
		return ""
	}

	// %anaconda block — re-emit from RawLines buffer if present, so
	// pwpolicy / etc. lines are written inside %anaconda ... %end.
	// Use the original opener line (with any flags) if we recorded one.
	if sectionSeen("anaconda") || hasCustomSection("anaconda") {
		var opener string
		if o, ok := sectionOpener["anaconda"]; ok && o != "" {
			opener = o
		} else if o := customSectionOpener("anaconda"); o != "" {
			opener = o
		} else {
			opener = "anaconda"
		}
		if len(sectionBuffer["anaconda"]) > 0 && opener == sectionOpener["anaconda"] {
			sb.WriteString(opener + "\n")
			for _, l := range sectionBuffer["anaconda"] {
				sb.WriteString(l + "\n")
			}
		} else {
			if strings.HasPrefix(opener, "%") {
				sb.WriteString(opener + "\n")
			} else {
				sb.WriteString("%" + opener + "\n")
			}
			for _, l := range sectionBuffer["anaconda"] {
				sb.WriteString(l + "\n")
			}
			if body := customSectionBody("anaconda"); body != "" {
				sb.WriteString(body + "\n")
			}
		}
		sb.WriteString("%end\n")
	}

	// %addon block — re-emit from RawLines buffer. The opener MUST be
	// the original line so flags like
	// "%addon com_redhat_kdump --enable --reserve-mb='auto'" survive;
	// we cannot rebuild that from the section name alone.
	// Structured kdump config takes precedence over raw addon sections.
	if c.Kdump.Enabled {
		sb.WriteString("%addon com_redhat_kdump --enable")
		if c.Kdump.ReserveMb != "" {
			sb.WriteString(fmt.Sprintf(" --reserve-mb='%s'", c.Kdump.ReserveMb))
		}
		sb.WriteString("\n%end\n")
	} else if sectionSeen("addon") || hasCustomSection("addon") {
		var opener string
		if o, ok := sectionOpener["addon"]; ok && o != "" {
			opener = o
		} else if o := customSectionOpener("addon"); o != "" {
			// CustomSecs key is the full opener (e.g.
			// "addon com_redhat_kdump --enable --reserve-mb='auto'")
			// — write it as the %addon line directly.
			opener = o
		} else {
			opener = "addon"
		}
		// Avoid emitting the opener twice when it was already in
		// RawLines and was buffered in sectionBuffer.
		if len(sectionBuffer["addon"]) > 0 && opener == sectionOpener["addon"] {
			sb.WriteString(opener + "\n")
			for _, l := range sectionBuffer["addon"] {
				sb.WriteString(l + "\n")
			}
		} else {
			// sectionOpener values already include the leading "%";
			// customSectionOpener / fallback do not.
			if strings.HasPrefix(opener, "%") {
				sb.WriteString(opener + "\n")
			} else {
				sb.WriteString("%" + opener + "\n")
			}
			for _, l := range sectionBuffer["addon"] {
				sb.WriteString(l + "\n")
			}
			if body := customSectionBody("addon"); body != "" {
				sb.WriteString(body + "\n")
			}
		}
		sb.WriteString("%end\n")
	}

	// Custom sections. Exclude anaconda / addon sections (and their
	// variants that carry flags after the name, e.g.
	// "addon com_redhat_kdump --enable --reserve-mb='auto'") because
	// we already emit those via the dedicated %addon / %anaconda
	// blocks above — otherwise the %addon block would be printed
	// twice (once from sectionBuffer, once from CustomSecs).
	for name, content := range c.CustomSecs {
		// Match the section "head" (first whitespace-delimited token).
		head := name
		if idx := strings.IndexAny(name, " \t"); idx >= 0 {
			head = name[:idx]
		}
		if dropSectionContent[head] {
			continue
		}
		sb.WriteString(fmt.Sprintf("%%%s\n%s\n%%end\n", name, content))
	}

	// EULA fallback (only emit if not already covered above and agreed)
	if c.EULA == "agreed" && !emitted["eula"] {
		sb.WriteString("eula --agreed\n")
	}

	return sb.String()
}

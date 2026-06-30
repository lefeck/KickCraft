package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kickcraft/config"
)

// Validator validates Kickstart configuration
type Validator struct {
	cfg      *config.KickstartConfig
	errors   []string
	warnings []string
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a Kickstart configuration
func (v *Validator) Validate(cfg *config.KickstartConfig) (bool, []string, []string) {
	v.cfg = cfg
	v.errors = nil
	v.warnings = nil

	// Check locale settings
	v.validateLocale()

	// Check storage configuration
	v.validateStorage()

	// Check network configuration
	v.validateNetwork()

	// Check packages
	v.validatePackages()

	// Check scripts
	v.validateScripts()

	// Check root password
	v.validateRootPassword()

	return len(v.errors) == 0, v.errors, v.warnings
}

func (v *Validator) addError(format string, args ...interface{}) {
	v.errors = append(v.errors, fmt.Sprintf(format, args...))
}

func (v *Validator) addWarning(format string, args ...interface{}) {
	v.warnings = append(v.warnings, fmt.Sprintf(format, args...))
}

func (v *Validator) validateLocale() {
	if v.cfg.Locale.Lang == "" {
		v.addError("lang command is required")
	}
}

func (v *Validator) validateStorage() {
	storage := v.cfg.Storage

	// Check for zerombr or clearpart
	if !storage.Zerombr && !storage.ClearAll && len(storage.Partitions) == 0 && len(storage.LogVols) == 0 {
		v.addWarning("Consider adding zerombr or clearpart to avoid interactive partitioning")
	}

	// Validate partitions
	for i, part := range storage.Partitions {
		if part.Mountpoint == "" && part.FSType == "" {
			v.addError("partition %d: must specify mountpoint or fstype", i+1)
		}
	}

	// Validate RAID
	for i, raid := range storage.Raids {
		if raid.Level == "" {
			v.addError("raid %d: must specify --level", i+1)
		}
		if len(raid.Devices) < 2 {
			v.addError("raid %d: requires at least 2 devices", i+1)
		}
	}

	// Validate volume groups
	for i, vg := range storage.VolGroups {
		if vg.Name == "" {
			v.addError("volgroup %d: must specify --name", i+1)
		}
	}

	// Validate logical volumes
	for i, lv := range storage.LogVols {
		if lv.VGName == "" {
			v.addError("logvol %d: must specify --vgname", i+1)
		}
		if lv.Name == "" {
			v.addError("logvol %d: must specify --name", i+1)
		}
	}
}

func (v *Validator) validateNetwork() {
	// At least one network device should be configured
	if len(v.cfg.Networks) == 0 {
		v.addWarning("no network devices configured")
	}

	for i, net := range v.cfg.Networks {
		if net.Device == "" {
			v.addWarning("network device %d: --device not specified", i+1)
		}
		if net.BootProto == "static" && net.IP == "" {
			v.addError("network device %d: static bootproto requires --ip", i+1)
		}
	}
}

func (v *Validator) validatePackages() {
	// Packages are optional but warn if empty
	if len(v.cfg.Packages.Packages) == 0 && len(v.cfg.Packages.Groups) == 0 {
		v.addWarning("no packages specified - minimal installation will be performed")
	}
}

func (v *Validator) validateScripts() {
	for i, script := range v.cfg.PreScripts {
		if script.Content == "" {
			v.addWarning("pre script %d: empty script", i+1)
		}
	}
	for i, script := range v.cfg.PostScripts {
		if script.Content == "" {
			v.addWarning("post script %d: empty script", i+1)
		}
	}
}

func (v *Validator) validateRootPassword() {
	if !v.cfg.RootPassword.IsSet {
		v.addWarning("root password not set - installation may be interactive")
	}
}

// ValidateConfig validates a complete Kickstart configuration
func ValidateConfig(cfg *config.KickstartConfig) (bool, []string, []string) {
	return NewValidator().Validate(cfg)
}

// --- String validation helpers ---

var (
	validBootProto = map[string]bool{
		"dhcp":      true,
		"bootp":     true,
		"static":    true,
		"query":     true,
		"ieee8021x": true,
	}
	validFSType = map[string]bool{
		"xfs":      true,
		"ext4":     true,
		"ext3":     true,
		"ext2":     true,
		"btrfs":    true,
		"vfat":     true,
		"swap":     true,
		"biosboot": true,
		"boot":     true,
		"pvfs":     true,
		"ceph":     true,
	}
	validSELinuxMode = map[string]bool{
		"enforcing":  true,
		"permissive": true,
		"disabled":   true,
	}
)

// ValidateBootProto validates the boot protocol
func ValidateBootProto(protocol string) bool {
	return validBootProto[protocol]
}

// ValidateFSType validates the filesystem type
func ValidateFSType(fsType string) bool {
	return validFSType[fsType]
}

// ValidateSELinuxMode validates the SELinux mode
func ValidateSELinuxMode(mode string) bool {
	return validSELinuxMode[mode]
}

// ValidateHostname validates a hostname
func ValidateHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 255 {
		return false
	}
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	return hostnameRegex.MatchString(hostname)
}

// ValidateIP validates an IP address
func ValidateIP(ip string) bool {
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipRegex.MatchString(ip) {
		return false
	}
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		var num int
		fmt.Sscanf(part, "%d", &num)
		if num < 0 || num > 255 {
			return false
		}
	}
	return true
}

// ValidateCIDR validates a CIDR notation
func ValidateCIDR(cidr string) bool {
	cidrRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$`)
	return cidrRegex.MatchString(cidr)
}

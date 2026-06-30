package generator

// Default constants
const (
	// Volume labels
	DefaultVolumeLabel = "CentOS-AutoInstall"

	// Boot parameters
	DefaultBootParams = "inst.ks=cdrom:/ks.cfg"

	// Timeout settings
	ISOExtractTimeout = 300 // seconds
	ISOGenTimeout     = 600 // seconds
)

// Supported distributions for local ISO build (mirror repositories)
var SupportedDistros = []string{
	"rocky-8",
	"rocky-9",
	"rocky-10",
}

// Supported distributions for Download from Internet mode (ISO URLs)
var DownloadableDistros = []string{
	"rocky-8",
	"rocky-9",
	"rocky-10",
}

// ISO download sources (for Download from Internet mode)
var ISODownloadSources = map[string]ISODownloadSource{
	"rocky-8": {
		ID:   "rocky-8",
		Name: "Rocky Linux 8",
		URL:  "https://download.rockylinux.org/pub/rocky/8/isos/x86_64/Rocky-8-latest-x86_64-minimal.iso",
	},
	"rocky-9": {
		ID:   "rocky-9",
		Name: "Rocky Linux 9",
		URL:  "https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9-latest-x86_64-minimal.iso",
	},
	"rocky-10": {
		ID:   "rocky-10",
		Name: "Rocky Linux 10",
		URL:  "https://download.rockylinux.org/pub/rocky/10/isos/x86_64/Rocky-10-latest-x86_64-minimal.iso",
	},
}

type ISODownloadSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GetISODownloadSource returns the ISO download source for a distribution
func GetISODownloadSource(distro string) *ISODownloadSource {
	if src, ok := ISODownloadSources[distro]; ok {
		return &src
	}
	return nil
}

// IsValidDistro checks if a distribution is supported
func IsValidDistro(distro string) bool {
	for _, d := range SupportedDistros {
		if d == distro {
			return true
		}
	}
	return false
}

package api

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kickcraft/config"
	"github.com/kickcraft/generator"
	"github.com/kickcraft/logger"
	"github.com/kickcraft/parser"
)

var (
	handler *Handler
)

// templateEntry stores both parsed config and raw content
type templateEntry struct {
	config     *config.KickstartConfig
	rawContent string
	filePath   string // used to determine template type
}

type Handler struct {
	options     config.ServerOptions
	templates   map[string]*templateEntry
	templatesMu sync.RWMutex
	buildTasks  map[string]*BuildTask
	buildMu     sync.RWMutex
	generator   *generator.Generator
	embedDir    string
}

type BuildTask struct {
	ID           string            `json:"id"`
	Status       string            `json:"status"` // pending, running, completed, failed
	Progress     int               `json:"progress"`
	Message      string            `json:"message"`
	Steps        map[string]string `json:"steps"`     // step name -> status (pending, running, completed, failed)
	Logs         []string          `json:"logs"`      // new logs since last poll
	LogOffset    int               `json:"logOffset"` // total logs ever produced (used by frontend to detect new entries)
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime,omitempty"`
	OutputPath   string            `json:"outputPath,omitempty"`
	BuildDir     string            `json:"buildDir,omitempty"`
	VolumeLabel  string            `json:"volumeLabel,omitempty"`
	InstallMedia string            `json:"installMedia,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// HostInfo represents the host system information
type HostInfo struct {
	OS       OSInfo       `json:"os"`
	Platform PlatformInfo `json:"platform"`
	Kernel   KernelInfo   `json:"kernel"`
	Runtime  RuntimeInfo  `json:"runtime"`
}

// OSInfo contains operating system details
type OSInfo struct {
	PrettyName string `json:"prettyName"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Codename   string `json:"codename"`
}

// PlatformInfo contains platform details
type PlatformInfo struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// KernelInfo contains kernel details
type KernelInfo struct {
	Release string `json:"release"`
}

// RuntimeInfo contains runtime details
type RuntimeInfo struct {
	GoVersion string `json:"goVersion"`
	GoArch    string `json:"goArch"`
}

// InitHandler initializes the API handler
func InitHandler(opts config.ServerOptions) {
	// Create a fresh temporary directory for the build tree, matching
	// UbuntuCraft's design: /tmp/tmp.XXXX/build/mnt/packages/
	tempDir, err := os.MkdirTemp("", "tmp.")
	if err != nil {
		logger.Warn("Failed to create temporary directory: %v", err)
		tempDir = "/tmp/kickcraft"
		os.MkdirAll(tempDir, 0755)
	}

	// Initialize generator with the working directory structure
	gen := generator.New(generator.Options{
		TempDir: tempDir,
	})

	// Initialize all build directories (download, build, mnt, packages)
	if err := gen.InitDirs(); err != nil {
		logger.Warn("Failed to initialize build directories: %v", err)
	}

	// Embedded files root is at build/mnt (matches UbuntuCraft design)
	embedDir := gen.Path.Mount()

	handler = &Handler{
		options:    opts,
		templates:  make(map[string]*templateEntry),
		buildTasks: make(map[string]*BuildTask),
		generator:  gen,
		embedDir:   embedDir,
	}

	// Load preset templates
	loadPresetTemplates()
}

// loadPresetTemplates loads built-in templates
func loadPresetTemplates() {
	loadTemplatesFromDir(filepath.Join(handler.options.TemplatesDir, "presets"))
	loadTemplatesFromDir(filepath.Join(handler.options.TemplatesDir, "user"))
}

func loadTemplatesFromDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !isSupportedTemplateFile(path) {
			return nil
		}

		cfg, err := parser.ParseFromFile(path)
		if err != nil {
			logger.Warn("Failed to parse template %s: %v", path, err)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("Failed to read template file %s: %v", path, err)
			return nil
		}

		name := templateNameFromPath(path)
		handler.templatesMu.Lock()
		handler.templates[name] = &templateEntry{
			config:     cfg,
			rawContent: string(content),
			filePath:   path,
		}
		handler.templatesMu.Unlock()

		logger.Info("Loaded template: %s", name)
		return nil
	})
}

func isSupportedTemplateFile(path string) bool {
	return strings.HasSuffix(path, ".cfg")
}

func templateNameFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".cfg")
	return base
}

// HealthCheck handles health check requests
// @Summary Health check
// @Description Check if the server is running
// @Tags health
// @Produce json
// @Success 200 {object} utils.Response
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"status":  "healthy",
		"version": "1.0",
	})
}

// ParseConfig parses a Kickstart configuration
// @Summary Parse Kickstart config
// @Description Parse a Kickstart configuration string
// @Tags config
// @Accept json
// @Produce json
// @Param config body ParseRequest true "Kickstart config string"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/config/parse [post]
func ParseConfig(c *gin.Context) {
	var req ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	cfg, err := parser.ParseFromString(req.Config)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": cfg})
}

// ParseKSFile parses a raw Kickstart (.ks) file content
// @Summary Parse Kickstart file
// @Description Parse raw Kickstart file content to JSON config
// @Accept text/plain
// @Produce json
// @Param body body string true "Raw Kickstart file content"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/config/parse-ks [post]
func ParseKSFile(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "Failed to read request body"})
		return
	}

	if len(body) == 0 {
		c.JSON(400, gin.H{"success": false, "error": "Empty file content"})
		return
	}

	cfg, err := parser.ParseFromString(string(body))
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true, "data": cfg})
}

type ParseRequest struct {
	Config string `json:"config"`
}

// ValidateConfig validates a Kickstart configuration
// @Summary Validate Kickstart config
// @Description Validate a parsed Kickstart configuration
// @Tags config
// @Accept json
// @Produce json
// @Param config body config.KickstartConfig true "Kickstart config"
// @Success 200 {object} ValidateResponse
// @Router /api/config/validate [post]
func ValidateConfig(c *gin.Context) {
	var cfg config.KickstartConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		logger.Error("ValidateConfig bind error: %v", err)
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	logger.Info("Validating config...")
	valid, errors, warnings := parser.ValidateConfig(&cfg)
	logger.Info("Validation result: valid=%v, errors=%d, warnings=%d", valid, len(errors), len(warnings))
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"valid":    valid,
			"errors":   errors,
			"warnings": warnings,
		},
	})
}

type ValidateResponse struct {
	Success  bool
	Valid    bool
	Errors   []string
	Warnings []string
}

// GenerateConfig generates a Kickstart configuration from the config object
// @Summary Generate Kickstart config
// @Description Generate a Kickstart configuration string from config object
// @Tags config
// @Accept json
// @Produce json
// @Param config body config.KickstartConfig true "Kickstart config"
// @Success 200 {object} utils.Response
// @Router /api/config/generate [post]
func GenerateConfig(c *gin.Context) {
	var cfg config.KickstartConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		logger.Error("GenerateConfig bind error: %v", err)
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	logger.Info("Generating config, locale: %+v", cfg.Locale)
	ksConfig := cfg.ToString()
	logger.Info("Generated config length: %d", len(ksConfig))
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"config": ksConfig,
		},
	})
}

// defaultTemplateName is the canonical name of the preset served by
// GetDefaultConfig. The frontend's "Load Default Config" button and the
// app-startup auto-load both rely on this endpoint returning the
// contents of templates/presets/default.cfg (or templates/user/default.cfg,
// which overrides the preset if present).
const defaultTemplateName = "default"

// GetDefaultConfig returns the contents of the "default" Kickstart template.
// The lookup is by exact name rather than "first template in iteration order",
// which previously caused the returned template to change depending on the
// randomized Go map iteration seed.
// @Summary Get default configuration
// @Description Get the default Kickstart configuration (templates/presets/default.cfg)
// @Tags config
// @Produce json
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/config/default [get]
func GetDefaultConfig(c *gin.Context) {
	handler.templatesMu.RLock()
	defer handler.templatesMu.RUnlock()

	entry, ok := handler.templates[defaultTemplateName]
	if !ok || entry == nil || entry.config == nil {
		logger.Warn("GetDefaultConfig: default template not loaded")
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Default configuration template (default.cfg) not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"config":  entry.config.ToString(),
		"name":    defaultTemplateName,
	})
}

// GetDistros returns the list of supported distributions
// @Summary Get supported distributions
// @Description Get the list of supported distributions
// @Tags distros
// @Produce json
// @Success 200 {object} utils.Response
// @Router /api/distros [get]
func GetDistros(c *gin.Context) {
	distros := []Distro{
		{
			ID:      "rocky-8",
			Name:    "Rocky Linux 8",
			Version: "8",
		},
		{
			ID:      "rocky-9",
			Name:    "Rocky Linux 9",
			Version: "9",
		},
		{
			ID:      "rocky-10",
			Name:    "Rocky Linux 10",
			Version: "10",
		},
	}

	c.JSON(200, gin.H{"success": true, "data": distros})
}

type Distro struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// GetISODownloadSources returns the list of downloadable ISO sources
// @Summary Get ISO download sources
// @Description Get the list of available ISO download sources for Download from Internet mode
// @Tags distros
// @Produce json
// @Success 200 {object} utils.Response
// @Router /api/distros/sources [get]
func GetISODownloadSources(c *gin.Context) {
	sources := []ISODownloadSource{
		{
			ID:   "rocky-8",
			Name: "Rocky Linux 8",
			URL:  "https://download.rockylinux.org/pub/rocky/8/isos/x86_64/Rocky-8-latest-x86_64-minimal.iso",
		},
		{
			ID:   "rocky-9",
			Name: "Rocky Linux 9",
			URL:  "https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9-latest-x86_64-minimal.iso",
		},
		{
			ID:   "rocky-10",
			Name: "Rocky Linux 10",
			URL:  "https://download.rockylinux.org/pub/rocky/10/isos/x86_64/Rocky-10-latest-x86_64-minimal.iso",
		},
	}

	c.JSON(200, gin.H{"success": true, "data": sources})
}

type ISODownloadSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// GetTemplates returns all available templates
// @Summary Get templates
// @Description Get all available Kickstart templates
// @Tags templates
// @Produce json
// @Success 200 {object} utils.Response
// @Router /api/templates [get]
func GetTemplates(c *gin.Context) {
	handler.templatesMu.RLock()
	defer handler.templatesMu.RUnlock()

	templates := make([]TemplateInfo, 0, len(handler.templates))
	presetsDir := filepath.Join(handler.options.TemplatesDir, "presets")
	userDir := filepath.Join(handler.options.TemplatesDir, "user")

	for name, entry := range handler.templates {
		// Determine template type based on path
		templateType := "preset"
		if entry.filePath != "" {
			// Check if file is in user directory
			if strings.HasPrefix(entry.filePath, userDir) {
				templateType = "user"
			} else if strings.HasPrefix(entry.filePath, presetsDir) {
				templateType = "preset"
			}
		}

		templates = append(templates, TemplateInfo{
			Name:       name,
			Type:       templateType,
			Config:     entry.config,
			RawContent: entry.rawContent,
		})
	}

	c.JSON(200, gin.H{"success": true, "data": templates})
}

type TemplateInfo struct {
	Name       string                  `json:"name"`
	Type       string                  `json:"type"` // preset, user
	Config     *config.KickstartConfig `json:"config,omitempty"`
	RawContent string                  `json:"rawContent,omitempty"`
}

// GetTemplate returns a specific template
// @Summary Get template by name
// @Description Get a specific Kickstart template by name
// @Tags templates
// @Produce json
// @Param name path string true "Template name"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/templates/{name} [get]
func GetTemplate(c *gin.Context) {
	name := c.Param("name")

	handler.templatesMu.RLock()
	entry, exists := handler.templates[name]
	handler.templatesMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"success": false, "error": "Template not found"})
		return
	}

	// Determine template type
	presetsDir := filepath.Join(handler.options.TemplatesDir, "presets")
	userDir := filepath.Join(handler.options.TemplatesDir, "user")
	templateType := "preset"
	if entry.filePath != "" {
		if strings.HasPrefix(entry.filePath, userDir) {
			templateType = "user"
		} else if strings.HasPrefix(entry.filePath, presetsDir) {
			templateType = "preset"
		}
	}

	// Return proper TemplateInfo structure
	templateInfo := TemplateInfo{
		Name:       name,
		Type:       templateType,
		Config:     entry.config,
		RawContent: entry.rawContent,
	}

	c.JSON(200, gin.H{"success": true, "data": templateInfo})
}

// SaveTemplate saves a user template
// @Summary Save template
// @Description Save a user-defined template
// @Tags templates
// @Accept json
// @Produce json
// @Param template body SaveTemplateRequest true "Template to save"
// @Success 200 {object} utils.Response
// @Router /api/templates [post]
func SaveTemplate(c *gin.Context) {
	var req SaveTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Validate template name
	if strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") {
		c.JSON(400, gin.H{"success": false, "error": "Invalid template name"})
		return
	}

	// Parse the config if provided as string
	var cfg *config.KickstartConfig
	var rawContent string

	if req.RawContent != "" {
		// Use raw content directly
		rawContent = req.RawContent
		// Also parse it to update the config
		cfg, _ = parser.ParseFromString(req.RawContent)
	} else if req.Config != nil {
		cfg = req.Config
		rawContent = cfg.ToString()
	} else if req.ConfigString != "" {
		var err error
		cfg, err = parser.ParseFromString(req.ConfigString)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": err.Error()})
			return
		}
		rawContent = req.ConfigString
	} else {
		c.JSON(400, gin.H{"success": false, "error": "No config provided"})
		return
	}

	// Save to user templates directory
	userDir := filepath.Join(handler.options.TemplatesDir, "user")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// If rawContent is still empty, generate from config
	if rawContent == "" && cfg != nil {
		rawContent = cfg.ToString()
	}

	// Save as .cfg file
	cfgPath := filepath.Join(userDir, req.Name+".cfg")
	if err := os.WriteFile(cfgPath, []byte(rawContent), 0644); err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Also save as JSON for easy loading
	if cfg != nil {
		jsonPath := filepath.Join(userDir, req.Name+".json")
		data, _ := json.Marshal(cfg)
		os.WriteFile(jsonPath, data, 0644)
	}

	// Update in-memory map with filePath
	userFilePath := filepath.Join(userDir, req.Name+".cfg")
	handler.templatesMu.Lock()
	handler.templates[req.Name] = &templateEntry{
		config:     cfg,
		rawContent: rawContent,
		filePath:   userFilePath,
	}
	handler.templatesMu.Unlock()

	c.JSON(200, gin.H{"success": true, "message": "Template saved"})
}

type SaveTemplateRequest struct {
	Name         string                  `json:"name" binding:"required"`
	Config       *config.KickstartConfig `json:"config,omitempty"`
	ConfigString string                  `json:"configString,omitempty"`
	RawContent   string                  `json:"rawContent,omitempty"`
}

// UpdateTemplateRequest for PUT /api/templates/:name
type UpdateTemplateRequest struct {
	Config       *config.KickstartConfig `json:"config,omitempty"`
	ConfigString string                  `json:"configString,omitempty"`
	RawContent   string                  `json:"rawContent,omitempty"`
}

// UpdateTemplate updates an existing user template
// @Summary Update template
// @Description Update an existing user-defined template
// @Tags templates
// @Accept json
// @Produce json
// @Param name path string true "Template name"
// @Param template body UpdateTemplateRequest true "Template data to update"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/templates/{name} [put]
func UpdateTemplate(c *gin.Context) {
	name := c.Param("name")

	// Check if template exists
	handler.templatesMu.RLock()
	_, exists := handler.templates[name]
	handler.templatesMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"success": false, "error": "Template not found"})
		return
	}

	// Check if it's a user template (cannot update preset templates)
	userCfgPath := filepath.Join(handler.options.TemplatesDir, "user", name+".cfg")
	if _, err := os.Stat(userCfgPath); os.IsNotExist(err) {
		c.JSON(403, gin.H{"success": false, "error": "Cannot update preset templates"})
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	var cfg *config.KickstartConfig
	var rawContent string

	if req.RawContent != "" {
		// Use raw content directly
		rawContent = req.RawContent
		// Also parse it to update the config
		cfg, _ = parser.ParseFromString(req.RawContent)
	} else if req.Config != nil {
		cfg = req.Config
		rawContent = cfg.ToString()
	} else if req.ConfigString != "" {
		var err error
		cfg, err = parser.ParseFromString(req.ConfigString)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": err.Error()})
			return
		}
		rawContent = req.ConfigString
	} else {
		c.JSON(400, gin.H{"success": false, "error": "No content provided"})
		return
	}

	// If rawContent is still empty, generate from config
	if rawContent == "" && cfg != nil {
		rawContent = cfg.ToString()
	}

	// Save as .cfg file
	if err := os.WriteFile(userCfgPath, []byte(rawContent), 0644); err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Update in-memory map
	handler.templatesMu.Lock()
	if cfg != nil {
		handler.templates[name] = &templateEntry{
			config:     cfg,
			rawContent: rawContent,
			filePath:   userCfgPath,
		}
	}
	handler.templatesMu.Unlock()

	c.JSON(200, gin.H{"success": true, "message": "Template updated"})
}

// DeleteTemplate deletes a user template
// @Summary Delete template
// @Description Delete a user-defined template
// @Tags templates
// @Produce json
// @Param name path string true "Template name"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /api/templates/{name} [delete]
func DeleteTemplate(c *gin.Context) {
	name := c.Param("name")

	// Check if it's a preset template
	handler.templatesMu.RLock()
	_, exists := handler.templates[name]
	handler.templatesMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"success": false, "error": "Template not found"})
		return
	}

	// Only allow deleting user templates
	userCfgPath := filepath.Join(handler.options.TemplatesDir, "user", name+".cfg")
	if _, err := os.Stat(userCfgPath); os.IsNotExist(err) {
		c.JSON(403, gin.H{"success": false, "error": "Cannot delete preset templates"})
		return
	}

	// Delete files
	_ = os.Remove(userCfgPath)
	_ = os.Remove(filepath.Join(handler.options.TemplatesDir, "user", name+".json"))

	// Update in-memory map
	handler.templatesMu.Lock()
	delete(handler.templates, name)
	handler.templatesMu.Unlock()

	c.JSON(200, gin.H{"success": true, "message": "Template deleted"})
}

// DownloadISO downloads a source ISO
// @Summary Download ISO
// @Description Download a source ISO file
// @Tags iso
// @Accept json
// @Produce json
// @Param request body DownloadISORequest true "Download request"
// @Success 200 {object} utils.Response
// @Router /api/iso/download [post]
func DownloadISO(c *gin.Context) {
	var req DownloadISORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// This would trigger a download in a real implementation
	// For now, return a mock response
	taskID := uuid.New().String()

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"taskId":  taskID,
			"status":  "started",
			"message": "ISO download started",
		},
	})
}

// UploadISO receives a multipart-uploaded ISO file from the browser,
// stores it in the configured download directory, and returns its
// server-side absolute path so the client can pass it to /api/iso/generate.
// @Summary Upload ISO file
// @Description Upload a local ISO file from the browser to the server
// @Tags iso
// @Accept multipart/form-data
// @Produce json
// @Param iso formData file true "ISO file to upload"
// @Success 200 {object} map[string]interface{} "ISO uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Router /api/iso/upload [post]
func UploadISO(c *gin.Context) {
	file, err := c.FormFile("iso")
	if err != nil {
		// "unexpected EOF" typically means the client disconnected
		// (e.g. computer slept/hibernated mid-upload). Return a friendlier
		// message so the browser-side handler can give a clear prompt.
		if errors.Is(err, io.ErrUnexpectedEOF) ||
			strings.Contains(err.Error(), "unexpected EOF") ||
			strings.Contains(err.Error(), "http: request body closed") {
			logger.Warn("ISO upload: client disconnected mid-upload (computer slept?)")
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "upload was interrupted — please retry",
			})
			return
		}
		logger.Error("ISO upload: failed to read form file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "failed to retrieve uploaded file: " + err.Error(),
		})
		return
	}

	if ext := strings.ToLower(filepath.Ext(file.Filename)); ext != ".iso" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "only ISO file format is supported",
		})
		return
	}

	// Cap uploads at 10 GB to match UbuntuCraft's behaviour and avoid
	// unbounded memory / disk usage in the web handler.
	const maxSize = 10 << 30
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("file size exceeds the maximum limit of %d bytes", maxSize),
		})
		return
	}

	downloadDir := handler.generator.Path.DownloadDir()
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		logger.Error("ISO upload: cannot create download dir %s: %v", downloadDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to prepare upload directory",
		})
		return
	}

	dst := filepath.Join(downloadDir, file.Filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		logger.Error("ISO upload: failed to save file to %s: %v", dst, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "failed to save uploaded file: " + err.Error(),
		})
		return
	}

	logger.Info("ISO uploaded: %s (%d bytes)", dst, file.Size)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"filePath": dst,
		"fileName": file.Filename,
		"size":     file.Size,
	})
}

type DownloadISORequest struct {
	Distro    string `json:"distro" binding:"required"`
	URL       string `json:"url,omitempty"`
	LocalPath string `json:"localPath,omitempty"`
}

// GenerateISO generates an ISO with Kickstart configuration
// @Summary Generate ISO
// @Description Generate an ISO with embedded Kickstart configuration
// @Tags iso
// @Accept json
// @Produce json
// @Param request body GenerateISORequest true "Generate request"
// @Success 200 {object} utils.Response
// @Router /api/iso/generate [post]
func GenerateISO(c *gin.Context) {
	var req GenerateISORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Validate request
	if err := validateGenerateISORequest(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Create build task
	taskID := uuid.New().String()
	task := &BuildTask{
		ID:        taskID,
		Status:    "pending",
		Progress:  0,
		StartTime: time.Now(),
		Message:   "Starting ISO generation...",
	}

	handler.buildMu.Lock()
	handler.buildTasks[taskID] = task
	handler.buildMu.Unlock()

	// Start generation in background
	go runISOGeneration(taskID, req)

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"taskId": taskID,
			"status": "pending",
		},
	})
}

type GenerateISORequest struct {
	SourceType     string                  `json:"sourceType" binding:"required"` // "local" or "download"
	Distro         string                  `json:"distro"`                        // Distribution ID (when sourceType is "download")
	SourceISO      string                  `json:"sourceIso"`                     // Local ISO file path (when sourceType is "local")
	DestinationISO string                  `json:"destinationISO"`                // Output ISO filename
	InstallMedia   string                  `json:"installMedia"`                  // "cdrom" (default) or "harddrive"
	EthNaming      bool                    `json:"ethNaming"`                     // Append biosdevname=0 net.ifnames=0 to boot params
	SerialConsole  bool                    `json:"serialConsole"`                 // Append console=tty0 console=ttyS0,<baudRate>n8 to boot params
	BaudRate       string                  `json:"baudRate"`                      // Serial console baud rate (9600, 19200, 38400, 57600, 115200)
	Config         *config.KickstartConfig `json:"config" binding:"required"`
}

// validateGenerateISORequest validates the ISO generation request parameters
func validateGenerateISORequest(req *GenerateISORequest) error {
	if req.SourceType != "local" && req.SourceType != "download" {
		return fmt.Errorf("sourceType must be 'local' or 'download'")
	}
	if req.SourceType == "local" && req.SourceISO == "" {
		return fmt.Errorf("sourceISO is required when sourceType is 'local'")
	}
	if req.SourceType == "download" && req.Distro == "" {
		return fmt.Errorf("distro is required when sourceType is 'download'")
	}
	return nil
}

// runISOGeneration runs the ISO generation in background
func runISOGeneration(taskID string, req GenerateISORequest) {
	handler.buildMu.Lock()
	task := handler.buildTasks[taskID]
	handler.buildMu.Unlock()

	task.Status = "running"
	task.Steps = map[string]string{
		"prepare":   "completed",
		"source":    "pending",
		"kickstart": "pending",
		"boot":      "pending",
		"packages":  "pending",
		"repackage": "pending",
		"complete":  "pending",
	}
	task.Message = "Generating ISO..."

	// Determine distro and sourceISO based on sourceType
	distro := req.Distro
	sourceISO := req.SourceISO
	if req.SourceType == "download" {
		sourceISO = "" // No local ISO path for download mode
		// Look up ISO download URL from distro
		if src := generator.GetISODownloadSource(req.Distro); src != nil {
			sourceISO = src.URL
			logger.Info("ISO download source: %s -> %s", req.Distro, sourceISO)
		} else {
			logger.Error("Unknown distro for download: %s", req.Distro)
		}
	} else {
		distro = "" // No distro ID for local ISO mode
	}

	// Progress callback - updates task steps and logs
	progressCallback := func(step, status, message string) {
		handler.buildMu.Lock()
		defer handler.buildMu.Unlock()

		if task, exists := handler.buildTasks[taskID]; exists {
			task.Steps[step] = status
			task.Logs = append(task.Logs, message)
			task.LogOffset = len(task.Logs)

			// Update overall progress based on steps
			completed := 0
			for _, s := range task.Steps {
				if s == "completed" {
					completed++
				}
			}
			task.Progress = (completed * 100) / len(task.Steps)
			task.Message = message
		}
	}

	// Reuse the global generator (initialized with the /tmp/tmp.XXXX
	// build tree) so embedded files and packages are visible.
	gen := handler.generator

	// Build extra boot params from advanced options
	var extraParts []string
	if req.EthNaming {
		extraParts = append(extraParts, "biosdevname=0 net.ifnames=0")
	}
	if req.SerialConsole {
		baudRate := req.BaudRate
		if baudRate == "" {
			baudRate = "115200" // default
		}
		extraParts = append(extraParts, fmt.Sprintf("console=tty0 console=ttyS0,%sn8", baudRate))
	}
	extraBootParams := strings.Join(extraParts, " ")
	if extraBootParams != "" {
		logger.Info("Extra boot params: %s", extraBootParams)
	}

	result, err := gen.Generate(distro, req.Config, sourceISO, req.DestinationISO, req.InstallMedia, extraBootParams, req.Config.AdditionalPackages, progressCallback)
	if err != nil {
		handler.buildMu.Lock()
		task.Steps["complete"] = "failed"
		handler.buildMu.Unlock()
		task.Status = "failed"
		task.Error = err.Error()
		task.EndTime = time.Now()
		logger.Error("ISO generation failed: %v", err)
		return
	}

	handler.buildMu.Lock()
	task.Steps["complete"] = "completed"
	handler.buildMu.Unlock()
	task.Status = "completed"
	task.Progress = 100
	task.Message = "ISO generated successfully"
	task.OutputPath = result.OutputPath
	task.BuildDir = result.BuildDir
	task.VolumeLabel = result.VolumeLabel
	task.InstallMedia = result.InstallMedia
	task.EndTime = time.Now()

	logger.Info("ISO generated: %s", result.OutputPath)
}

// GetISOStatus returns the status of an ISO generation task
// @Summary Get ISO status
// @Description Get the status of an ISO generation task
// @Tags iso
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} utils.Response
// @Router /api/iso/status/{id} [get]
func GetISOStatus(c *gin.Context) {
	id := c.Param("id")
	clientLogOffset := c.Query("logOffset")

	handler.buildMu.RLock()
	task, exists := handler.buildTasks[id]
	handler.buildMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"success": false, "error": "Task not found"})
		return
	}

	offset := 0
	if clientLogOffset != "" {
		fmt.Sscanf(clientLogOffset, "%d", &offset)
	}

	response := *task
	if offset >= len(task.Logs) {
		response.Logs = nil
	} else {
		response.Logs = task.Logs[offset:]
	}

	c.JSON(200, gin.H{"success": true, "data": response})
}

// DownloadISOFile downloads the generated ISO file
// @Summary Download generated ISO
// @Description Download the generated ISO file
// @Tags iso
// @Produce octet-stream
// @Param id path string true "Task ID"
// @Success 200 {file} binary
// @Failure 404 {object} utils.Response
// @Router /api/iso/download/{id} [get]
func DownloadISOFile(c *gin.Context) {
	id := c.Param("id")

	handler.buildMu.RLock()
	task, exists := handler.buildTasks[id]
	handler.buildMu.RUnlock()

	if !exists {
		c.JSON(404, gin.H{"success": false, "error": "Task not found"})
		return
	}

	if task.Status != "completed" || task.OutputPath == "" {
		c.JSON(400, gin.H{"success": false, "error": "ISO not ready for download"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(task.OutputPath)))
	c.Header("Content-Type", "application/octet-stream")
	c.File(task.OutputPath)
}

// SearchPackages searches for RPM packages
// @Summary Search packages
// @Description Search for RPM packages in repositories
// @Tags packages
// @Accept json
// @Produce json
// @Param request body SearchPackagesRequest true "Search request"
// @Success 200 {object} utils.Response
// @Router /api/packages/search [post]
func SearchPackages(c *gin.Context) {
	var req SearchPackagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Mock package search results
	packages := []PackageInfo{
		{Name: "nginx", Summary: "nginx web server"},
		{Name: "httpd", Summary: "Apache HTTP Server"},
		{Name: "docker", Summary: "Docker container runtime"},
		{Name: "kubernetes", Summary: "Container orchestration platform"},
	}

	c.JSON(200, gin.H{"success": true, "data": packages})
}

type SearchPackagesRequest struct {
	Query  string `json:"query" binding:"required"`
	Distro string `json:"distro,omitempty"`
}

type PackageInfo struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// DownloadPackages downloads RPM packages
// @Summary Download packages
// @Description Download RPM packages to local repository
// @Tags packages
// @Accept json
// @Produce json
// @Param request body DownloadPackagesRequest true "Download request"
// @Success 200 {object} utils.Response
// @Router /api/packages/download [post]
func DownloadPackages(c *gin.Context) {
	var req DownloadPackagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Ensure packages directory exists
	packagesDir := handler.generator.Path.Packages()
	if err := os.MkdirAll(packagesDir, 0755); err != nil {
		c.JSON(500, gin.H{"success": false, "error": "failed to create packages directory"})
		return
	}

	taskID := uuid.New().String()

	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"taskId":  taskID,
			"message": "Package download started",
			"path":    packagesDir,
		},
	})
}

type DownloadPackagesRequest struct {
	Packages []string `json:"packages" binding:"required"`
	Distro   string   `json:"distro" binding:"required"`
}

// GetHostInfo returns comprehensive host system information
// @Summary Get host system info
// @Description Get detailed information about the current host system (OS, kernel, platform, runtime)
// @Tags host
// @Produce json
// @Success 200 {object} map[string]interface{} "Host info retrieved successfully"
// @Failure 500 {object} map[string]interface{} "Failed to retrieve host info"
// @Router /api/host/info [get]
func GetHostInfo(c *gin.Context) {
	hostInfo := HostInfo{
		OS:       readOSRelease(),
		Platform: readPlatformInfo(),
		Kernel:   readKernelInfo(),
		Runtime: RuntimeInfo{
			GoVersion: runtime.Version(),
			GoArch:    runtime.GOARCH,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"host":    hostInfo,
		"message": "Host info retrieved successfully",
	})
}

// readOSRelease parses /etc/os-release to extract OS information
func readOSRelease() OSInfo {
	info := OSInfo{
		PrettyName: "Unknown",
		Name:       "unknown",
		Version:    "unknown",
		Codename:   "unknown",
	}

	file, err := os.Open("/etc/os-release")
	if err != nil {
		logger.Warn("Failed to open /etc/os-release: " + err.Error())
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], "\"")

		switch key {
		case "PRETTY_NAME":
			info.PrettyName = value
		case "NAME":
			info.Name = value
		case "VERSION_ID":
			info.Version = value
		case "VERSION_CODENAME":
			info.Codename = value
		}
	}

	return info
}

// readPlatformInfo extracts platform information from Go runtime
func readPlatformInfo() PlatformInfo {
	return PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// readKernelInfo reads kernel release from /proc/version
func readKernelInfo() KernelInfo {
	info := KernelInfo{
		Release: "unknown",
	}

	file, err := os.Open("/proc/version")
	if err != nil {
		logger.Warn("Failed to open /proc/version: " + err.Error())
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		versionLine := scanner.Text()
		parts := strings.Fields(versionLine)
		if len(parts) >= 3 {
			info.Release = parts[2]
		}
	}

	return info
}

// =====================================================
// Embedded Files API
// =====================================================

// WriteEmbeddedFile writes a file to the embedded directory
// POST /api/embedded/write
func WriteEmbeddedFile(c *gin.Context) {
	var req struct {
		Path     string `json:"path" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Encoding string `json:"encoding"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.Contains(req.Path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	targetPath := filepath.Join(handler.embedDir, req.Path)
	absRoot, _ := filepath.Abs(handler.embedDir)
	absTarget, _ := filepath.Abs(targetPath)
	if !strings.HasPrefix(absTarget, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}

	// Build directory tree from root outward, creating only missing directories.
	// We deliberately avoid os.MkdirAll because it sets the same mode on all
	// created dirs, overwriting any pre-existing permissions.
	rel, err := filepath.Rel(absRoot, filepath.Dir(targetPath))
	if err == nil && rel != "." {
		current := absRoot
		for _, part := range strings.Split(rel, string(os.PathSeparator)) {
			if part == "" {
				continue
			}
			current = filepath.Join(current, part)
			if err := os.Mkdir(current, 0755); err != nil && !os.IsExist(err) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory: " + err.Error()})
				return
			}
		}
	}

	content := []byte(req.Content)
	var mode os.FileMode = 0644
	if info, err := os.Stat(targetPath); err == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(targetPath, content, mode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"targetPath": targetPath,
	})
}

// DeleteEmbeddedFile deletes a file from the embedded directory
// DELETE /api/embedded/delete
func DeleteEmbeddedFile(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.Contains(req.Path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	targetPath := filepath.Join(handler.embedDir, req.Path)
	absRoot, _ := filepath.Abs(handler.embedDir)
	absTarget, _ := filepath.Abs(targetPath)
	if !strings.HasPrefix(absTarget, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	var err error
	if info, statErr := os.Stat(targetPath); statErr == nil && info.IsDir() {
		err = os.RemoveAll(targetPath)
	} else {
		err = os.Remove(targetPath)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MkdirEmbedded creates a directory in the embedded directory
// POST /api/embedded/mkdir
func MkdirEmbedded(c *gin.Context) {
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.Contains(req.Path, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	// Only allow "packages" (plural), not "package" (singular)
	if req.Path == "package" || strings.HasSuffix(strings.TrimSuffix(req.Path, "/"), "/package") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "directory name must be 'packages' (plural), not 'package'"})
		return
	}
	targetDir := filepath.Join(handler.embedDir, req.Path)
	absRoot, _ := filepath.Abs(handler.embedDir)
	absTarget, _ := filepath.Abs(targetDir)
	if !strings.HasPrefix(absTarget, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ReadEmbeddedFile reads a file from the embedded directory
// GET /api/embedded/read
func ReadEmbeddedFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}
	if strings.Contains(filePath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	targetPath := filepath.Join(handler.embedDir, filePath)
	absRoot, _ := filepath.Abs(handler.embedDir)
	absTarget, _ := filepath.Abs(targetPath)
	if !strings.HasPrefix(absTarget, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"content": string(content),
	})
}

// countEmbeddedFiles counts all files under a path
func countEmbeddedFiles(path string) int {
	count := 0
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			count += countEmbeddedFiles(filepath.Join(path, entry.Name()))
		} else {
			count++
		}
	}
	return count
}

// ListDirectory lists files and directories in the embedded directory
// GET /api/embedded/dir
func ListDirectory(c *gin.Context) {
	relPath := c.Query("path")
	if relPath == "" {
		relPath = "/"
	}
	if strings.Contains(relPath, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	targetDir := filepath.Join(handler.embedDir, relPath)
	absRoot, _ := filepath.Abs(handler.embedDir)
	absTarget, _ := filepath.Abs(targetDir)
	if !strings.HasPrefix(absTarget, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"path":    relPath,
			"rootDir": handler.embedDir,
			"dirs":    []map[string]interface{}{},
			"files":   []map[string]interface{}{},
		})
		return
	}
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read directory"})
		return
	}
	var dirs []map[string]interface{}
	var files []map[string]interface{}
	for _, entry := range entries {
		// Skip "package" (singular) — the correct directory name is "packages" (plural).
		// This prevents accidental duplication when users create folders with the
		// wrong name while the build system creates "packages".
		if entry.IsDir() && entry.Name() == "package" {
			continue
		}
		if entry.IsDir() {
			subPath := filepath.Join(targetDir, entry.Name())
			fileCount := countEmbeddedFiles(subPath)
			dirs = append(dirs, map[string]interface{}{
				"name":      entry.Name(),
				"fileCount": fileCount,
			})
		} else {
			info, _ := entry.Info()
			files = append(files, map[string]interface{}{
				"name":     entry.Name(),
				"size":     info.Size(),
				"modified": info.ModTime().Format(time.RFC3339),
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"path":    relPath,
		"rootDir": handler.embedDir,
		"dirs":    dirs,
		"files":   files,
	})
}

// ResetBuildConfig cleans the build/ directory under the temp root,
// then recreates empty mnt/{packages,script} subdirectories so the next
// build starts fresh. The ks.cfg file is also removed.
// @Summary Reset build config
// @Description Wipe build/mnt/* and rebuild the empty mnt/{packages,script} skeleton
// @Tags iso
// @Success 200 {object} utils.Response
// @Router /api/iso/reset [post]
func ResetBuildConfig(c *gin.Context) {
	gen := handler.generator

	buildDir := gen.Path.BuildDir()
	mountDir := gen.Path.Mount()

	handler.buildMu.Lock()
	defer handler.buildMu.Unlock()

	// Remove the entire build/ tree
	if err := os.RemoveAll(buildDir); err != nil {
		logger.Error("ResetBuildConfig: failed to remove %s: %v", buildDir, err)
		c.JSON(500, gin.H{"success": false, "error": "failed to clean build directory"})
		return
	}

	// Recreate the required skeleton
	requiredDirs := []string{
		gen.Path.Packages(),
		gen.Path.Scripts(),
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error("ResetBuildConfig: failed to create %s: %v", dir, err)
			c.JSON(500, gin.H{"success": false, "error": "failed to recreate build directories"})
			return
		}
	}

	logger.Info("ResetBuildConfig: cleaned %s, preserved mnt/{packages,script}", buildDir)
	c.JSON(200, gin.H{"success": true, "data": gin.H{
		"cleaned":  buildDir,
		"mount":    mountDir,
		"packages": gen.Path.Packages(),
		"scripts":  gen.Path.Scripts(),
	}})
}

// ExtractArchive extracts a zip or tar.gz archive to the embedded directory,
// preserving file permissions stored in the archive headers.
// POST /api/embedded/extract-zip
func ExtractZipArchive(c *gin.Context) {
	prefix := c.PostForm("prefix")
	baseDir := handler.embedDir
	if prefix != "" {
		baseDir = filepath.Join(baseDir, prefix)
	}
	absRoot, _ := filepath.Abs(handler.embedDir)
	absBase, _ := filepath.Abs(baseDir)
	if !strings.HasPrefix(absBase, absRoot) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path traversal not allowed"})
		return
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create base directory: " + err.Error()})
		return
	}

	file, _, err := c.Request.FormFile("archive")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no archive provided: " + err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read archive: " + err.Error()})
		return
	}

	// Try gzip (tar.gz) first, then fall back to zip
	extracted := 0
	var lastErr string

	// Check for gzip magic bytes: 1f 8b
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		extracted, lastErr = extractTarGz(data, baseDir, absRoot)
	} else {
		extracted, lastErr = extractZip(data, baseDir, absRoot)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"extracted": extracted,
		"lastError": lastErr,
	})
}

func extractZip(data []byte, baseDir, absRoot string) (int, string) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, "invalid zip file: " + err.Error()
	}

	extracted := 0
	var lastErr string
	for _, zipEntry := range zr.File {
		if strings.Contains(zipEntry.Name, "..") {
			lastErr = "disallowed path: " + zipEntry.Name
			continue
		}

		targetPath := filepath.Join(baseDir, zipEntry.Name)
		absTarget, _ := filepath.Abs(targetPath)
		if !strings.HasPrefix(absTarget, absRoot) {
			lastErr = "path outside root: " + zipEntry.Name
			continue
		}

		if zipEntry.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				lastErr = "failed to create dir " + zipEntry.Name + ": " + err.Error()
			}
		} else {
			rc, err := zipEntry.Open()
			if err != nil {
				lastErr = "failed to read zip entry " + zipEntry.Name + ": " + err.Error()
				continue
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				rc.Close()
				lastErr = "failed to create parent dir for " + zipEntry.Name
				continue
			}

			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				rc.Close()
				lastErr = "failed to create file " + zipEntry.Name + ": " + err.Error()
				continue
			}

			if _, err := io.Copy(outFile, rc); err != nil {
				rc.Close()
				outFile.Close()
				lastErr = "failed to write file " + zipEntry.Name + ": " + err.Error()
				continue
			}
			rc.Close()
			outFile.Close()

			mode := zipEntry.FileInfo().Mode().Perm()
			if mode != 0 {
				os.Chmod(targetPath, mode)
			}
		}
		extracted++
	}
	return extracted, lastErr
}

func extractTarGz(data []byte, baseDir, absRoot string) (int, string) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return 0, "invalid tar.gz: " + err.Error()
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	extracted := 0
	var lastErr string

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			lastErr = "failed to read tar: " + err.Error()
			break
		}

		if strings.Contains(hdr.Name, "..") {
			lastErr = "disallowed path in tar: " + hdr.Name
			continue
		}

		targetPath := filepath.Join(baseDir, hdr.Name)
		absTarget, _ := filepath.Abs(targetPath)
		if !strings.HasPrefix(absTarget, absRoot) {
			lastErr = "path outside root: " + hdr.Name
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, hdr.FileInfo().Mode().Perm()); err != nil {
				lastErr = "failed to create dir " + hdr.Name + ": " + err.Error()
			}
		case tar.TypeReg, tar.TypeRegA:
			parentDir := filepath.Dir(targetPath)
			if parentDir != "" && parentDir != "." {
				if err := os.MkdirAll(parentDir, 0755); err != nil {
					lastErr = "failed to create parent dir for " + hdr.Name
					continue
				}
			}

			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
			if err != nil {
				lastErr = "failed to create file " + hdr.Name + ": " + err.Error()
				continue
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				lastErr = "failed to write file " + hdr.Name + ": " + err.Error()
				continue
			}
			outFile.Close()
			os.Chmod(targetPath, hdr.FileInfo().Mode().Perm())
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil && !os.IsExist(err) {
				lastErr = "failed to create symlink " + hdr.Name + ": " + err.Error()
			}
		}
		extracted++
	}
	return extracted, lastErr
}

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kickcraft/api"
	"github.com/kickcraft/config"
	"github.com/kickcraft/logger"
	"github.com/kickcraft/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server represents the HTTP server
type Server struct {
	Engine     *gin.Engine
	options    config.ServerOptions
	httpServer *http.Server
}

// New creates a new server instance
func New(opts config.ServerOptions) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Apply middleware
	engine.Use(middleware.Recovery())
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.CORS())

	// Initialize API handler
	api.InitHandler(opts)

	// Register routes
	registerRoutes(engine, opts)

	return &Server{
		Engine:  engine,
		options: opts,
	}
}

// registerRoutes sets up all HTTP routes
func registerRoutes(engine *gin.Engine, opts config.ServerOptions) {
	// Health check
	engine.GET("/health", api.HealthCheck)

	// Serve static files
	engine.Static("/static", opts.StaticDir)
	engine.Static("/css", opts.StaticDir+"/css")
	engine.Static("/js", opts.StaticDir+"/js")
	engine.Static("/views", opts.StaticDir+"/views")

	// Serve index.html for root
	engine.GET("/", func(c *gin.Context) {
		c.File(opts.StaticDir + "/index.html")
	})

	// API routes
	apiGroup := engine.Group("/api")
	{
		// Config endpoints
		apiGroup.POST("/config/parse", api.ParseConfig)
		apiGroup.POST("/config/parse-ks", api.ParseKSFile)
		apiGroup.POST("/config/validate", api.ValidateConfig)
		apiGroup.POST("/config/generate", api.GenerateConfig)
		apiGroup.GET("/config/default", api.GetDefaultConfig)

		// Distro endpoints
		apiGroup.GET("/distros", api.GetDistros)
		apiGroup.GET("/distros/sources", api.GetISODownloadSources)

		// Host info endpoint
		apiGroup.GET("/host/info", api.GetHostInfo)

		// Template endpoints
		apiGroup.GET("/templates", api.GetTemplates)
		apiGroup.GET("/templates/:name", api.GetTemplate)
		apiGroup.POST("/templates", api.SaveTemplate)
		apiGroup.PUT("/templates/:name", api.UpdateTemplate)
		apiGroup.DELETE("/templates/:name", api.DeleteTemplate)

		// ISO endpoints
		apiGroup.POST("/iso/download", api.DownloadISO)
		apiGroup.POST("/iso/upload", api.UploadISO)
		apiGroup.POST("/iso/generate", api.GenerateISO)
		apiGroup.POST("/iso/reset", api.ResetBuildConfig)
		apiGroup.GET("/iso/status/:id", api.GetISOStatus)
		apiGroup.GET("/iso/download/:id", api.DownloadISOFile)

		// Package endpoints
		apiGroup.POST("/packages/search", api.SearchPackages)
		apiGroup.POST("/packages/download", api.DownloadPackages)

		// Embedded files endpoints
		apiGroup.POST("/embedded/write", api.WriteEmbeddedFile)
		apiGroup.DELETE("/embedded/delete", api.DeleteEmbeddedFile)
		apiGroup.POST("/embedded/mkdir", api.MkdirEmbedded)
		apiGroup.GET("/embedded/read", api.ReadEmbeddedFile)
		apiGroup.GET("/embedded/dir", api.ListDirectory)
		apiGroup.POST("/embedded/extract-zip", api.ExtractZipArchive)
	}

	// Swagger documentation
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// Run starts the HTTP server
func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:         ":" + s.options.Port,
		Handler:      s.Engine,
		ReadTimeout:  10 * time.Minute, // large ISO uploads can take several minutes
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("Server starting on port %s", s.options.Port)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

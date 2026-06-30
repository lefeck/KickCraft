package cmd

import (
	"fmt"
	"os"

	"github.com/kickcraft/config"
	"github.com/kickcraft/server"
	"github.com/spf13/cobra"
)

var (
	port         string
	staticDir    string
	templatesDir string
	logLevel     string
)

var rootCmd = &cobra.Command{
	Use:   "kickcraft",
	Short: "KickCraft - Rocky Linux Kickstart ISO Generator",
	Long: `KickCraft is a web-based tool for generating automated installation ISOs
for Rocky Linux using Kickstart configuration files.

Supported distributions:
  - Rocky Linux 8
  - Rocky Linux 9
  - Rocky Linux 10`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to listen on")
	rootCmd.Flags().StringVar(&staticDir, "static-dir", "./static", "Directory for static files")
	rootCmd.Flags().StringVar(&templatesDir, "templates-dir", "./templates", "Directory for templates")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Set defaults from environment variables
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	if envStatic := os.Getenv("STATIC_DIR"); envStatic != "" {
		staticDir = envStatic
	}
}

func startServer() {
	srv := server.New(config.ServerOptions{
		Port:         port,
		StaticDir:    staticDir,
		TemplatesDir: templatesDir,
	})

	fmt.Printf("KickCraft starting on http://0.0.0.0:%s\n", port)
	fmt.Printf("Static files: %s\n", staticDir)

	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

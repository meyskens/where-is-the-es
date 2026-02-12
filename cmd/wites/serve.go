package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	apiv1 "github.com/meyskens/where-is-the-es/pkg/api/v1"
	"github.com/spf13/cobra"
)

//go:embed all:frontend
var frontendFS embed.FS

func init() {
	rootCmd.AddCommand(NewServeCmd())
}

type serveCmdOptions struct {
	BindAddr string
	Port     int
	TCURL    string
}

// NewServeCmd generates the `serve` command
func NewServeCmd() *cobra.Command {
	s := serveCmdOptions{}
	c := &cobra.Command{
		Use:     "serve",
		Short:   "Serves the HTTP endpoints",
		Long:    `Serves the HTTP endpoints on the given bind address and port`,
		PreRunE: s.Validate,
		RunE:    s.RunE,
	}
	c.Flags().StringVarP(&s.BindAddr, "bind-address", "b", "0.0.0.0", "address to bind port to")
	c.Flags().IntVarP(&s.Port, "port", "p", 8080, "Port to listen on")
	c.Flags().StringVarP(&s.TCURL, "tc-url", "t", "", "address of API with more accurate composition")

	return c
}

func (s *serveCmdOptions) Validate(cmd *cobra.Command, args []string) error {
	return nil
}

func (s *serveCmdOptions) RunE(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Register API routes
	v1 := apiv1.New(s.TCURL)
	v1.Register(e)

	// Serve frontend static files
	frontendSubFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}

	// Serve static files
	assetHandler := http.FileServer(http.FS(frontendSubFS))
	e.GET("/", echo.WrapHandler(assetHandler))
	e.GET("/*", echo.WrapHandler(assetHandler))

	go func() {
		e.Start(fmt.Sprintf("%s:%d", s.BindAddr, s.Port))
		cancel() // server ended, stop the world
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	for {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			return nil
		}
	}
}

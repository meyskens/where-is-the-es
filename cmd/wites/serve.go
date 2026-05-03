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
	BindAddr          string
	Port              int
	TCURL             string
	DBAPIKey          string
	DBClientID        string
	NSSubscriptionKey string
	FlareSolverrURL   string
	GrapperURL        string
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
	c.Flags().StringVar(&s.DBAPIKey, "db-api-key", os.Getenv("DB_API_KEY"), "Deutsche Bahn RIS-Journeys API key (DB-Api-Key), defaults to $DB_API_KEY")
	c.Flags().StringVar(&s.DBClientID, "db-client-id", os.Getenv("DB_CLIENT_ID"), "Deutsche Bahn RIS-Journeys client ID (DB-Client-Id), defaults to $DB_CLIENT_ID")
	c.Flags().StringVar(&s.NSSubscriptionKey, "ns-subscription-key", os.Getenv("NS_SUBSCRIPTION_KEY"), "NS Reisinformatie API subscription key (Ocp-Apim-Subscription-Key), defaults to $NS_SUBSCRIPTION_KEY")
	c.Flags().StringVar(&s.FlareSolverrURL, "flaresolverr-url", os.Getenv("FLARESOLVERR_URL"), "FlareSolverr base URL used for NMBS realtime scraping, defaults to $FLARESOLVERR_URL")
	c.Flags().StringVar(&s.GrapperURL, "grapper-url", os.Getenv("GRAPPER_URL"), "Grapper base URL used for Czech realtime data, defaults to $GRAPPER_URL")

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
	v1 := apiv1.New(s.TCURL, s.DBAPIKey, s.DBClientID, s.NSSubscriptionKey, s.FlareSolverrURL, s.GrapperURL)
	v1.Register(e)

	// Serve frontend static files
	frontendSubFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return fmt.Errorf("failed to create sub filesystem: %w", err)
	}

	// Serve static files with SPA fallback for client-side routing
	e.GET("/*", func(c echo.Context) error {
		// Try to open the requested file
		path := c.Path()

		// Check if the file exists in the embedded filesystem
		f, err := frontendSubFS.Open(path)
		if err == nil {
			f.Close()
			// File exists, serve it
			assetHandler := http.FileServer(http.FS(frontendSubFS))
			assetHandler.ServeHTTP(c.Response(), c.Request())
			return nil
		}

		// File doesn't exist, serve index.html for SPA routing
		indexContent, err := fs.ReadFile(frontendSubFS, "index.html")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "index.html not found")
		}

		// Serve index.html
		c.Response().Header().Set("Content-Type", "text/html")
		c.Response().Write(indexContent)
		return nil
	})

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

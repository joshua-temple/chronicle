package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joshua-temple/chronicle/pkg/daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run Chronicle as a daemon with REST API",
	Long: `Run Chronicle as a background daemon with a REST API.

The daemon provides HTTP endpoints for:
- Starting and managing test runs
- Listing scenarios and components
- Viewing results
- Hot-reloading configuration

Example:
  chronicle daemon --addr :8080

Then interact with the API:
  curl http://localhost:8080/api/v1/health
  curl http://localhost:8080/api/v1/scenarios
  curl -X POST http://localhost:8080/api/v1/runs -d '{"scenario_name":"my-scenario"}'`,
	RunE: runDaemon,
}

func init() {
	daemonCmd.Flags().StringP("addr", "a", ":8080", "address to listen on")
	daemonCmd.Flags().String("api-key", "", "API key for authentication (generated if not provided)")
	daemonCmd.Flags().Bool("no-auth", false, "disable authentication (not recommended for production)")
	daemonCmd.Flags().Bool("watch", false, "watch for configuration changes and auto-reload")
	daemonCmd.Flags().Duration("watch-interval", 5*time.Second, "interval for checking configuration changes")
}

func runDaemon(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")
	apiKey, _ := cmd.Flags().GetString("api-key")
	noAuth, _ := cmd.Flags().GetBool("no-auth")
	watch, _ := cmd.Flags().GetBool("watch")
	watchInterval, _ := cmd.Flags().GetDuration("watch-interval")
	verbose := viper.GetBool("verbose")

	configFile := viper.GetString("config")
	if configFile == "" {
		configFile = "chronicle.yaml"
	}

	// Setup authentication
	var authOpts []daemon.ServerOption
	if noAuth {
		authOpts = append(authOpts, daemon.WithAuth(daemon.NewAuth(daemon.AuthConfig{
			Method: daemon.AuthMethodNone,
		})))
		if verbose {
			fmt.Println("Warning: Authentication disabled")
		}
	} else {
		authConfig := daemon.AuthConfig{Method: daemon.AuthMethodAPIKey}
		if apiKey != "" {
			authConfig.APIKey = apiKey
		}
		auth := daemon.NewAuth(authConfig)
		authOpts = append(authOpts, daemon.WithAuth(auth))
		if verbose {
			fmt.Printf("API Key: %s\n", auth.GetAPIKey())
		}
	}

	// Create server
	server, err := daemon.NewServer(configFile, authOpts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("Starting Chronicle daemon on %s\n", addr)
		if verbose {
			fmt.Println("Endpoints:")
			fmt.Println("  GET  /api/v1/health       - Health check")
			fmt.Println("  POST /api/v1/runs         - Start a run")
			fmt.Println("  GET  /api/v1/runs         - List runs")
			fmt.Println("  GET  /api/v1/runs/{id}    - Get run details")
			fmt.Println("  DELETE /api/v1/runs/{id}  - Cancel run")
			fmt.Println("  GET  /api/v1/scenarios    - List scenarios")
			fmt.Println("  GET  /api/v1/scenarios/{name} - Get scenario")
			fmt.Println("  GET  /api/v1/components   - List components")
			fmt.Println("  GET  /api/v1/components/{name} - Get component")
			fmt.Println("  GET  /api/v1/results      - List results")
			fmt.Println("  GET  /api/v1/results/{id} - Get result")
			fmt.Println("  DELETE /api/v1/results/{id} - Delete result")
			fmt.Println("  GET  /api/v1/config       - Get config")
			fmt.Println("  POST /api/v1/config/reload - Reload config")
		}
		if err := server.Start(addr); err != nil {
			errCh <- err
		}
	}()

	// Setup config watcher if enabled
	if watch {
		watcher := daemon.NewConfigWatcher(
			[]string{configFile},
			func() error {
				if verbose {
					fmt.Println("Configuration changed, reloading...")
				}
				return server.ReloadConfig()
			},
			daemon.WithInterval(watchInterval),
		)
		go func() {
			if err := watcher.Start(context.Background()); err != nil {
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}()
	}

	// Wait for shutdown
	select {
	case sig := <-sigCh:
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	fmt.Println("Daemon stopped")
	return nil
}

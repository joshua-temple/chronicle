package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/joshua-temple/chronicle/pkg/ui"
	"github.com/joshua-temple/chronicle/pkg/ui/standalone"
	"github.com/spf13/cobra"
)

// NewUICmd creates the ui command.
func NewUICmd() *cobra.Command {
	var port int
	var dir string
	var noBrowser bool
	var standaloneMode bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the Chronicle UI for editing configuration",
		Long: `Launch a local web server that serves the Chronicle UI.

The UI allows you to:
- Edit chronicle.yaml configuration
- Build and modify scenarios
- Browse discovered components

In standalone mode (--standalone), the UI serves as a multi-project
control center for managing multiple Chronicle projects.

Example:
  chronicle ui
  chronicle ui --port 8080
  chronicle ui --dir ./my-project
  chronicle ui --standalone --port 3001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI(port, dir, noBrowser, standaloneMode)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to serve on")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Project directory")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")
	cmd.Flags().BoolVar(&standaloneMode, "standalone", false, "Run as multi-project control center")

	return cmd
}

func runUI(port int, dir string, noBrowser bool, standaloneMode bool) error {
	if standaloneMode {
		// Standalone mode manages its own projects from ~/.chronicle/projects.json
		// The dir parameter is intentionally not used in this mode
		return runStandalone(port, noBrowser)
	}

	// Single-project mode: validate directory exists
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", absDir)
	}

	server := ui.New(
		ui.WithPort(port),
		ui.WithDir(absDir),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	cleanup := setupSignalHandler(cancel)
	defer cleanup()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Chronicle UI available at %s\n", url)
	fmt.Printf("Project directory: %s\n", absDir)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	if !noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func runStandalone(port int, noBrowser bool) error {
	server, err := standalone.NewServer(standalone.WithPort(port))
	if err != nil {
		return fmt.Errorf("failed to create standalone server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	cleanup := setupSignalHandler(cancel)
	defer cleanup()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Chronicle Control Center available at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	if !noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// setupSignalHandler sets up signal handling for graceful shutdown.
// Returns a cleanup function that must be called to stop signal notifications.
func setupSignalHandler(cancel context.CancelFunc) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()
	return func() { signal.Stop(sigCh) }
}

// openBrowser attempts to open the given URL in the default browser.
// Returns an error if the browser cannot be opened.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	return nil
}

package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/joshua-temple/chronicle/pkg/ui"
	"github.com/spf13/cobra"
)

// NewUICmd creates the ui command.
func NewUICmd() *cobra.Command {
	var port int
	var dir string
	var noBrowser bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the Chronicle UI for editing configuration",
		Long: `Launch a local web server that serves the Chronicle UI.

The UI allows you to:
- Edit chronicle.yaml configuration
- Build and modify scenarios
- Browse discovered components

Example:
  chronicle ui
  chronicle ui --port 8080
  chronicle ui --dir ./my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI(port, dir, noBrowser)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to serve on")
	cmd.Flags().StringVarP(&dir, "dir", "d", ".", "Project directory")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")

	return cmd
}

func runUI(port int, dir string, noBrowser bool) error {
	server := ui.New(
		ui.WithPort(port),
		ui.WithDir(dir),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Chronicle UI available at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// Open browser
	if !noBrowser {
		go openBrowser(url)
	}

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	_ = cmd.Start()
}

package metrics

import (
	"context"
	"fmt"
	"io"
	"os"
)

// ConsoleCollector prints events to stdout (for development)
type ConsoleCollector struct {
	verbose bool
	writer  io.Writer
}

// NewConsoleCollector creates a console collector
func NewConsoleCollector(verbose bool) *ConsoleCollector {
	return &ConsoleCollector{verbose: verbose, writer: os.Stdout}
}

// NewConsoleCollectorWithWriter creates a console collector with custom writer
func NewConsoleCollectorWithWriter(verbose bool, writer io.Writer) *ConsoleCollector {
	return &ConsoleCollector{verbose: verbose, writer: writer}
}

// Collect prints the event to the console
func (c *ConsoleCollector) Collect(ctx context.Context, event Event) error {
	switch e := event.(type) {
	case *ScenarioStarted:
		if c.verbose {
			fmt.Fprintf(c.writer, "[METRIC] scenario_started name=%s\n", e.ScenarioName)
		}
	case *ScenarioCompleted:
		fmt.Fprintf(c.writer, "[METRIC] scenario=%s duration=%s status=%v\n",
			e.ScenarioName, e.Duration, e.Status)
	case *ComponentCompleted:
		if c.verbose {
			fmt.Fprintf(c.writer, "[METRIC] component=%s duration=%s status=%v\n",
				e.ComponentName, e.Duration, e.Status)
		}
	case *TestCompleted:
		fmt.Fprintf(c.writer, "[METRIC] test=%s duration=%s status=%v retries=%d\n",
			e.TestID, e.Duration, e.Status, e.Retries)
	case *SuiteCompleted:
		fmt.Fprintf(c.writer, "[METRIC] suite=%s duration=%s passed=%d failed=%d skipped=%d\n",
			e.SuiteID, e.Duration, e.Passed, e.Failed, e.Skipped)
	case *ServiceStarted:
		if c.verbose {
			fmt.Fprintf(c.writer, "[METRIC] service_started name=%s startup=%s\n",
				e.ServiceName, e.Duration)
		}
	case *ServiceStopped:
		if c.verbose {
			fmt.Fprintf(c.writer, "[METRIC] service_stopped name=%s uptime=%s\n",
				e.ServiceName, e.Uptime)
		}
	case *Custom:
		fmt.Fprintf(c.writer, "[METRIC] %s=%.2f%s\n", e.Name, e.Value, e.Unit)
	}
	return nil
}

// Flush is a no-op for console collector
func (c *ConsoleCollector) Flush(ctx context.Context) error {
	return nil
}

// Close is a no-op for console collector
func (c *ConsoleCollector) Close() error {
	return nil
}

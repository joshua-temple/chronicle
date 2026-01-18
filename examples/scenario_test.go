package examples

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joshua-temple/chronicle/core"
	"github.com/joshua-temple/chronicle/daemon"
	"github.com/joshua-temple/chronicle/infrastructure"
	"github.com/joshua-temple/chronicle/scenarios"
)

// TestScenarioExample demonstrates how to use the Scenario Builder pattern
func TestScenarioExample(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Define a test setup component
	setupDatabase := func(ctx *scenarios.Context) error {
		ctx.Info("Setting up database...")
		ctx.Set("db_connection", "redis://localhost:6379")
		return nil
	}

	// Define test components
	createUser := func(ctx *scenarios.Context) error {
		ctx.Info("Creating user...")
		// User parameters from the scenario context
		username, _ := ctx.GetParameter("username")
		ctx.Info("Creating user: %v", username)
		return nil
	}

	validateUser := func(ctx *scenarios.Context) error {
		ctx.Info("Validating user...")
		username, _ := ctx.GetParameter("username")
		ctx.Info("User %v is valid", username)
		return nil
	}

	// Define a test teardown component
	cleanupResources := func(ctx *scenarios.Context) error {
		ctx.Info("Cleaning up resources...")
		return nil
	}

	// Build a scenario using the builder pattern
	scenario := scenarios.NewBuilder("User Management Test").
		WithDescription("Tests user creation and validation").
		WithTimeout(30*time.Second).
		SetupWith(setupDatabase).
		AddComponent("CreateUser", createUser).
		AddComponent("ValidateUser", validateUser).
		TeardownWith(cleanupResources).
		WithStringParameter(
			"username",
			"Username for testing",
			5,
			10,
			"abcdefghijklmnopqrstuvwxyz",
		).
		Build()

	// Create a standalone context for running the scenario
	ctx := scenarios.NewContext()

	// Execute the scenario
	err := scenario.Execute(ctx)
	if err != nil {
		t.Fatalf("Scenario execution failed: %v", err)
	}
}

// TestScenarioWithInfrastructure demonstrates using a scenario with TestContainers
func TestScenarioWithInfrastructure(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Define infrastructure requirements
	redisService := infrastructure.NewServiceRequirement("redis", "redis:latest")
	infrastructure.AddPort(&redisService, "default", 6379)
	infrastructure.BuildPortWaitStrategy(&redisService, 6379, 10*time.Second)

	// Create infrastructure provider (TestContainers)
	provider := infrastructure.NewTestContainersProvider()
	provider.AddService(redisService)

	// Initialize context with test.T
	tc := core.NewTestContext(t)
	tc.InfraProvider = provider

	// Initialize infrastructure
	bgCtx := context.Background()
	if err := provider.Initialize(bgCtx); err != nil {
		t.Fatalf("Failed to initialize infrastructure: %v", err)
	}
	defer provider.Stop(bgCtx)

	// Start infrastructure
	if err := provider.Start(bgCtx); err != nil {
		t.Fatalf("Failed to start infrastructure: %v", err)
	}

	// Create a scenario context from the core context
	scenarioCtx := scenarios.WithCoreContext(tc)

	// Define a component that uses the infrastructure
	connectToRedis := func(ctx *scenarios.Context) error {
		if tc := ctx.ParentContext(); tc != nil {
			endpoint, err := tc.InfraProvider.ServiceEndpoint("redis", "default")
			if err != nil {
				return fmt.Errorf("failed to get Redis endpoint: %w", err)
			}
			ctx.Info("Connected to Redis at %s", endpoint)
		}
		return nil
	}

	// Build a simple scenario
	scenario := scenarios.NewBuilder("Redis Connectivity Test").
		AddComponent("ConnectToRedis", connectToRedis).
		Build()

	// Execute the scenario
	err := scenario.Execute(scenarioCtx)
	if err != nil {
		t.Fatalf("Scenario execution failed: %v", err)
	}
}

// Example_daemonUsage demonstrates how to set up and run the daemon service
func Example_daemonUsage() {
	// Create a component registry
	registry := daemon.NewComponentRegistry()

	// Register some test components
	registry.RegisterComponent("CreateUser", func(ctx *scenarios.Context) error {
		username, _ := ctx.GetParameter("username")
		ctx.Info("Creating user %s", username)
		return nil
	})

	registry.RegisterComponent("ValidateUser", func(ctx *scenarios.Context) error {
		username, _ := ctx.GetParameter("username")
		ctx.Info("Validating user %s", username)
		return nil
	})

	registry.RegisterComponent("DeleteUser", func(ctx *scenarios.Context) error {
		username, _ := ctx.GetParameter("username")
		ctx.Info("Deleting user %s", username)
		return nil
	})

	// Define daemon configuration
	config := daemon.DaemonConfig{
		TestSets: []daemon.TestSet{
			{
				Name:              "UserManagement",
				Components:        []string{"CreateUser", "ValidateUser", "DeleteUser"},
				SelectionStrategy: daemon.RandomSelection,
				ExecutionWeight:   1,
				ParameterGenerators: map[string]scenarios.Generator{
					"username": &scenarios.StringGenerator{
						MinLength: 5,
						MaxLength: 10,
						Charset:   "abcdefghijklmnopqrstuvwxyz",
					},
				},
			},
		},
		MaxConcurrency:    2,
		ExecutionInterval: 500 * time.Millisecond,
		TotalRunTime:      1 * time.Minute,
		ReportInterval:    5 * time.Second,
		FailureBehavior:   daemon.Continue,
		OutputHandlers: []daemon.OutputHandler{
			&ConsoleOutputHandler{},
		},
	}

	// Create and start the daemon
	d := daemon.NewDaemon(config)

	// Register components from the registry
	for _, name := range registry.ListComponents() {
		comp, _ := registry.GetComponent(name)
		d.RegisterComponent(name, comp)
	}

	// Start the daemon
	if err := d.Start(); err != nil {
		fmt.Printf("Failed to start daemon: %v\n", err)
		return
	}

	// Daemon will run for the configured time (1 minute)
	// Wait for it to complete
	time.Sleep(config.TotalRunTime)

	// Stop can be called explicitly, but in this case the daemon
	// will stop automatically after the total run time
	// d.Stop()

	fmt.Println("Daemon completed execution")
}

// ConsoleOutputHandler is a simple output handler that writes to the console
type ConsoleOutputHandler struct{}

// HandleResult processes a test result
func (h *ConsoleOutputHandler) HandleResult(result *daemon.ScenarioResult) {
	status := "SUCCEEDED"
	if !result.Success {
		status = "FAILED"
	}
	fmt.Printf("[%s] %s (%s): %v\n", status, result.Scenario.Name, result.Duration, result.Parameters)
}

// Flush ensures all results are processed
func (h *ConsoleOutputHandler) Flush() {
	// Nothing to flush for console output
}

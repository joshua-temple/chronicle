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
	"github.com/joshua-temple/chronicle/suite"
)

// Define some typed identifiers
const (
	// Service IDs
	RedisService core.ServiceID = "infrastructure.redis"

	// Test IDs
	UserCreationTest core.TestID = "user.creation"
	UserAuthTest     core.TestID = "user.auth"

	// Component IDs
	SetupDatabaseComp  core.ComponentID = "components.setup.database"
	CreateUserComp     core.ComponentID = "components.user.create"
	ValidateUserComp   core.ComponentID = "components.user.validate"
	AuthenticateComp   core.ComponentID = "components.user.authenticate"
	TeardownSystemComp core.ComponentID = "components.teardown.system"
)

// TestEnhancedSuiteExample demonstrates how to use the Suite pattern with typed IDs
func TestEnhancedSuiteExample(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Create a test registry to register our test components and types
	registry := core.NewIDRegistry()

	// Register IDs to enable validation
	registry.RegisterServiceID(RedisService)
	registry.RegisterTestID(UserCreationTest)
	registry.RegisterTestID(UserAuthTest)
	registry.RegisterComponentID(SetupDatabaseComp)
	registry.RegisterComponentID(CreateUserComp)
	registry.RegisterComponentID(ValidateUserComp)
	registry.RegisterComponentID(AuthenticateComp)
	registry.RegisterComponentID(TeardownSystemComp)

	// Define test components with typed IDs
	setupDatabase := func(ctx *scenarios.Context) error {
		ctx.Info("Setting up database...")
		ctx.Set("db_connection", "redis://localhost:6379")
		return nil
	}

	createUser := func(ctx *scenarios.Context) error {
		ctx.Info("Creating user...")
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

	authenticateUser := func(ctx *scenarios.Context) error {
		ctx.Info("Authenticating user...")
		username, _ := ctx.GetParameter("username")
		ctx.Info("User %v authenticated", username)
		return nil
	}

	teardownSystem := func(ctx *scenarios.Context) error {
		ctx.Info("Tearing down system...")
		return nil
	}

	// Define infrastructure requirements
	redisService := infrastructure.NewServiceRequirement(string(RedisService), "redis:latest")
	infrastructure.AddPort(&redisService, "default", 6379)
	infrastructure.BuildPortWaitStrategy(&redisService, 6379, 10*time.Second)

	// Create infrastructure provider
	provider := infrastructure.NewTestContainersProvider()
	provider.AddService(redisService)

	// Create a suite
	testSuite := suite.NewSuite(core.SuiteID("user-suite"))
	testSuite.WithDescription("User Management Test Suite")
	testSuite.WithInfraProvider(provider)
	testSuite.WithGlobalSetup(setupDatabase)
	testSuite.WithGlobalTeardown(teardownSystem)

	// Register components with the suite
	testSuite.RegisterComponent(SetupDatabaseComp, setupDatabase)
	testSuite.RegisterComponent(CreateUserComp, createUser)
	testSuite.RegisterComponent(ValidateUserComp, validateUser)
	testSuite.RegisterComponent(AuthenticateComp, authenticateUser)
	testSuite.RegisterComponent(TeardownSystemComp, teardownSystem)

	// Create tests
	userCreationTest := suite.NewTest(UserCreationTest)
	userCreationTest.WithDescription("Tests user creation")
	userCreationTest.WithRequiredServices(RedisService)
	userCreationTest.WithComponents(CreateUserComp, ValidateUserComp)
	userCreationTest.WithParameter("username", "testuser")
	userCreationTest.WithTimeout(5 * time.Second)

	userAuthTest := suite.NewTest(UserAuthTest)
	userAuthTest.WithDescription("Tests user authentication")
	userAuthTest.WithRequiredServices(RedisService)
	userAuthTest.WithComponents(AuthenticateComp)
	userAuthTest.WithDependencies(UserCreationTest) // This test depends on user creation
	userAuthTest.WithParameter("username", "testuser")
	userAuthTest.WithRetryPolicy(&suite.RetryPolicy{
		MaxAttempts: 3,
		BackoffStrategy: &suite.ExponentialBackoff{
			InitialDelay: 100 * time.Millisecond,
			Factor:       2.0,
			MaxDelay:     1 * time.Second,
		},
		RetryCondition: func(err error) bool {
			// Retry on all errors
			return err != nil
		},
	})

	// Add tests to the suite
	testSuite.AddTests(userCreationTest, userAuthTest)

	// Run the suite
	err := testSuite.Run(context.Background())
	if err != nil {
		t.Fatalf("Suite execution failed: %v", err)
	}
}

// TestEnhancedDaemonExample demonstrates how to use the enhanced daemon configuration
func TestEnhancedDaemonExample(t *testing.T) {
	// Skip if not running as part of the example suite
	if testing.Short() {
		t.Skip("Skipping example test in short mode")
	}

	// Create an enhanced daemon configuration
	config := daemon.NewEnhancedDaemonConfig()

	// Set basic execution parameters
	config.MaxConcurrency = 2
	config.ShutdownGracePeriod = 5 * time.Second
	config.GlobalThrottleRate = 10 // Max 10 test starts per minute

	// Set resource constraints
	config.MaxResourceAllocation = daemon.ResourceUsage{
		CPU:    50,   // 50% CPU max
		Memory: 1024, // 1GB RAM max
		Disk:   5120, // 5GB disk max
	}

	// Configure chaos testing (low intensity)
	config.ChaosLevel = 0.1 // 10% chance of chaos
	config.ChaosFeatures = []daemon.ChaosFeature{
		daemon.NetworkLatencyChaos,
		daemon.ServiceRestartChaos,
	}

	// Create test configurations
	userTestConfig := &daemon.TestExecutionConfig{
		Frequency:       5 * time.Minute,
		MaxParallelSelf: 1,
		Jitter:          30 * time.Second,
		TimeWindows: []daemon.TimeWindow{
			{
				DaysOfWeek: []time.Weekday{
					time.Monday, time.Tuesday, time.Wednesday,
					time.Thursday, time.Friday,
				},
				StartTime: "09:00",
				EndTime:   "17:00",
				TimeZone:  "America/New_York",
			},
		},
		CooldownPeriod:         30 * time.Second,
		ExecutionQuota:         100,
		QuotaPeriod:            24 * time.Hour,
		MaxConsecutiveFailures: 5,
		QuarantineDuration:     1 * time.Hour,
	}

	// Add test configs to the daemon
	config.TestConfigs[UserCreationTest] = userTestConfig

	// Create a test store for the daemon
	store := daemon.NewTestScenarioStore()

	// Register components using typed IDs
	setupDatabase := func(ctx *scenarios.Context) error {
		ctx.Info("Setting up database...")
		return nil
	}

	createUser := func(ctx *scenarios.Context) error {
		username, _ := ctx.GetParameter("username")
		ctx.Info("Creating user %s", username)
		return nil
	}

	validateUser := func(ctx *scenarios.Context) error {
		username, _ := ctx.GetParameter("username")
		ctx.Info("Validating user %s", username)
		return nil
	}

	store.RegisterComponent(core.ComponentID("SetupDatabase"), setupDatabase)
	store.RegisterComponent(core.ComponentID("CreateUser"), createUser)
	store.RegisterComponent(core.ComponentID("ValidateUser"), validateUser)

	// Create a scenario and register it
	builder := scenarios.NewBuilder("User Creation Test")
	builder.AddComponent("SetupDatabase", setupDatabase)
	builder.AddComponent("CreateUser", createUser)
	builder.AddComponent("ValidateUser", validateUser)
	builder.WithStringParameter(
		"username",
		"Username for testing",
		5, 10,
		"abcdefghijklmnopqrstuvwxyz",
	)

	// Register the scenario
	store.RegisterScenario(UserCreationTest, builder.Build())

	// Create and configure scheduler
	logger := daemon.NewDefaultLogger()
	scheduler := daemon.NewExecutionScheduler(config, store, logger)

	// In a real daemon, we would start the scheduler
	// and let it run for a while
	scheduler.Start()
	time.Sleep(100 * time.Millisecond) // Just for demonstration
	scheduler.Stop()

	fmt.Println("Enhanced daemon example completed")
}

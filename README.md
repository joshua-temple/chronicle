# Chronicle - Testing Framework

Chronicle is a comprehensive testing framework for Golang that enables robust integration testing, infrastructure management, and continuous scenario-based testing.

## Core Features

- **Infrastructure Management**: Provision and manage test infrastructure using real containers
- **Scenario-Based Testing**: Build and compose test scenarios using a fluent builder pattern
- **Continuous Testing**: Run scenarios continuously with randomized inputs for chaos testing
- **Extensible Architecture**: Easily extend with custom providers, components, and generators
- **Suite & Test Separation**: Clear separation between suite-level and test-level constructs
- **Type-Safe Identifiers**: Strongly-typed identifiers for tests, services, and components

## Getting Started

### Installation

```bash
go get github.com/joshua-temple/chronicle
```

### Basic Usage

```go
import (
    "testing"
    "github.com/joshua-temple/chronicle/core"
    "github.com/joshua-temple/chronicle/infrastructure"
    "github.com/joshua-temple/chronicle/runner"
)

func TestExample(t *testing.T) {
    // Define infrastructure requirements
    redisService := infrastructure.NewServiceRequirement("redis", "redis:latest")
    infrastructure.AddPort(&redisService, "default", 6379)
    
    // Create infrastructure provider
    provider := infrastructure.NewTestContainersProvider()
    provider.AddService(redisService)
    
    // Run test with infrastructure
    runner.RunTest(t, func(ctx context.Context, tc *core.TestContext) error {
        // Your test code here
        return nil
    }, runner.WithInfrastructureProvider(provider))
}
```

## Components

### Core

The `core` package provides the foundation of the framework with types and interfaces that define test contexts, infrastructure providers, services, and more.

#### Type-Safe Identifiers

Chronicle supports strongly-typed identifiers for tests, services, components, and more:

```go
// Define typed identifiers
const (
    // Service IDs
    RedisService core.ServiceID = "infrastructure.redis"
    
    // Test IDs
    UserCreationTest core.TestID = "user.creation"
    
    // Component IDs
    SetupDatabaseComp core.ComponentID = "components.setup.database"
)

// Use typed identifiers
registry := core.NewIDRegistry()
registry.RegisterServiceID(RedisService)
registry.RegisterTestID(UserCreationTest)
registry.RegisterComponentID(SetupDatabaseComp)
```

### Infrastructure Providers

Chronicle supports multiple infrastructure providers:

#### TestContainers

The TestContainers provider integrates with the [testcontainers-go](https://github.com/testcontainers/testcontainers-go) library to provide real container-based infrastructure for tests.

```go
// Create TestContainers provider
provider := infrastructure.NewTestContainersProvider()

// Define and add service requirements
redisService := infrastructure.NewServiceRequirement("redis", "redis:latest")
infrastructure.AddPort(&redisService, "default", 6379)
provider.AddService(redisService)

// Initialize and start infrastructure
provider.Initialize(ctx)
provider.Start(ctx)
```

#### Docker Compose

For more complex infrastructure needs, Chronicle provides a Docker Compose provider:

```go
// Create Docker Compose provider
provider := infrastructure.NewDockerComposeProvider("./docker-compose.yml", "my-project")
```

### Scenarios Framework

The Scenarios framework allows for composition of test components into reusable scenarios:

```go
import (
    "github.com/joshua-temple/chronicle/scenarios"
)

// Define components
setupDatabase := func(ctx *scenarios.Context) error {
    ctx.Info("Setting up database...")
    return nil
}

createUser := func(ctx *scenarios.Context) error {
    username, _ := ctx.GetParameter("username")
    ctx.Info("Creating user: %v", username)
    return nil
}

// Build scenario
scenario := scenarios.NewBuilder("User Management Test").
    WithDescription("Tests user creation and validation").
    WithTimeout(30*time.Second).
    SetupWith(setupDatabase).
    AddComponent("CreateUser", createUser).
    WithStringParameter(
        "username", 
        "Username for testing",
        5, 10, 
        "abcdefghijklmnopqrstuvwxyz",
    ).
    Build()

// Execute the scenario
ctx := scenarios.NewContext()
err := scenario.Execute(ctx)
```

### Suite Framework

The Suite framework provides a way to manage groups of related tests with shared infrastructure:

```go
import (
    "github.com/joshua-temple/chronicle/suite"
    "github.com/joshua-temple/chronicle/core"
)

// Create a suite
testSuite := suite.NewSuite(core.SuiteID("user-suite"))
testSuite.WithDescription("User Management Suite")
testSuite.WithInfraProvider(provider)
testSuite.WithGlobalSetup(setupDatabase)
testSuite.WithGlobalTeardown(cleanupDatabase)

// Register components with the suite
testSuite.RegisterComponent(core.ComponentID("setup"), setupDatabase)
testSuite.RegisterComponent(core.ComponentID("create"), createUser)

// Create tests
userCreationTest := suite.NewTest(core.TestID("user.create"))
userCreationTest.WithComponents(core.ComponentID("create"))
userCreationTest.WithStateAccess(suite.ReadWrite)

// Add tests to the suite
testSuite.AddTests(userCreationTest)

// Run the suite
err := testSuite.Run(context.Background())
```

### Daemon Service

The Daemon service provides continuous testing capabilities with enhanced configuration:

```go
import (
    "github.com/joshua-temple/chronicle/daemon"
)

// Create enhanced daemon configuration
config := daemon.NewEnhancedDaemonConfig()
config.MaxConcurrency = 5
config.GlobalThrottleRate = 30 // Max 30 test starts per minute
config.ShutdownGracePeriod = 30 * time.Second

// Configure time windows
userTestConfig := &daemon.TestExecutionConfig{
    Frequency:       5 * time.Minute,
    MaxParallelSelf: 1,
    Jitter:          30 * time.Second,
    TimeWindows: []daemon.TimeWindow{
        {
            DaysOfWeek: []time.Weekday{time.Monday, time.Friday},
            StartTime:  "09:00",
            EndTime:    "17:00",
            TimeZone:   "America/New_York",
        },
    },
}

// Add test config
config.TestConfigs[core.TestID("user.creation")] = userTestConfig

// Create and start the scheduler
store := daemon.NewTestScenarioStore()
logger := daemon.NewDefaultLogger()
scheduler := daemon.NewExecutionScheduler(config, store, logger)
scheduler.Start()
```

## Advanced Features

### Parameter Generation

Chronicle provides built-in generators for test parameters:

- `StringGenerator`: Generates random strings with configurable length and character set
- `IntGenerator`: Generates random integers within a specified range
- `EnumGenerator`: Selects random values from a predefined set of options

Custom generators can be created by implementing the `Generator` interface.

### Conditional Component Execution

Components can be conditionally executed based on runtime conditions:

```go
// Add conditional component
scenario.WithConditionalComponent(
    "PaymentProcessor",
    func(ctx *scenarios.Context) bool {
        paymentType, _ := ctx.GetParameter("paymentType")
        return paymentType == "credit_card"
    },
    processCreditCardPayment,
)
```

### Result Reporting

The daemon service supports customizable result reporting:

```go
// Create custom output handler
type MyOutputHandler struct{}

func (h *MyOutputHandler) HandleResult(result *daemon.ScenarioResult) {
    // Process and store/display results
}

func (h *MyOutputHandler) Flush() {
    // Ensure all results are processed
}

// Add to daemon config
config.OutputHandlers = []daemon.OutputHandler{
    &MyOutputHandler{},
}
```

### Suite and Test Separation

Chronicle clearly separates suite-level and test-level concerns:

- **Suite**: Manages infrastructure, coordinates tests, and provides shared state
- **Test**: Focuses on specific functionality with clear dependencies

Benefits:
- Improved organization of related tests
- Centralized management of infrastructure
- Coordinated setup and teardown
- Dependency-based test ordering
- Controlled state sharing between tests

### Enhanced Daemon Features

The enhanced daemon configuration provides:

- **Time Windows**: Run tests only during specific days and times
- **Jitter**: Add randomness to scheduling to prevent resource contention
- **Resource Limits**: Control CPU, memory, and disk usage
- **Quota Management**: Limit test execution frequency
- **Quarantine**: Automatically disable tests after consecutive failures
- **Chaos Testing**: Introduce controlled chaos to test resilience

## Example

See the `examples/` directory for complete usage examples, including:
- `examples/simple_test.go` - Basic usage
- `examples/scenario_test.go` - Scenario-based testing
- `examples/enhanced_test.go` - Suite pattern and enhanced daemon

## License

This project is licensed under the terms specified in the LICENSE file.

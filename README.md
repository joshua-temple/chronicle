# Chronicle - Testing Framework

Chronicle is a comprehensive testing framework for Golang that enables robust integration testing, infrastructure management, and continuous scenario-based testing.

## Core Features

- **Infrastructure Management**: Provision and manage test infrastructure using real containers
- **Scenario-Based Testing**: Build and compose test scenarios using a fluent builder pattern
- **Continuous Testing**: Run scenarios continuously with randomized inputs for chaos testing
- **Extensible Architecture**: Easily extend with custom providers, components, and generators

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

### Daemon Service

The Daemon service provides continuous testing capabilities:

```go
import (
    "github.com/joshua-temple/chronicle/daemon"
)

// Create component registry
registry := daemon.NewComponentRegistry()

// Register test components
registry.RegisterComponent("CreateUser", createUserComponent)
registry.RegisterComponent("ValidateUser", validateUserComponent)

// Configure the daemon
config := daemon.DaemonConfig{
    TestSets: []daemon.TestSet{
        {
            Name:              "UserManagement",
            Components:        []string{"CreateUser", "ValidateUser"},
            SelectionStrategy: daemon.RandomSelection,
            ExecutionWeight:   1,
            ParameterGenerators: map[string]scenarios.Generator{
                "username": &scenarios.StringGenerator{
                    MinLength: 5,
                    MaxLength: 10,
                },
            },
        },
    },
    MaxConcurrency:    2,
    ExecutionInterval: 500 * time.Millisecond,
    TotalRunTime:      1 * time.Minute,
}

// Create and start the daemon
d := daemon.NewDaemon(config)

// Register components
for _, name := range registry.ListComponents() {
    comp, _ := registry.GetComponent(name)
    d.RegisterComponent(name, comp)
}

// Start the daemon
d.Start()
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

## Example

See the `examples/` directory for complete usage examples.

## License

This project is licensed under the terms specified in the LICENSE file.

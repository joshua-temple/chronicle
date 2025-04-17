# Chronicle Framework Documentation

## Overview
- [Introduction](README.md)
- [Getting Started](README.md#getting-started)

## Core Components
- [Test Context](core/types.go)
- [Infrastructure Providers](infrastructure/)
  - [TestContainers Provider](infrastructure/testcontainers.go)
  - [Docker Compose Provider](infrastructure/dockercompose.go)
- [Scenarios Framework](scenarios/)
  - [Scenario Builder](scenarios/builder.go)
  - [Components](scenarios/types.go)
- [Daemon Service](daemon/)
  - [Continuous Testing](daemon/daemon.go)
  
## Examples
- [Simple Test](examples/simple_test.go)
- [Scenario Test](examples/scenario_test.go)

## Advanced Topics
- [Parameter Generators](README.md#parameter-generation)
- [Conditional Execution](README.md#conditional-component-execution)
- [Result Reporting](README.md#result-reporting)

## API Reference
- [Core API](core/types.go)
- [Infrastructure API](infrastructure/provider.go)
- [Scenarios API](scenarios/types.go)
- [Daemon API](daemon/daemon.go)

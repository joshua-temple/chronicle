# Chaos Testing

Chronicle includes built-in chaos engineering capabilities for testing system resilience. Inject faults like network latency, packet loss, and resource constraints to verify your application handles failures gracefully.

## Chaos Profiles

Define chaos profiles in `chronicle.yaml`:

```yaml
chaos_profiles:
  network_latency:
    description: Simulate slow network conditions
    network:
      latency:
        enabled: true
        min: 100ms
        max: 500ms
        jitter: 0.1

  unstable_network:
    description: Simulate unreliable network
    network:
      packet_loss:
        enabled: true
        percentage: 10
      latency:
        enabled: true
        min: 50ms
        max: 200ms
```

## Network Chaos

### Latency Injection

Add delays to network calls:

```yaml
chaos_profiles:
  slow_network:
    network:
      latency:
        enabled: true
        min: 100ms      # Minimum delay
        max: 500ms      # Maximum delay
        jitter: 0.1     # Random variation (0-1)
```

### Packet Loss

Simulate unreliable connections:

```yaml
chaos_profiles:
  lossy_network:
    network:
      packet_loss:
        enabled: true
        percentage: 5   # 5% of packets dropped
```

### Network Partitions

Simulate network splits:

```yaml
chaos_profiles:
  partition:
    network:
      partition:
        enabled: true
        duration: 30s
        targets: [database, cache]  # Services to isolate
```

## Resource Chaos

### CPU Stress

Simulate high CPU load:

```yaml
chaos_profiles:
  cpu_stress:
    resource:
      cpu:
        enabled: true
        percentage: 80   # 80% CPU usage
        duration: 30s
```

### Memory Pressure

Simulate memory constraints:

```yaml
chaos_profiles:
  memory_pressure:
    resource:
      memory:
        enabled: true
        percentage: 70   # Use 70% of available memory
        duration: 30s
```

### I/O Slowdown

Simulate slow disk operations:

```yaml
chaos_profiles:
  slow_disk:
    resource:
      io:
        enabled: true
        percentage: 50   # 50% I/O throughput reduction
        duration: 30s
```

## Custom Chaos

Define custom fault injection:

```yaml
chaos_profiles:
  custom_chaos:
    custom:
      before:
        - name: setup_fault
          command: ./scripts/inject-fault.sh
          args: [--type, network, --target, api]
      during:
        - name: monitor_fault
          command: ./scripts/monitor.sh
      after:
        - name: cleanup_fault
          command: ./scripts/cleanup-fault.sh
```

## Selectors

Control which targets receive chaos:

### All Selector (Default)

Apply to everything:

```yaml
chaos_profiles:
  everywhere:
    selector: all
    network:
      latency:
        enabled: true
        min: 100ms
        max: 200ms
```

### Name Selector

Target by name pattern:

```yaml
chaos_profiles:
  api_only:
    selector:
      name: "api-*"    # Prefix match
      # Or: name: "*-service"  # Suffix match
    network:
      latency:
        enabled: true
        min: 100ms
        max: 200ms
```

### Type Selector

Target by component type:

```yaml
chaos_profiles:
  tasks_only:
    selector:
      types: [task, validation]
    network:
      latency:
        enabled: true
        min: 50ms
        max: 100ms
```

### Tag Selector

Target by tags:

```yaml
chaos_profiles:
  external_calls:
    selector:
      tags: [external, api]
      match_all: false  # Match any tag (default)
    network:
      latency:
        enabled: true
        min: 200ms
        max: 1s
```

### Probability Selector

Apply randomly:

```yaml
chaos_profiles:
  random_failures:
    selector:
      probability: 0.1  # 10% chance
    network:
      packet_loss:
        enabled: true
        percentage: 100  # When selected, 100% loss
```

### Composite Selector

Combine multiple selectors:

```yaml
chaos_profiles:
  targeted_chaos:
    selector:
      mode: and  # or: or
      selectors:
        - tags: [external]
        - probability: 0.5
    network:
      latency:
        enabled: true
        min: 100ms
        max: 500ms
```

## Applying Chaos

### In Scenarios

```yaml
scenarios:
  - name: test_with_latency
    chaos_profiles: [network_latency]
    flow:
      - task: CallExternalAPI
      - validation: VerifyResponse
```

### Multiple Profiles

```yaml
scenarios:
  - name: stress_test
    chaos_profiles: [network_latency, cpu_stress]
    flow:
      - task: ProcessUnderLoad
```

### At Runtime

```bash
chronicle run --chaos network_latency
chronicle run --chaos network_latency --chaos cpu_stress
```

## Programmatic API

Create chaos profiles in code:

```go
import "github.com/joshua-temple/chronicle/pkg/chaos"

profile := chaos.NewProfile("network_latency",
    chaos.WithDescription("Simulate slow network"),
    chaos.WithFaults(
        chaos.NewLatencyFault(100*time.Millisecond, 500*time.Millisecond),
        chaos.NewPacketLossFault(0.05),
    ),
    chaos.WithSelector(chaos.NewTagSelector("external", "api")),
)
```

### Fault Types

```go
// Latency fault
fault := chaos.NewLatencyFault(min, max)
fault.WithJitter(0.1)

// Packet loss fault
fault := chaos.NewPacketLossFault(0.1)  // 10% loss

// Error injection fault
fault := chaos.NewErrorFault(errors.New("injected failure"))
fault.WithProbability(0.2)  // 20% chance
```

### Selectors

```go
// All targets
selector := chaos.AllSelector{}

// By name
selector := chaos.NameSelector{
    Pattern: "api-",
    Prefix: true,
}

// By type
selector := chaos.TypeSelector{
    Types: []string{"task", "validation"},
}

// By tag
selector := chaos.TagSelector{
    Tags: []string{"external"},
    MatchAll: false,
}

// By probability
selector := chaos.NewProbabilitySelector(0.1)

// Composite
selector := chaos.CompositeSelector{
    Mode: chaos.ModeAnd,
    Selectors: []chaos.Selector{
        chaos.TagSelector{Tags: []string{"external"}},
        chaos.NewProbabilitySelector(0.5),
    },
}
```

## Best Practices

1. **Start Small** - Begin with low probabilities and short durations
2. **Test Recovery** - Verify systems recover after chaos stops
3. **Use Selectors** - Target specific components, not everything
4. **Monitor** - Collect metrics during chaos experiments
5. **Document** - Record what chaos profiles test and why
6. **Gradual Increase** - Increase severity incrementally
7. **Production Readiness** - Use chaos testing to validate production resilience

## Common Patterns

### Testing Timeouts

```yaml
chaos_profiles:
  timeout_test:
    network:
      latency:
        enabled: true
        min: 5s
        max: 10s

scenarios:
  - name: verify_timeout_handling
    chaos_profiles: [timeout_test]
    timeout: 3s  # Should timeout before latency completes
    flow:
      - task: SlowExternalCall
      - validation: VerifyTimeoutError
```

### Testing Retries

```yaml
chaos_profiles:
  intermittent_failures:
    selector:
      probability: 0.5  # 50% of calls fail
    custom:
      during:
        - name: inject_error
          command: return_error
          args: [--code, 503]

scenarios:
  - name: verify_retry_logic
    chaos_profiles: [intermittent_failures]
    flow:
      - task: CallWithRetry
      - validation: VerifyEventualSuccess
```

### Testing Circuit Breakers

```yaml
chaos_profiles:
  sustained_failure:
    network:
      packet_loss:
        enabled: true
        percentage: 100  # Complete failure

scenarios:
  - name: verify_circuit_breaker
    chaos_profiles: [sustained_failure]
    flow:
      - task: MultipleFailingCalls
      - validation: VerifyCircuitOpen
      - task: WaitForRecovery
      - validation: VerifyCircuitClosed
```

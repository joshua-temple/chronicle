# Chronicle Infrastructure Networking

This example demonstrates Chronicle's infrastructure networking features:

- Shared Docker networks for container-to-container communication
- Endpoint discovery for service connection info
- Custom client registration for service-layer access

## Shared Networking

Enable shared networking in your Chronicle config:

```yaml
# chronicle.yaml
infrastructure:
  networking:
    enabled: true  # All containers join a shared network

  postgres:
    provider: postgres
    image: postgres:15
```

With networking enabled, containers can reach each other by name:
- External access: `localhost:54321` (mapped port)
- Container-to-container: `postgres:5432` (internal address)

## Endpoint Discovery

Access endpoint info in your components:

```go
func SetupDatabase(ctx chronicle.Context) error {
    ep, ok := ctx.Endpoint("postgres")
    if !ok {
        return errors.New("postgres endpoint not found")
    }

    // For host access
    dsn := fmt.Sprintf("postgres://user:pass@%s/db", ep.Address())

    // For container-to-container
    internalDSN := fmt.Sprintf("postgres://user:pass@%s/db", ep.InternalAddress())

    return nil
}
```

## Environment Variables

Chronicle exports endpoint info as environment variables:

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=54321
POSTGRES_INTERNAL_HOST=postgres
POSTGRES_INTERNAL_PORT=5432
POSTGRES_ADDRESS=localhost:54321
```

## Custom Clients

Register service-layer clients for use across components:

```go
func SetupOrderService(ctx chronicle.Context) error {
    ep, _ := ctx.Endpoint("postgres")
    client := orders.NewClient(ep.Address())
    ctx.RegisterClient("order-service", client)
    return nil
}

func TestCreateOrder(ctx chronicle.Context) error {
    client, _ := ctx.Client("order-service")
    orderClient := client.(*orders.Client)
    // use client...
    return nil
}
```

## Host Services

Register services running on the host (mocks, local processes):

```go
mockServer := httptest.NewServer(handler)
ctx.Endpoints().Register("payment-mock", infrastructure.HostEndpoint(8080))
// Containers can reach it at host.docker.internal:8080
```

# Mocking

Chronicle's mock system lets you simulate external services with configurable responses. Test error conditions, edge cases, and failure scenarios without depending on real services.

## Mock Profiles

Define mock profiles in `chronicle.yaml`:

```yaml
mock_profiles:
  payment_declined:
    description: Simulate declined payment
    services:
      - name: payment-api
        type: http
        rules:
          - match:
              method: POST
              path: /api/payments
            response:
              status: 402
              body: '{"error": "payment_declined", "code": "CARD_DECLINED"}'

  external_timeout:
    description: Simulate slow external service
    services:
      - name: external-api
        type: http
        rules:
          - match:
              path: /api/*
            delay: 30s
            response:
              status: 504
              body: '{"error": "gateway_timeout"}'
```

## Mock Rules

### Basic Matching

```yaml
mock_profiles:
  example:
    services:
      - name: api
        type: http
        rules:
          - match:
              method: GET
              path: /users/123
            response:
              status: 200
              body: '{"id": "123", "name": "Test User"}'
```

### Path Patterns

```yaml
rules:
  # Exact match
  - match:
      path: /api/users/123

  # Wildcard match
  - match:
      path: /api/users/*

  # Prefix match
  - match:
      path: /api/**
```

### Method Matching

```yaml
rules:
  - match:
      method: POST
      path: /api/orders

  - match:
      method: DELETE
      path: /api/orders/*
```

### Header Matching

```yaml
rules:
  - match:
      path: /api/users
      headers:
        Authorization: "Bearer admin-token"
    response:
      status: 200
      body: '{"users": [...]}'

  - match:
      path: /api/users
      headers:
        Authorization: "Bearer user-token"
    response:
      status: 200
      body: '{"users": []}'  # Limited results
```

### Body Matching

```yaml
rules:
  # String body match
  - match:
      path: /api/payments
      method: POST
      body: '{"amount": 0}'
    response:
      status: 400
      body: '{"error": "invalid_amount"}'

  # JSON body match (partial)
  - match:
      path: /api/payments
      method: POST
      body_json:
        currency: INVALID
    response:
      status: 400
      body: '{"error": "invalid_currency"}'
```

## Responses

### Status and Body

```yaml
rules:
  - match:
      path: /api/health
    response:
      status: 200
      headers:
        Content-Type: application/json
      body: '{"status": "healthy"}'
```

### Response from File

```yaml
rules:
  - match:
      path: /api/users
    response:
      status: 200
      file: ./fixtures/users.json
```

### Delayed Response

```yaml
rules:
  - match:
      path: /api/slow
    delay: 5s
    response:
      status: 200
      body: '{"result": "eventually"}'
```

## Fallback Behavior

Control what happens when no rule matches:

```yaml
mock_profiles:
  strict_mock:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/known
            response:
              status: 200
        fallback:
          action: error  # Fail on unmatched requests
          status: 500
          body: '{"error": "unexpected_request"}'

  passthrough_mock:
    services:
      - name: api
        type: http
        passthrough: true  # Forward unmatched to real service
        rules:
          - match:
              path: /api/override
            response:
              status: 200
              body: '{"mocked": true}'
```

### Fallback Actions

| Action | Description |
|--------|-------------|
| `error` | Return error response (default) |
| `passthrough` | Forward to real service |
| `default` | Return default response |

## Applying Mocks

### In Scenarios

```yaml
scenarios:
  - name: test_payment_failure
    mock_profiles: [payment_declined]
    flow:
      - setup: CreateOrder
      - task: ProcessPayment
      - validation: VerifyPaymentFailed
```

### Multiple Profiles

```yaml
scenarios:
  - name: test_multiple_failures
    mock_profiles: [payment_declined, inventory_empty]
    flow:
      - task: Checkout
      - validation: VerifyErrors
```

### At Runtime

```bash
chronicle run --mock payment_declined
chronicle run --mock payment_declined --mock slow_api
```

## Mock Service Types

### HTTP Mock

```yaml
services:
  - name: api
    type: http
    rules:
      - match:
          method: GET
          path: /resource
        response:
          status: 200
```

### gRPC Mock (if supported)

```yaml
services:
  - name: grpc-service
    type: grpc
    rules:
      - match:
          method: GetUser
          service: UserService
        response:
          message:
            id: "123"
            name: "Test"
```

## Common Patterns

### Error Scenarios

```yaml
mock_profiles:
  # 4xx Client Errors
  not_found:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/users/*
            response:
              status: 404
              body: '{"error": "not_found"}'

  unauthorized:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/**
            response:
              status: 401
              body: '{"error": "unauthorized"}'

  rate_limited:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/**
            response:
              status: 429
              headers:
                Retry-After: "60"
              body: '{"error": "rate_limited"}'

  # 5xx Server Errors
  server_error:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/**
            response:
              status: 500
              body: '{"error": "internal_error"}'

  service_unavailable:
    services:
      - name: api
        type: http
        rules:
          - match:
              path: /api/**
            response:
              status: 503
              body: '{"error": "service_unavailable"}'
```

### Conditional Responses

```yaml
mock_profiles:
  conditional:
    services:
      - name: payment-api
        type: http
        rules:
          # Decline specific card
          - match:
              method: POST
              path: /api/payments
              body_json:
                card_number: "4000000000000002"
            response:
              status: 402
              body: '{"error": "card_declined"}'

          # Approve others
          - match:
              method: POST
              path: /api/payments
            response:
              status: 200
              body: '{"status": "approved", "transaction_id": "txn_123"}'
```

### Stateful Mocking

```yaml
mock_profiles:
  stateful:
    services:
      - name: api
        type: http
        rules:
          # First call creates resource
          - match:
              method: POST
              path: /api/resources
            response:
              status: 201
              body: '{"id": "res_123"}'

          # Subsequent GET returns it
          - match:
              method: GET
              path: /api/resources/res_123
            response:
              status: 200
              body: '{"id": "res_123", "status": "created"}'
```

### Testing Validation

```yaml
mock_profiles:
  validation_errors:
    services:
      - name: api
        type: http
        rules:
          - match:
              method: POST
              path: /api/users
              body_json:
                email: ""
            response:
              status: 400
              body: '{"errors": [{"field": "email", "message": "required"}]}'

          - match:
              method: POST
              path: /api/users
              body_json:
                email: "invalid"
            response:
              status: 400
              body: '{"errors": [{"field": "email", "message": "invalid format"}]}'
```

## Best Practices

1. **Be Specific** - Use precise matchers to avoid unexpected matches
2. **Test Errors** - Create profiles for all error conditions
3. **Use Delays** - Test timeout handling with delayed responses
4. **Document Profiles** - Add descriptions to explain what each profile tests
5. **Organize by Feature** - Group mock profiles by the feature they test
6. **Keep in Sync** - Update mocks when real API contracts change
7. **Use Fixtures** - Store large response bodies in files

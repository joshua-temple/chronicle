package fullstack

import (
	"context"
	"testing"

	"github.com/joshua-temple/chronicle/examples/full-stack/components"
	"github.com/joshua-temple/chronicle/pkg/core"
	"github.com/joshua-temple/chronicle/pkg/execution"
	"github.com/joshua-temple/chronicle/pkg/scenario"
)

// TestCompleteOrderFlow demonstrates a full order flow using Chronicle components.
func TestCompleteOrderFlow(t *testing.T) {
	// Create executor
	executor := execution.NewExecutor()

	// Register components
	registerComponents(executor)

	// Build scenario programmatically
	s := scenario.NewBuilder("complete_order_success").
		Description("User successfully completes an order").
		Tags("order", "happy-path", "smoke").
		Setup("CreateTestUser").
		Setup("CreateTestProduct").
		Task("AddItemToCart").
		Task("InitiateCheckout").
		Task("ProcessPayment").
		Validation("VerifyOrderCreated").
		Validation("VerifyInventoryDeducted").
		Validation("VerifyPaymentRecorded").
		Teardown("CleanupTestOrder").
		Teardown("CleanupTestUser").
		Build()

	// Execute scenario
	result := executor.Execute(context.Background(), s)

	// Check result
	if !result.IsSuccess() {
		t.Errorf("Scenario failed: %v", result.Error)
		for _, fr := range result.FlowResults {
			if fr.Error != nil {
				t.Errorf("  %s: %v", fr.Name, fr.Error)
			}
		}
	}
}

// TestPaymentDeclinedFlow demonstrates handling a payment failure.
func TestPaymentDeclinedFlow(t *testing.T) {
	// Create executor
	executor := execution.NewExecutor()

	// Register components
	registerComponents(executor)

	// Build scenario
	s := scenario.NewBuilder("payment_declined_flow").
		Description("Order fails due to payment decline").
		Tags("order", "payment", "error-handling").
		Setup("CreateTestUser").
		Setup("CreateTestProduct").
		Task("AddItemToCart").
		Task("InitiateCheckout").
		// In a real test, we would configure the mock to decline payment
		// For this example, we skip ProcessPayment to simulate decline
		Validation("VerifyOrderNotCreated").
		Validation("VerifyInventoryNotDeducted").
		Teardown("CleanupTestUser").
		Build()

	// Execute scenario
	result := executor.Execute(context.Background(), s)

	// Check result
	if !result.IsSuccess() {
		t.Errorf("Scenario failed: %v", result.Error)
	}
}

// TestParallelValidations demonstrates parallel validation execution.
func TestParallelValidations(t *testing.T) {
	// Create executor with parallelism enabled
	executor := execution.NewExecutor(
		execution.WithParallelism(4),
	)

	// Register components
	registerComponents(executor)

	// Build scenario with parallel validations
	// Note: In this example, we add validations sequentially but they could
	// be marked as parallel in the config
	s := scenario.NewBuilder("parallel_validations").
		Description("Comprehensive order validation with parallel checks").
		Setup("CreateTestUser").
		Setup("CreateTestProduct").
		Task("AddItemToCart").
		Task("InitiateCheckout").
		Task("ProcessPayment").
		Validation("VerifyOrderCreated").
		Validation("VerifyInventoryDeducted").
		Validation("VerifyPaymentRecorded").
		Validation("VerifyOrderEventPublished").
		Validation("VerifyAnalyticsTracked").
		Teardown("CleanupTestOrder").
		Teardown("CleanupTestUser").
		Build()

	// Execute scenario
	result := executor.Execute(context.Background(), s)

	// Check result
	if !result.IsSuccess() {
		t.Errorf("Scenario failed: %v", result.Error)
	}
}

// registerComponents registers all e-commerce components with the executor.
func registerComponents(executor *execution.Executor) {
	// Setup components
	executor.RegisterComponent(core.NewComponent("CreateTestUser", core.ComponentSetup).
		WithFunc(components.CreateTestUser).
		WithProduces("user", "*User"))

	executor.RegisterComponent(core.NewComponent("CreateTestProduct", core.ComponentSetup).
		WithFunc(components.CreateTestProduct).
		WithProduces("product", "*Product"))

	// Task components
	executor.RegisterComponent(core.NewComponent("AddItemToCart", core.ComponentTask).
		WithFunc(components.AddItemToCart).
		WithRequires("user", "*User").
		WithRequires("product", "*Product").
		WithProduces("cart", "*Cart"))

	executor.RegisterComponent(core.NewComponent("InitiateCheckout", core.ComponentTask).
		WithFunc(components.InitiateCheckout).
		WithRequires("user", "*User").
		WithRequires("cart", "*Cart").
		WithProduces("checkout", "*Checkout"))

	executor.RegisterComponent(core.NewComponent("ProcessPayment", core.ComponentTask).
		WithFunc(components.ProcessPayment).
		WithRequires("checkout", "*Checkout").
		WithProduces("order", "*Order").
		WithProduces("payment", "*Payment"))

	executor.RegisterComponent(core.NewComponent("SendOrderConfirmation", core.ComponentTask).
		WithFunc(components.SendOrderConfirmation).
		WithRequires("order", "*Order").
		WithProduces("notification_sent", "bool"))

	// Validation components
	executor.RegisterComponent(core.NewComponent("VerifyOrderCreated", core.ComponentValidation).
		WithFunc(components.VerifyOrderCreated).
		WithRequires("order", "*Order"))

	executor.RegisterComponent(core.NewComponent("VerifyOrderNotCreated", core.ComponentValidation).
		WithFunc(components.VerifyOrderNotCreated))

	executor.RegisterComponent(core.NewComponent("VerifyInventoryDeducted", core.ComponentValidation).
		WithFunc(components.VerifyInventoryDeducted).
		WithRequires("product", "*Product"))

	executor.RegisterComponent(core.NewComponent("VerifyInventoryNotDeducted", core.ComponentValidation).
		WithFunc(components.VerifyInventoryNotDeducted).
		WithRequires("product", "*Product"))

	executor.RegisterComponent(core.NewComponent("VerifyPaymentRecorded", core.ComponentValidation).
		WithFunc(components.VerifyPaymentRecorded).
		WithRequires("payment", "*Payment"))

	executor.RegisterComponent(core.NewComponent("VerifyCartShowsOutOfStock", core.ComponentValidation).
		WithFunc(components.VerifyCartShowsOutOfStock).
		WithRequires("cart", "*Cart"))

	executor.RegisterComponent(core.NewComponent("VerifyOrderEventPublished", core.ComponentValidation).
		WithFunc(components.VerifyOrderEventPublished).
		WithRequires("order", "*Order"))

	executor.RegisterComponent(core.NewComponent("VerifyAnalyticsTracked", core.ComponentValidation).
		WithFunc(components.VerifyAnalyticsTracked).
		WithRequires("order", "*Order"))

	executor.RegisterComponent(core.NewComponent("VerifyNotificationSent", core.ComponentValidation).
		WithFunc(components.VerifyNotificationSent).
		WithRequires("notification_sent", "bool"))

	// Teardown components
	executor.RegisterComponent(core.NewComponent("CleanupTestOrder", core.ComponentTeardown).
		WithFunc(components.CleanupTestOrder).
		WithRequires("order", "*Order"))

	executor.RegisterComponent(core.NewComponent("CleanupTestUser", core.ComponentTeardown).
		WithFunc(components.CleanupTestUser).
		WithRequires("user", "*User"))
}


// Package components provides test components for the e-commerce integration tests.
package components

import (
	"fmt"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// User represents a test user.
type User struct {
	ID    string
	Email string
	Name  string
}

// Product represents a test product.
type Product struct {
	ID       string
	SKU      string
	Name     string
	Price    float64
	Quantity int
}

// @chronicle:setup
// @produces user:*User
// @teardown CleanupTestUser
// @description Creates a test user for the order flow
func CreateTestUser(ctx core.Context) error {
	user := &User{
		ID:    "test-user-001",
		Email: "test@example.com",
		Name:  "Test User",
	}

	ctx.Set("user", user)
	fmt.Printf("Created test user: %s (%s)\n", user.Name, user.Email)
	return nil
}

// @chronicle:setup
// @produces product:*Product
// @description Creates a test product for the order flow
func CreateTestProduct(ctx core.Context) error {
	product := &Product{
		ID:       "prod-001",
		SKU:      "TEST-SKU-001",
		Name:     "Test Product",
		Price:    99.99,
		Quantity: 100,
	}

	ctx.Set("product", product)
	fmt.Printf("Created test product: %s (SKU: %s, Price: $%.2f)\n", product.Name, product.SKU, product.Price)
	return nil
}

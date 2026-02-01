package basic

import (
	"errors"

	"github.com/joshua-temple/chronicle/pkg/context"
)

// @chronicle:setup name="CreateUser" produces="user:User" teardown="DeleteUser"
// @chronicle:description "Creates a test user for the scenario"
// @chronicle:tags setup,user
func CreateUser(ctx context.Context) error {
	user := &User{ID: "usr_123", Email: "test@example.com"}
	context.Set(ctx, "user", user)
	return nil
}

// @chronicle:teardown name="DeleteUser" requires="user:User"
// @chronicle:description "Deletes the test user"
func DeleteUser(ctx context.Context) error {
	user := context.Get[*User](ctx, "user")
	if user == nil {
		return nil
	}
	// cleanup logic would go here
	_ = user
	return nil
}

// @chronicle:task name="CreateOrder" requires="user:User" produces="order:Order"
// @chronicle:description "Creates an order for the user"
func CreateOrder(ctx context.Context) (*Order, error) {
	user := context.Get[*User](ctx, "user")
	if user == nil {
		return nil, errors.New("user not found in context")
	}
	order := &Order{ID: "ord_456", UserID: user.ID, Total: 99.99}
	context.Set(ctx, "order", order)
	return order, nil
}

// @chronicle:validation name="OrderValid" requires="order:Order"
// @chronicle:description "Validates the order was created correctly"
func OrderValid(ctx context.Context, result any) error {
	order := result.(*Order)
	if order.ID == "" {
		return errors.New("order ID should not be empty")
	}
	if order.Total <= 0 {
		return errors.New("order total should be positive")
	}
	return nil
}

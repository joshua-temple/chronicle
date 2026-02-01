package components

import (
	"fmt"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// @chronicle:teardown
// @requires order:*Order
// @description Cleans up test order data
// @tags cleanup
func CleanupTestOrder(ctx core.Context) error {
	orderVal, ok := ctx.Get("order")
	if !ok {
		// Order may not exist if test failed early
		fmt.Println("No order to cleanup")
		return nil
	}
	order := orderVal.(*Order)

	// In a real test, this would delete from database
	fmt.Printf("Cleaned up test order: %s\n", order.ID)
	return nil
}

// @chronicle:teardown
// @requires user:*User
// @description Cleans up test user data
// @tags cleanup
func CleanupTestUser(ctx core.Context) error {
	userVal, ok := ctx.Get("user")
	if !ok {
		// User may not exist if test failed early
		fmt.Println("No user to cleanup")
		return nil
	}
	user := userVal.(*User)

	// In a real test, this would delete from database
	fmt.Printf("Cleaned up test user: %s\n", user.ID)
	return nil
}

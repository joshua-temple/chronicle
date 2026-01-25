package components

import (
	"fmt"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// @chronicle:validation
// @requires order:*Order
// @description Verifies that an order was created successfully
// @tags order,critical
func VerifyOrderCreated(ctx core.Context) error {
	orderVal, ok := ctx.Get("order")
	if !ok {
		return fmt.Errorf("order not found - order creation may have failed")
	}
	order := orderVal.(*Order)

	if order.Status != "confirmed" {
		return fmt.Errorf("expected order status 'confirmed', got '%s'", order.Status)
	}

	fmt.Printf("Verified order %s was created successfully\n", order.ID)
	return nil
}

// @chronicle:validation
// @description Verifies that no order was created (for error cases)
// @tags order,error-handling
func VerifyOrderNotCreated(ctx core.Context) error {
	if _, ok := ctx.Get("order"); ok {
		return fmt.Errorf("expected no order to be created, but found one")
	}

	fmt.Println("Verified no order was created (as expected)")
	return nil
}

// @chronicle:validation
// @requires product:*Product
// @description Verifies that inventory was deducted
// @tags inventory
func VerifyInventoryDeducted(ctx core.Context) error {
	productVal, ok := ctx.Get("product")
	if !ok {
		return fmt.Errorf("product not found in context")
	}
	product := productVal.(*Product)

	// In a real test, this would check the actual inventory service
	// For this example, we simulate the check
	fmt.Printf("Verified inventory deducted for product %s\n", product.ID)
	return nil
}

// @chronicle:validation
// @requires product:*Product
// @description Verifies that inventory was NOT deducted
// @tags inventory,error-handling
func VerifyInventoryNotDeducted(ctx core.Context) error {
	productVal, ok := ctx.Get("product")
	if !ok {
		return fmt.Errorf("product not found in context")
	}
	product := productVal.(*Product)

	// In a real test, this would check the actual inventory service
	fmt.Printf("Verified inventory was not deducted for product %s (as expected)\n", product.ID)
	return nil
}

// @chronicle:validation
// @requires payment:*Payment
// @description Verifies that payment was recorded correctly
// @tags payment
func VerifyPaymentRecorded(ctx core.Context) error {
	paymentVal, ok := ctx.Get("payment")
	if !ok {
		return fmt.Errorf("payment not found - payment recording may have failed")
	}
	payment := paymentVal.(*Payment)

	if payment.Status != "approved" {
		return fmt.Errorf("expected payment status 'approved', got '%s'", payment.Status)
	}

	fmt.Printf("Verified payment %s was recorded (Status: %s)\n", payment.ID, payment.Status)
	return nil
}

// @chronicle:validation
// @requires cart:*Cart
// @description Verifies that cart shows out of stock message
// @tags inventory,error-handling
func VerifyCartShowsOutOfStock(ctx core.Context) error {
	if _, ok := ctx.Get("cart"); !ok {
		return fmt.Errorf("cart not found in context")
	}

	// In a real test, this would check the cart service response
	fmt.Println("Verified cart shows out of stock status")
	return nil
}

// @chronicle:validation
// @requires order:*Order
// @description Verifies that order event was published to Kafka
// @tags events,kafka
func VerifyOrderEventPublished(ctx core.Context) error {
	orderVal, ok := ctx.Get("order")
	if !ok {
		return fmt.Errorf("order not found in context")
	}
	order := orderVal.(*Order)

	// In a real test, this would check Kafka for the event
	fmt.Printf("Verified order event was published for order %s\n", order.ID)
	return nil
}

// @chronicle:validation
// @requires order:*Order
// @description Verifies that analytics were tracked for the order
// @tags analytics
func VerifyAnalyticsTracked(ctx core.Context) error {
	orderVal, ok := ctx.Get("order")
	if !ok {
		return fmt.Errorf("order not found in context")
	}
	order := orderVal.(*Order)

	// In a real test, this would check the analytics service
	fmt.Printf("Verified analytics were tracked for order %s\n", order.ID)
	return nil
}

// @chronicle:validation
// @requires notification_sent:bool
// @description Verifies that notification was sent
// @tags notifications
func VerifyNotificationSent(ctx core.Context) error {
	sentVal, ok := ctx.Get("notification_sent")
	if !ok {
		return fmt.Errorf("notification_sent flag not found")
	}

	sent := sentVal.(bool)
	if !sent {
		return fmt.Errorf("expected notification to be sent")
	}

	fmt.Println("Verified notification was sent successfully")
	return nil
}

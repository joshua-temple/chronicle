package components

import (
	"fmt"

	"github.com/joshua-temple/chronicle/pkg/core"
)

// CartItem represents an item in the shopping cart.
type CartItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

// Cart represents a shopping cart.
type Cart struct {
	ID     string
	UserID string
	Items  []CartItem
	Total  float64
}

// Checkout represents a checkout session.
type Checkout struct {
	ID      string
	CartID  string
	UserID  string
	Total   float64
	Status  string
	Address string
}

// Order represents a completed order.
type Order struct {
	ID        string
	UserID    string
	Items     []CartItem
	Total     float64
	Status    string
	PaymentID string
}

// Payment represents a payment record.
type Payment struct {
	ID            string
	OrderID       string
	Amount        float64
	Status        string
	TransactionID string
	Provider      string
}

// @chronicle:task
// @requires user:*User
// @requires product:*Product
// @produces cart:*Cart
// @description Adds a product to the user's shopping cart
func AddItemToCart(ctx core.Context) error {
	userVal, ok := ctx.Get("user")
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	user := userVal.(*User)

	productVal, ok := ctx.Get("product")
	if !ok {
		return fmt.Errorf("product not found in context")
	}
	product := productVal.(*Product)

	cart := &Cart{
		ID:     "cart-001",
		UserID: user.ID,
		Items: []CartItem{
			{
				ProductID: product.ID,
				Quantity:  1,
				Price:     product.Price,
			},
		},
		Total: product.Price,
	}

	ctx.Set("cart", cart)
	fmt.Printf("Added %s to cart (Total: $%.2f)\n", product.Name, cart.Total)
	return nil
}

// @chronicle:task
// @requires user:*User
// @requires cart:*Cart
// @produces checkout:*Checkout
// @description Initiates the checkout process
func InitiateCheckout(ctx core.Context) error {
	userVal, ok := ctx.Get("user")
	if !ok {
		return fmt.Errorf("user not found in context")
	}
	user := userVal.(*User)

	cartVal, ok := ctx.Get("cart")
	if !ok {
		return fmt.Errorf("cart not found in context")
	}
	cart := cartVal.(*Cart)

	checkout := &Checkout{
		ID:      "checkout-001",
		CartID:  cart.ID,
		UserID:  user.ID,
		Total:   cart.Total,
		Status:  "pending",
		Address: "123 Test Street, Test City, TS 12345",
	}

	ctx.Set("checkout", checkout)
	fmt.Printf("Initiated checkout for user %s (Total: $%.2f)\n", user.ID, checkout.Total)
	return nil
}

// @chronicle:task
// @requires checkout:*Checkout
// @produces order:*Order
// @produces payment:*Payment
// @description Processes payment and creates the order
// @tags payment,critical
func ProcessPayment(ctx core.Context) error {
	checkoutVal, ok := ctx.Get("checkout")
	if !ok {
		return fmt.Errorf("checkout not found in context")
	}
	checkout := checkoutVal.(*Checkout)

	// Simulate payment processing
	// In real tests, this would call the mock payment service
	payment := &Payment{
		ID:            "payment-001",
		OrderID:       "order-001",
		Amount:        checkout.Total,
		Status:        "approved",
		TransactionID: "txn_test_12345",
		Provider:      "stripe",
	}

	// Check if provider was specified in params
	if provider, ok := ctx.Get("provider"); ok {
		payment.Provider = provider.(string)
	}

	order := &Order{
		ID:        "order-001",
		UserID:    checkout.UserID,
		Items:     []CartItem{}, // Would be populated from cart
		Total:     checkout.Total,
		Status:    "confirmed",
		PaymentID: payment.ID,
	}

	ctx.Set("payment", payment)
	ctx.Set("order", order)
	fmt.Printf("Payment processed: %s (Amount: $%.2f, Provider: %s)\n", payment.TransactionID, payment.Amount, payment.Provider)
	fmt.Printf("Order created: %s (Status: %s)\n", order.ID, order.Status)
	return nil
}

// @chronicle:task
// @requires order:*Order
// @produces notification_sent:bool
// @description Sends order confirmation notification
// @tags notifications
func SendOrderConfirmation(ctx core.Context) error {
	orderVal, ok := ctx.Get("order")
	if !ok {
		return fmt.Errorf("order not found in context")
	}
	order := orderVal.(*Order)

	// Simulate sending notification
	fmt.Printf("Sending order confirmation for order %s\n", order.ID)
	ctx.Set("notification_sent", true)
	return nil
}

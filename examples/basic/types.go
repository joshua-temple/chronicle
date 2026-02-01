package basic

// @chronicle:type
type User struct {
	ID    string
	Email string
}

// @chronicle:type
type Order struct {
	ID     string
	UserID string
	Total  float64
}

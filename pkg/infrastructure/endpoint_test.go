package infrastructure

import (
	"testing"
)

func TestEndpoint_Address(t *testing.T) {
	ep := Endpoint{
		Host: "localhost",
		Port: 5432,
	}

	got := ep.Address()
	want := "localhost:5432"

	if got != want {
		t.Errorf("Address() = %q, want %q", got, want)
	}
}

func TestEndpoint_InternalAddress(t *testing.T) {
	ep := Endpoint{
		InternalHost: "postgres",
		InternalPort: 5432,
	}

	got := ep.InternalAddress()
	want := "postgres:5432"

	if got != want {
		t.Errorf("InternalAddress() = %q, want %q", got, want)
	}
}

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

func TestEndpointRegistry_RegisterAndGet(t *testing.T) {
	registry := NewEndpointRegistry()

	ep := Endpoint{
		Host:         "localhost",
		Port:         54321,
		InternalHost: "postgres",
		InternalPort: 5432,
		Protocol:     "tcp",
	}

	registry.Register("postgres", ep)

	got, ok := registry.Get("postgres")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}

	if got.Host != ep.Host || got.Port != ep.Port {
		t.Errorf("Get() = %+v, want %+v", got, ep)
	}
}

func TestEndpointRegistry_GetNotFound(t *testing.T) {
	registry := NewEndpointRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for nonexistent endpoint, want false")
	}
}

func TestEndpointRegistry_All(t *testing.T) {
	registry := NewEndpointRegistry()

	registry.Register("postgres", Endpoint{Host: "localhost", Port: 5432})
	registry.Register("redis", Endpoint{Host: "localhost", Port: 6379})

	all := registry.All()

	if len(all) != 2 {
		t.Errorf("All() returned %d endpoints, want 2", len(all))
	}

	if _, ok := all["postgres"]; !ok {
		t.Error("All() missing postgres endpoint")
	}
	if _, ok := all["redis"]; !ok {
		t.Error("All() missing redis endpoint")
	}
}

func TestEndpointRegistry_Names(t *testing.T) {
	registry := NewEndpointRegistry()

	registry.Register("postgres", Endpoint{Host: "localhost", Port: 5432})
	registry.Register("redis", Endpoint{Host: "localhost", Port: 6379})

	names := registry.Names()

	if len(names) != 2 {
		t.Errorf("Names() returned %d names, want 2", len(names))
	}
}

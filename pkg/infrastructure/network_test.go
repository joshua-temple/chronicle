package infrastructure

import "testing"

func TestHostInternal(t *testing.T) {
	if HostInternal == "" {
		t.Error("HostInternal should not be empty")
	}
	if HostInternal != "host.docker.internal" {
		t.Errorf("HostInternal = %q, want %q", HostInternal, "host.docker.internal")
	}
}

func TestHostEndpoint(t *testing.T) {
	ep := HostEndpoint(8080)

	if ep.Host != "localhost" {
		t.Errorf("HostEndpoint().Host = %q, want %q", ep.Host, "localhost")
	}
	if ep.Port != 8080 {
		t.Errorf("HostEndpoint().Port = %d, want 8080", ep.Port)
	}
	if ep.InternalHost != HostInternal {
		t.Errorf("HostEndpoint().InternalHost = %q, want %q", ep.InternalHost, HostInternal)
	}
	if ep.InternalPort != 8080 {
		t.Errorf("HostEndpoint().InternalPort = %d, want 8080", ep.InternalPort)
	}
	if ep.Protocol != "tcp" {
		t.Errorf("HostEndpoint().Protocol = %q, want %q", ep.Protocol, "tcp")
	}
}

func TestHostEndpointWithProtocol(t *testing.T) {
	ep := HostEndpointWithProtocol(8080, "http")

	if ep.Protocol != "http" {
		t.Errorf("HostEndpointWithProtocol().Protocol = %q, want %q", ep.Protocol, "http")
	}
	if ep.Host != "localhost" {
		t.Errorf("HostEndpointWithProtocol().Host = %q, want %q", ep.Host, "localhost")
	}
	if ep.Port != 8080 {
		t.Errorf("HostEndpointWithProtocol().Port = %d, want 8080", ep.Port)
	}
	if ep.InternalHost != HostInternal {
		t.Errorf("HostEndpointWithProtocol().InternalHost = %q, want %q", ep.InternalHost, HostInternal)
	}
	if ep.InternalPort != 8080 {
		t.Errorf("HostEndpointWithProtocol().InternalPort = %d, want 8080", ep.InternalPort)
	}
}

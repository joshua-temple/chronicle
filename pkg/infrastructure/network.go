package infrastructure

// HostInternal is the hostname containers use to reach the host machine.
// On Docker for Mac/Windows, this resolves to the host.
// On Linux, TestContainers configures this automatically with host access.
const HostInternal = "host.docker.internal"

// HostEndpoint creates an endpoint for a service running on the host machine.
// Use this to register mock servers or local processes that containers need to reach.
func HostEndpoint(port int) Endpoint {
	return Endpoint{
		Host:         "localhost",
		Port:         port,
		InternalHost: HostInternal,
		InternalPort: port,
		Protocol:     "tcp",
	}
}

// HostEndpointWithProtocol creates a host endpoint with a specific protocol.
func HostEndpointWithProtocol(port int, protocol string) Endpoint {
	ep := HostEndpoint(port)
	ep.Protocol = protocol
	return ep
}

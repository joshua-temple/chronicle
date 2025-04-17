package core

import (
	"fmt"
	"strings"
)

// TypedID is a base interface for all strongly-typed identifiers
type TypedID interface {
	// String returns the string representation of the ID
	String() string

	// Domain returns the top-level domain of the ID
	Domain() string

	// IsEmpty returns true if the ID is empty
	IsEmpty() bool
}

// baseID implements common functionality for all ID types
type baseID string

// String returns the string representation of the ID
func (id baseID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the ID
func (id baseID) Domain() string {
	parts := strings.Split(string(id), ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// IsEmpty returns true if the ID is empty
func (id baseID) IsEmpty() bool {
	return id == ""
}

// IDRegistry is a registry of all known IDs for validation
type IDRegistry struct {
	testIDs      map[TestID]bool
	serviceIDs   map[ServiceID]bool
	componentIDs map[ComponentID]bool
	portIDs      map[PortID]bool
	tagIDs       map[TagID]bool
}

// NewIDRegistry creates a new ID registry
func NewIDRegistry() *IDRegistry {
	return &IDRegistry{
		testIDs:      make(map[TestID]bool),
		serviceIDs:   make(map[ServiceID]bool),
		componentIDs: make(map[ComponentID]bool),
		portIDs:      make(map[PortID]bool),
		tagIDs:       make(map[TagID]bool),
	}
}

// RegisterTestID registers a test ID
func (r *IDRegistry) RegisterTestID(id TestID) {
	r.testIDs[id] = true
}

// RegisterServiceID registers a service ID
func (r *IDRegistry) RegisterServiceID(id ServiceID) {
	r.serviceIDs[id] = true
}

// RegisterComponentID registers a component ID
func (r *IDRegistry) RegisterComponentID(id ComponentID) {
	r.componentIDs[id] = true
}

// RegisterPortID registers a port ID
func (r *IDRegistry) RegisterPortID(id PortID) {
	r.portIDs[id] = true
}

// RegisterTagID registers a tag ID
func (r *IDRegistry) RegisterTagID(id TagID) {
	r.tagIDs[id] = true
}

// IsValidTestID checks if a test ID is registered
func (r *IDRegistry) IsValidTestID(id TestID) bool {
	return r.testIDs[id]
}

// IsValidServiceID checks if a service ID is registered
func (r *IDRegistry) IsValidServiceID(id ServiceID) bool {
	return r.serviceIDs[id]
}

// IsValidComponentID checks if a component ID is registered
func (r *IDRegistry) IsValidComponentID(id ComponentID) bool {
	return r.componentIDs[id]
}

// IsValidPortID checks if a port ID is registered
func (r *IDRegistry) IsValidPortID(id PortID) bool {
	return r.portIDs[id]
}

// IsValidTagID checks if a tag ID is registered
func (r *IDRegistry) IsValidTagID(id TagID) bool {
	return r.tagIDs[id]
}

// TestID is a strongly-typed identifier for a test
type TestID baseID

// String returns the string representation of the TestID
func (id TestID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the TestID
func (id TestID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the TestID is empty
func (id TestID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// ServiceID is a strongly-typed identifier for a service
type ServiceID baseID

// String returns the string representation of the ServiceID
func (id ServiceID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the ServiceID
func (id ServiceID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the ServiceID is empty
func (id ServiceID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// ComponentID is a strongly-typed identifier for a component
type ComponentID baseID

// String returns the string representation of the ComponentID
func (id ComponentID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the ComponentID
func (id ComponentID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the ComponentID is empty
func (id ComponentID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// PortID is a strongly-typed identifier for a port
type PortID baseID

// String returns the string representation of the PortID
func (id PortID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the PortID
func (id PortID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the PortID is empty
func (id PortID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// TagID is a strongly-typed identifier for a tag
type TagID baseID

// String returns the string representation of the TagID
func (id TagID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the TagID
func (id TagID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the TagID is empty
func (id TagID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// SuiteID is a strongly-typed identifier for a suite
type SuiteID baseID

// String returns the string representation of the SuiteID
func (id SuiteID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the SuiteID
func (id SuiteID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the SuiteID is empty
func (id SuiteID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// EnvID is a strongly-typed identifier for an environment variable
type EnvID baseID

// String returns the string representation of the EnvID
func (id EnvID) String() string {
	return string(id)
}

// Domain returns the top-level domain of the EnvID
func (id EnvID) Domain() string {
	return baseID(id).Domain()
}

// IsEmpty returns true if the EnvID is empty
func (id EnvID) IsEmpty() bool {
	return baseID(id).IsEmpty()
}

// IDDescriptor represents metadata for an ID
type IDDescriptor struct {
	ID          string // The actual ID value
	Description string // Description of what this ID represents
	Category    string // Optional categorization
}

// IDSet represents a collection of related IDs
type IDSet struct {
	Domain      string         // Top-level domain
	Description string         // Description of this domain
	IDs         []IDDescriptor // The IDs in this domain
}

// ValidateIDs validates that all provided IDs are registered
func ValidateIDs(registry *IDRegistry, ids ...TypedID) error {
	for _, id := range ids {
		switch typedID := id.(type) {
		case TestID:
			if !registry.IsValidTestID(typedID) {
				return fmt.Errorf("invalid test ID: %s", typedID)
			}
		case ServiceID:
			if !registry.IsValidServiceID(typedID) {
				return fmt.Errorf("invalid service ID: %s", typedID)
			}
		case ComponentID:
			if !registry.IsValidComponentID(typedID) {
				return fmt.Errorf("invalid component ID: %s", typedID)
			}
		case PortID:
			if !registry.IsValidPortID(typedID) {
				return fmt.Errorf("invalid port ID: %s", typedID)
			}
		case TagID:
			if !registry.IsValidTagID(typedID) {
				return fmt.Errorf("invalid tag ID: %s", typedID)
			}
		default:
			return fmt.Errorf("unknown ID type")
		}
	}
	return nil
}

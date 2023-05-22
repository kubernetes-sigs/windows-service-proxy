package hcn

import (
	"fmt"
)

// Endpoint is a user-oriented definition of an HostComputeEndpoint in its entirety.
type Endpoint struct {
	// host compute identifier, returned after creation
	ID string

	IP         string
	IsLocal    bool
	MacAddress string
	ProviderIP string
}

// Key returns identifier for diffstore.
func (ep *Endpoint) Key() string {
	return ep.IP
}

// Equal compares if two endpoints are equal.
func (ep *Endpoint) Equal(other *Endpoint) bool {
	return ep.IsLocal == other.IsLocal && ep.Key() == other.Key() &&
		ep.MacAddress == other.MacAddress && ep.ProviderIP == other.ProviderIP
}

// String returns string representation of endpoint.
func (ep *Endpoint) String() string {
	return fmt.Sprintf("<Endpoint Key=%s ID=%s>", ep.IP, ep.ID)
}

package util

import "net"

// NetworkInterfacer defines an interface for several net library functions. Production
// code will forward to net library functions, and unit tests will override the methods
// for testing purposes.
type NetworkInterfacer interface {
	InterfaceAddrs() ([]net.Addr, error)
}

// RealNetwork implements the NetworkInterfacer interface for production code, just
// wrapping the underlying net library function calls.
type RealNetwork struct{}

// InterfaceAddrs wraps net.InterfaceAddrs(), it's a part of NetworkInterfacer interface.
func (RealNetwork) InterfaceAddrs() ([]net.Addr, error) {
	return net.InterfaceAddrs()
}

var _ NetworkInterfacer = &RealNetwork{}

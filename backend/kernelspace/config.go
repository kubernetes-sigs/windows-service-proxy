package kernelspace

// KubeProxyWinkernelConfiguration contains Windows/HNS settings for
// the Kubernetes proxy server.
type KubeProxyWinkernelConfiguration struct {
	// NetworkName is the name of the network kube-proxy will use
	// to create endpoints and policies
	NetworkName string
	// SourceVip is the IP address of the source VIP endpoint used for
	// NAT when loadbalancing
	SourceVip string
	// enableDSR tells kube-proxy whether HNS policies should be created
	// with DSR
	EnableDSR bool
	// RootHnsEndpointName is the name of hnsendpoint that is attached to
	// l2bridge for root network namespace
	RootHnsEndpointName string
	// ForwardHealthCheckVip forwards service VIP for health check port on
	// Windows
	ForwardHealthCheckVip bool
}

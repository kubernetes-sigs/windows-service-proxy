package kernelspacefs

import (
	"sigs.k8s.io/kpng/client/diffstore"
	"sigs.k8s.io/windows-service-proxy/backend/kernelspacefs/hcn"
)

var (
	// endpointStore
	remoteEndpointStore = diffstore.NewAnyStore[string, *hcn.Endpoint](func(a, b *hcn.Endpoint) bool {
		return a.Equal(b)
	})

	// loadBalancerStore
	loadBalancerStore = diffstore.NewAnyStore[string, *hcn.LoadBalancer](func(a, b *hcn.LoadBalancer) bool {
		return a.Equal(b)
	})
)

func addRemoteEndpointToStore(endpoint *hcn.Endpoint) {
	// associate host compute identifier
	if id, ok := endpointIDMap[endpoint.Key()]; ok {
		endpoint.ID = id
	}

	remoteEndpointStore.Get(endpoint.Key()).Set(endpoint)
}

func addLoadBalancerToStore(loadBalancer *hcn.LoadBalancer) {
	// associate host compute identifier
	if id, ok := loadBalancerIDMap[loadBalancer.Key()]; ok {
		loadBalancer.ID = id
	}

	loadBalancerStore.Get(loadBalancer.Key()).Set(loadBalancer)
}

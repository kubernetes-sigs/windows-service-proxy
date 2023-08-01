/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kernelspacefs

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kpng/client"
	"sigs.k8s.io/windows-service-proxy/backend/kernelspacefs/hcn"
	"sync"
)

// hcnClient is used to program HostCompute objects in Windows kernel.
var hcnClient hcn.Interface

// network represents HostComputeNetwork configured by cloud provider.
var network *hcn.Network

// mu is mu
var mu sync.Mutex

// localEndpoints maps endpoint IP with corresponding hcn.Endpoint programmed by CNI.
var localEndpoints map[string]*hcn.Endpoint

// remoteEndpoints maps endpoint IP with corresponding hcn.Endpoint programmed by us
var remoteEndpoints map[string]*hcn.Endpoint

// endpointIDMap maps hcn.Endpoint (IP) to its HostComputeEndpoint identifier in Windows kernel.
var endpointIDMap map[string]string

// loadBalancerIDMap maps hcn.LoadBalancer (IP-Port-Protocol) to its HostComputeLoadBalancer identifier in Windows kernel.
var loadBalancerIDMap map[string]string

// Setup is used by KPNG brain to:
// 1. creates hcnClient
// 2. initialize required maps
// 3. load hcn.Network
// 4. initialize diffstores
func Setup() {
	var err error
	var networkName string

	// initialize maps
	localEndpoints = make(map[string]*hcn.Endpoint)
	remoteEndpoints = make(map[string]*hcn.Endpoint)
	endpointIDMap = make(map[string]string)
	loadBalancerIDMap = make(map[string]string)

	// create hcnClient
	hcnClient = hcn.New(*enableDSR)

	networkName, err = getNetworkName("")
	if err != nil {

	}

	// load hcn network
	network, err = hcnClient.GetNetworkByName(networkName)
	if err != nil {
		klog.Fatal()
	}

	mu.Lock()
	defer mu.Unlock()

	remoteEndpointStore.RunDeferred()
	loadBalancerStore.RunDeferred()

	// populate diffstore with hcn.Endpoint(s)
	endpoints, _ := hcnClient.ListEndpoints()
	for _, endpoint := range endpoints {
		endpointIDMap[endpoint.Key()] = endpoint.ID

		if endpoint.IsLocal {
			localEndpoints[endpoint.Key()] = endpoint
		} else {
			remoteEndpoints[endpoint.Key()] = endpoint
			// populate diffstore with remote endpoints only
			addRemoteEndpointToStore(endpoint)
		}
	}

	// populate diffstore with hcn.LoadBalancer(s)
	loadBalancers, _ := hcnClient.ListLoadBalancers()
	for _, loadBalancer := range loadBalancers {
		loadBalancerIDMap[loadBalancer.Key()] = loadBalancer.ID
		addLoadBalancerToStore(loadBalancer)
	}

	remoteEndpointStore.Done()
	loadBalancerStore.Done()

	remoteEndpointStore.Reset()
	loadBalancerStore.Reset()
}

func Callback(ch <-chan *client.ServiceEndpoints) {
	mu.Lock()
	defer mu.Unlock()

	// reset the diffstore
	defer remoteEndpointStore.Reset()
	defer loadBalancerStore.Reset()

	// load existing hcn.Endpoint(s)
	endpoints, _ := hcnClient.ListEndpoints()
	localEndpoints = make(map[string]*hcn.Endpoint)

	// update maps
	for _, endpoint := range endpoints {
		endpointIDMap[endpoint.Key()] = endpoint.ID
		if endpoint.IsLocal {
			localEndpoints[endpoint.Key()] = endpoint
		}
	}

	remoteEndpointStore.RunDeferred()
	loadBalancerStore.RunDeferred()

	// iterate over *client.ServiceEndpoints and program data path
	for serviceEndpoints := range ch {
		klog.V(2).InfoS("programming", "service", serviceEndpoints.Service.NamespacedName())
		switch serviceEndpoints.Service.Type {
		case ClusterIPService.String():
			addServiceEndpointsForClusterIP(serviceEndpoints)
		case NodePortService.String():
			addServiceEndpointsForNodePort(serviceEndpoints)
		case LoadBalancerService.String():
			klog.V(2).InfoS("not implemented")
		}
	}
	remoteEndpointStore.Done()
	loadBalancerStore.Done()

	// execute the changes; this call will have actual side effects,
	// kernel will be programed to achieve the desired data path.
	programHostComputeObjects()
}

// addServiceEndpointsForClusterIP breaks down *client.ServiceEndpoints into multiple network
// objects which needs to be programmed in the kernel to achieve the desired data path.
func addServiceEndpointsForClusterIP(serviceEndpoints *client.ServiceEndpoints) {

	// iterate over service ports
	for _, portMapping := range serviceEndpoints.Service.Ports {

		// iterate over ipFamily
		// todo: IPv6 support
		for _, ipFamily := range []v1.IPFamily{v1.IPv4Protocol} {

			// iterate over ClusterIPs
			for _, clusterIP := range getClusterIPs(serviceEndpoints.Service, ipFamily) {
				hcnEndpoints := make([]*hcn.Endpoint, 0)

				// iterate over service endpoints
				for _, endpoint := range serviceEndpoints.Endpoints {

					// iterate over EndpointIPs
					for _, endpointIP := range getEndpointIPs(endpoint, ipFamily) {

						hcnEndpoint := getEndpoint(endpointIP)
						if !hcnEndpoint.IsLocal {
							// important: only add remote hcn.Endpoints to diffstore, we don't own
							// the lifecycle of node local endpoints, so we should never delete it.
							addRemoteEndpointToStore(hcnEndpoint)
						}

						hcnEndpoints = append(hcnEndpoints, hcnEndpoint)

					}
				}

				if len(hcnEndpoints) > 0 {
					loadBalancer := getLoadBalancerForClusterIP(clusterIP, hcnEndpoints, portMapping)
					addLoadBalancerToStore(loadBalancer)
				}
			}
		}
	}
}

// addServiceEndpointsForNodePort breaks down *client.ServiceEndpoints into multiple network
// objects which needs to be programmed in the kernel to achieve the desired data path.
func addServiceEndpointsForNodePort(serviceEndpoints *client.ServiceEndpoints) {
	addServiceEndpointsForClusterIP(serviceEndpoints)

	// iterate over service ports
	for _, portMapping := range serviceEndpoints.Service.Ports {

		// iterate over ipFamily
		// todo: IPv6 support
		for _, ipFamily := range []v1.IPFamily{v1.IPv4Protocol} {

			hcnEndpoints := make([]*hcn.Endpoint, 0)

			// iterate over service endpoints
			for _, endpoint := range serviceEndpoints.Endpoints {

				// iterate over EndpointIPs
				for _, endpointIP := range getEndpointIPs(endpoint, ipFamily) {

					hcnEndpoint := getEndpoint(endpointIP)
					if !hcnEndpoint.IsLocal {
						// important: only add remote hcn.Endpoints to diffstore, we don't own
						// the lifecycle of node local endpoints, so we should never delete it.
						addRemoteEndpointToStore(hcnEndpoint)
					}
					hcnEndpoints = append(hcnEndpoints, hcnEndpoint)
				}
			}

			if len(hcnEndpoints) > 0 {
				loadBalancer := getLoadBalancerForNodePort(hcnEndpoints, portMapping)
				addLoadBalancerToStore(loadBalancer)
			}
		}
	}
}

func programHostComputeObjects() {
	var err error

	// remove loadBalancers
	for _, item := range loadBalancerStore.Deleted() {
		loadBalancer := item.Value().Get()
		hcID := loadBalancer.ID

		klog.V(4).InfoS("deleting loadBalancer", "loadBalancer", loadBalancer.String())
		err = hcnClient.DeleteLoadBalancer(loadBalancer)
		if err != nil {
			klog.V(4).ErrorS(err, "error deleting loadbalancer", "loadBalancer", loadBalancer.String())
			continue
		}

		// remove host compute identifier for loadbalancer
		delete(loadBalancerIDMap, hcID)
	}

	// remove endpoints
	for _, item := range remoteEndpointStore.Deleted() {
		endpoint := item.Value().Get()
		hcID := endpoint.ID

		klog.V(4).ErrorS(err, "deleting remote endpoint", "endpoint", endpoint.String())
		err = hcnClient.DeleteEndpoint(network, endpoint)
		if err != nil {
			klog.V(4).ErrorS(err, "error deleting remote endpoint", "endpoint", endpoint.String())
			continue
		}

		// remove host compute identifier for endpoint
		delete(endpointIDMap, hcID)
		delete(remoteEndpoints, endpoint.Key())
	}

	// create/update endpoints
	for _, item := range remoteEndpointStore.Changed() {
		endpoint := item.Value().Get()
		klog.V(4).ErrorS(err, "creating remote endpoint", "endpoint", endpoint.String())
		if item.Created() {
			err = hcnClient.CreateEndpoint(network, endpoint)
			if err != nil {
				klog.V(4).ErrorS(err, "error creating remote endpoint", "endpoint", endpoint.String())
				continue
			}

			endpointIDMap[endpoint.Key()] = endpoint.ID
			klog.V(4).InfoS("remote endpoint created", "endpoint", endpoint.String())

		} else if item.Updated() {
			klog.V(4).ErrorS(err, "updatinge remote endpoint", "endpoint", endpoint.String())
			err = hcnClient.UpdateEndpoint(network, endpoint)
			if err != nil {
				klog.V(4).ErrorS(err, "error deleting remote endpoint", "endpoint", endpoint.String())
				continue
			}

			endpointIDMap[endpoint.Key()] = endpoint.ID
			klog.V(4).InfoS("remote endpoint updated", "endpoint", endpoint.String())

		}
	}

	// create/update loadBalancers
	for _, item := range loadBalancerStore.Changed() {
		loadBalancer := item.Value().Get()

		if item.Created() {
			klog.V(4).InfoS("creating loadBalancer", "loadBalancer", loadBalancer.String())
			err = hcnClient.CreateLoadBalancer(loadBalancer)
			if err != nil {
				klog.V(4).ErrorS(err, "error creating loadBalancer", "loadBalancer", loadBalancer.String())
				continue
			}

			loadBalancerIDMap[loadBalancer.Key()] = loadBalancer.ID
			klog.V(4).InfoS("loadBalancer created", "loadBalancer", loadBalancer.String())

		} else if item.Updated() {
			klog.V(4).InfoS("updating loadBalancer", "loadBalancer", loadBalancer.String())
			err = hcnClient.UpdateLoadBalancer(loadBalancer)
			if err != nil {
				klog.V(4).ErrorS(err, "error updating loadBalancer", "loadBalancer", loadBalancer.String())
				continue
			}

			loadBalancerIDMap[loadBalancer.Key()] = loadBalancer.ID
			klog.V(4).InfoS("loadBalancer updated", "loadBalancer", loadBalancer.String())
		}
	}
}

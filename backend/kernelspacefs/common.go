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
	netutils "k8s.io/utils/net"
	"sigs.k8s.io/kpng/api/localv1"
	"sigs.k8s.io/windows-service-proxy/backend/kernelspacefs/hcn"
	utilproxy "sigs.k8s.io/windows-service-proxy/pkg/util"
)

// getEndpointIPs returns EndpointIPs for given IPFamily associated with the endpoint.
func getEndpointIPs(endpoint *localv1.Endpoint, ipFamily v1.IPFamily) []string {
	IPs := make([]string, 0)
	if endpoint.IPs != nil {
		if ipFamily == v1.IPv4Protocol {
			return endpoint.IPs.GetV4()
		} else if ipFamily == v1.IPv6Protocol {
			return endpoint.IPs.GetV6()
		}
	}
	return IPs
}

// getClusterIPs returns ClusterIPs for given IPFamily associated with the service.
func getClusterIPs(service *localv1.Service, ipFamily v1.IPFamily) []string {
	IPs := make([]string, 0)
	if service.IPs.ClusterIPs != nil {
		if ipFamily == v1.IPv4Protocol {
			return service.IPs.ClusterIPs.GetV4()
		} else if ipFamily == v1.IPv6Protocol {
			return service.IPs.ClusterIPs.GetV6()
		}
	}
	return IPs
}

// getProtocol returns protocol in uint32 which windows kernel understands.
func getProtocol(protocol localv1.Protocol) uint32 {
	switch protocol {
	case localv1.Protocol_TCP:
		return 6
	case localv1.Protocol_UDP:
		return 17
	case localv1.Protocol_SCTP:
		return 132
	default:
		return 0
	}
}

// getLoadBalancerForClusterIP returns hcn.LoadBalancer for ClusterIP service.
func getLoadBalancerForClusterIP(IP string, hcnEndpoints []*hcn.Endpoint, portMapping *localv1.PortMapping) *hcn.LoadBalancer {
	flags := hcn.LoadBalancerFlagsNone
	if *enableDSR {
		flags |= hcn.LoadBalancerFlagsDSR
	}

	portMappingFlags := hcn.LoadBalancerPortMappingFlagsNone

	return &hcn.LoadBalancer{
		IP:               IP,
		Endpoints:        hcnEndpoints,
		Flags:            flags,
		PortMappingFlags: portMappingFlags,
		SourceVip:        *sourceVip,
		Protocol:         getProtocol(portMapping.Protocol),
		Port:             portMapping.Port,
		TargetPort:       portMapping.TargetPort,
	}
}

// getLoadBalancerForNodePort returns hcn.LoadBalancer for NodePort service.
func getLoadBalancerForNodePort(hcnEndpoints []*hcn.Endpoint, portMapping *localv1.PortMapping) *hcn.LoadBalancer {
	flags := hcn.LoadBalancerFlagsNone
	if *enableDSR {
		flags |= hcn.LoadBalancerFlagsDSR
	}

	portMappingFlags := hcn.LoadBalancerPortMappingFlagsNone
	portMappingFlags |= hcn.LoadBalancerPortMappingFlagsLocalRoutedVIP

	return &hcn.LoadBalancer{
		Endpoints:        hcnEndpoints,
		Flags:            flags,
		PortMappingFlags: portMappingFlags,
		SourceVip:        *sourceVip,
		Protocol:         getProtocol(portMapping.Protocol),
		Port:             portMapping.NodePort,
		TargetPort:       portMapping.TargetPort,
	}
}

// getEndpoint returns hcn.Endpoint if it already exists, else create and return.
func getEndpoint(IP string) *hcn.Endpoint {

	// return local hcn.Endpoint if it exists
	if ep, ok := localEndpoints[IP]; ok {
		return ep
	}

	// return remote hcn.Endpoint
	if ep, ok := remoteEndpoints[IP]; ok {
		return ep
	}

	// create new hcn.Endpoint object
	ep := &hcn.Endpoint{
		IP:         IP,
		IsLocal:    false,
		MacAddress: utilproxy.ConjureMac("02-11", netutils.ParseIPSloppy(IP)),
	}

	remoteEndpoints[IP] = ep
	return ep
}

package hcn

import (
	"encoding/json"
	"fmt"
	hcnshim "github.com/Microsoft/hcsshim/hcn"
	"k8s.io/klog/v2"
)

// Interface is an injectable interface for making HCN calls.
type Interface interface {
	GetNetworkByName(string) (*Network, error)
	GetNetworkByID(string) (*Network, error)

	CreateEndpoint(*Network, *Endpoint) error
	UpdateEndpoint(*Network, *Endpoint) error
	DeleteEndpoint(*Network, *Endpoint) error
	ListEndpoints() ([]*Endpoint, error)

	CreateLoadBalancer(*LoadBalancer) error
	UpdateLoadBalancer(*LoadBalancer) error
	DeleteLoadBalancer(*LoadBalancer) error
	ListLoadBalancers() ([]*LoadBalancer, error)
}

func New(enableDSR bool) Interface {
	return &hcn{enableDSR: enableDSR}
}

//func NewFake() Interface {
//	return &fake{}
//}

type hcn struct {
	enableDSR bool
}

func (h *hcn) GetNetworkByID(id string) (*Network, error) {
	hcNetwork, err := hcnshim.GetNetworkByID(id)
	if err != nil {
		return nil, err
	}

	return getNetwork(hcNetwork)
}

func (h *hcn) GetNetworkByName(name string) (*Network, error) {
	hcNetwork, err := hcnshim.GetNetworkByName(name)
	if err != nil {
		return nil, err
	}

	return getNetwork(hcNetwork)
}

func getNetwork(hcNetwork *hcnshim.HostComputeNetwork) (*Network, error) {
	var err error

	var remoteSubnets []*RemoteSubnetInfo
	for _, policy := range hcNetwork.Policies {
		if policy.Type == hcnshim.RemoteSubnetRoute {
			policySettings := hcnshim.RemoteSubnetRoutePolicySetting{}
			err = json.Unmarshal(policy.Settings, &policySettings)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal Remote Subnet policy settings")
			}
			rs := &RemoteSubnetInfo{
				DestinationPrefix: policySettings.DestinationPrefix,
				IsolationID:       policySettings.IsolationId,
				ProviderAddress:   policySettings.ProviderAddress,
				DrMacAddress:      policySettings.DistributedRouterMacAddress,
			}
			remoteSubnets = append(remoteSubnets, rs)
		}
	}

	return &Network{
		Name:          hcNetwork.Name,
		Id:            hcNetwork.Id,
		Type:          NetworkType(hcNetwork.Type),
		RemoteSubnets: remoteSubnets,
	}, nil
}

func (h *hcn) CreateEndpoint(network *Network, endpoint *Endpoint) error {
	if endpoint.ID != "" {
		klog.V(4).InfoS("skipping create, endpoint already exists", "endpoint", endpoint.String())
		return nil
	}

	ipConfig := &hcnshim.IpConfig{
		IpAddress: endpoint.IP,
	}

	var err error
	var flags hcnshim.EndpointFlags
	if !endpoint.IsLocal {
		flags |= hcnshim.EndpointFlagsRemoteEndpoint
	}

	hcnEndpoint := &hcnshim.HostComputeEndpoint{
		HostComputeNetwork: network.Id,
		IpConfigurations:   []hcnshim.IpConfig{*ipConfig},
		MacAddress:         endpoint.MacAddress,
		Flags:              flags,
		SchemaVersion: hcnshim.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
	}

	var createdEndpoint *hcnshim.HostComputeEndpoint

	if len(endpoint.ProviderIP) != 0 {
		policySettings := hcnshim.ProviderAddressEndpointPolicySetting{
			ProviderAddress: endpoint.ProviderIP,
		}
		policySettingsJson, err := json.Marshal(policySettings)
		if err != nil {
			return fmt.Errorf("PA Policy creation failed: %v", err)
		}
		paPolicy := hcnshim.EndpointPolicy{
			Type:     hcnshim.NetworkProviderAddress,
			Settings: policySettingsJson,
		}
		hcnEndpoint.Policies = append(hcnEndpoint.Policies, paPolicy)
	}

	//createdEndpoint, err = hcnNetwork.CreateRemoteEndpoint(hcnEndpoint)
	createdEndpoint, err = hcnEndpoint.Create()

	if err != nil {
		return err
	}

	// important: associate host compute identifier for endpoint
	endpoint.ID = createdEndpoint.Id

	return nil

}

func (h *hcn) UpdateEndpoint(network *Network, endpoint *Endpoint) error {
	var err error
	err = h.DeleteEndpoint(network, endpoint)
	err = h.CreateEndpoint(network, endpoint)
	return err
}

func (h *hcn) DeleteEndpoint(_ *Network, endpoint *Endpoint) error {
	if endpoint.IsLocal {
		return fmt.Errorf("skipping delete, endpoint is local")

	}

	if endpoint.ID == "" {
		return fmt.Errorf("missing host compute identifier for endpoint")
	}

	hcnEndpoint, err := hcnshim.GetEndpointByID(endpoint.ID)
	if err != nil {
		return fmt.Errorf("endpoint not found")
	}

	err = hcnEndpoint.Delete()
	if err != nil {
		return err
	}

	// important: unset host compute identifier for endpoint
	endpoint.ID = ""
	return nil
}

func (h *hcn) ListEndpoints() ([]*Endpoint, error) {
	var err error
	query := hcnshim.HostComputeQuery{
		SchemaVersion: hcnshim.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
	}

	// getProviderIPFromPolicy returns ProviderAddress from HostComputeEndpoint Policies
	getProviderIPFromPolicy := func(policies []hcnshim.EndpointPolicy) string {
		for _, policy := range policies {
			if policy.Type == hcnshim.NetworkProviderAddress {

				var policySetting *hcnshim.ProviderAddressEndpointPolicySetting
				err = json.Unmarshal(policy.Settings, policySetting)
				if err != nil {
					klog.V(4).ErrorS(err, "Unable to unmarshal policy setting")
					return ""
				}
				return policySetting.ProviderAddress

			}
		}
		return ""
	}

	// Load HostComputeEndpoints
	hcEndpoints, _ := hcnshim.ListEndpointsQuery(query)
	endpoints := make([]*Endpoint, 0)

	for _, hcEndpoint := range hcEndpoints {
		endpoint := &Endpoint{
			ID: hcEndpoint.Id,

			IP:         hcEndpoint.IpConfigurations[0].IpAddress,
			IsLocal:    hcEndpoint.Flags != hcnshim.EndpointFlagsRemoteEndpoint,
			MacAddress: hcEndpoint.MacAddress,
			ProviderIP: getProviderIPFromPolicy(hcEndpoint.Policies),
		}

		endpoints = append(endpoints, endpoint)

	}
	return endpoints, nil
}

func (h *hcn) CreateLoadBalancer(loadBalancer *LoadBalancer) error {
	if loadBalancer.ID != "" {
		klog.V(4).InfoS("skipping create, loadBalancer already exists", "loadBalancer", loadBalancer.String())
		return nil
	}

	hcEndpoints := make([]string, 0)
	for _, endpoint := range loadBalancer.Endpoints {
		if endpoint.ID != "" {
			hcEndpoints = append(hcEndpoints, endpoint.ID)
		}
	}

	frontendVIPs := make([]string, 0)
	if len(loadBalancer.IP) > 0 {
		frontendVIPs = append(frontendVIPs, loadBalancer.IP)
	}

	hcLoadBalancer := &hcnshim.HostComputeLoadBalancer{
		HostComputeEndpoints: hcEndpoints,
		SourceVIP:            loadBalancer.SourceVip,
		PortMappings: []hcnshim.LoadBalancerPortMapping{
			{
				Protocol:         loadBalancer.Protocol,
				InternalPort:     uint16(loadBalancer.TargetPort),
				ExternalPort:     uint16(loadBalancer.Port),
				DistributionType: hcnshim.LoadBalancerDistributionNone,
				Flags:            hcnshim.LoadBalancerPortMappingFlags(loadBalancer.PortMappingFlags),
			},
		},
		FrontendVIPs: frontendVIPs,
		SchemaVersion: hcnshim.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
		Flags: hcnshim.LoadBalancerFlags(loadBalancer.Flags),
	}

	hcLoadBalancer, err := hcLoadBalancer.Create()
	if err != nil {
		return err
	}

	// important: associate host compute identifier for endpoint
	loadBalancer.ID = hcLoadBalancer.Id
	return nil
}

func (h *hcn) UpdateLoadBalancer(loadBalancer *LoadBalancer) error {
	var err error
	err = h.DeleteLoadBalancer(loadBalancer)
	err = h.CreateLoadBalancer(loadBalancer)
	return err
}

func (h *hcn) DeleteLoadBalancer(loadBalancer *LoadBalancer) error {
	if loadBalancer.ID == "" {
		return fmt.Errorf("missing host compute identifier for loadbalancer")
	}

	hcLoadBalancer, err := hcnshim.GetLoadBalancerByID(loadBalancer.ID)
	if err != nil {
		return fmt.Errorf("loadbalancer not found")
	}

	err = hcLoadBalancer.Delete()
	if err != nil {
		return err
	}

	// important: unset host compute identifier for loadBalancer
	loadBalancer.ID = ""
	return nil
}

func (h *hcn) ListLoadBalancers() ([]*LoadBalancer, error) {
	var err error
	query := hcnshim.HostComputeQuery{
		SchemaVersion: hcnshim.SchemaVersion{
			Major: 2,
			Minor: 0,
		},
	}

	// Load HostComputeLoadBalancers
	hcLoadBalancers, err := hcnshim.ListLoadBalancersQuery(query)
	if err != nil {
		return nil, err
	}

	loadBalancers := make([]*LoadBalancer, 0)

	// getPortNProtocol returns Port and Protocol from HostCompute PortMapping
	getPortNProtocol := func(portMappings []hcnshim.LoadBalancerPortMapping) (uint16, uint16, uint32, LoadBalancerPortMappingFlags) {
		for _, portMapping := range portMappings {
			return portMapping.ExternalPort, portMapping.InternalPort, portMapping.Protocol, LoadBalancerPortMappingFlags(portMapping.Flags)
		}
		return 0, 0, 0, LoadBalancerPortMappingFlagsNone
	}

	endpoints, err := h.ListEndpoints()
	endpointMap := make(map[string]*Endpoint)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints {
		endpointMap[endpoint.ID] = endpoint
	}

	getEndpoints := func(endpointIDs []string) []*Endpoint {
		eps := make([]*Endpoint, 0)
		for _, epID := range endpointIDs {
			eps = append(eps, endpointMap[epID])
		}
		return eps
	}

	for _, hcLoadBalancer := range hcLoadBalancers {
		port, targetPort, protocol, pmFlags := getPortNProtocol(hcLoadBalancer.PortMappings)
		ip := ""
		if len(hcLoadBalancer.FrontendVIPs) > 0 {
			ip = hcLoadBalancer.FrontendVIPs[0]
		}

		loadBalancer := &LoadBalancer{
			ID:               hcLoadBalancer.Id,
			Endpoints:        getEndpoints(hcLoadBalancer.HostComputeEndpoints),
			IP:               ip,
			Port:             int32(port),
			TargetPort:       int32(targetPort),
			Protocol:         protocol,
			Flags:            LoadBalancerFlags(hcLoadBalancer.Flags),
			PortMappingFlags: pmFlags,
			SourceVip:        hcLoadBalancer.SourceVIP,
		}
		loadBalancers = append(loadBalancers, loadBalancer)
	}

	return loadBalancers, nil
}

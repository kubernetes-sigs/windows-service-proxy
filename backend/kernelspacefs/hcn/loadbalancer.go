package hcn

import (
	"fmt"
	hcnshim "github.com/Microsoft/hcsshim/hcn"
	"k8s.io/apimachinery/pkg/util/sets"
	"net"
	"strconv"
)

type LoadBalancerFlags hcnshim.LoadBalancerFlags

var (
	// LoadBalancerFlagsNone is the default.
	LoadBalancerFlagsNone = LoadBalancerFlags(hcnshim.LoadBalancerFlagsNone)
	// LoadBalancerFlagsDSR enables Direct Server Return (DSR)
	LoadBalancerFlagsDSR = LoadBalancerFlags(hcnshim.LoadBalancerFlagsDSR)
)

// LoadBalancerPortMappingFlags are special settings on a loadbalancer.
type LoadBalancerPortMappingFlags hcnshim.LoadBalancerPortMappingFlags

var (
	// LoadBalancerPortMappingFlagsNone is the default.
	LoadBalancerPortMappingFlagsNone = LoadBalancerPortMappingFlags(hcnshim.LoadBalancerPortMappingFlagsNone)
	// LoadBalancerPortMappingFlagsILB enables internal loadbalancing.
	LoadBalancerPortMappingFlagsILB = LoadBalancerPortMappingFlags(hcnshim.LoadBalancerPortMappingFlagsILB)
	// LoadBalancerPortMappingFlagsLocalRoutedVIP enables VIP access from the host.
	LoadBalancerPortMappingFlagsLocalRoutedVIP = LoadBalancerPortMappingFlags(hcnshim.LoadBalancerPortMappingFlagsLocalRoutedVIP)
	// LoadBalancerPortMappingFlagsUseMux enables DSR for NodePort access of VIP.
	LoadBalancerPortMappingFlagsUseMux = LoadBalancerPortMappingFlags(hcnshim.LoadBalancerPortMappingFlagsUseMux)
	// LoadBalancerPortMappingFlagsPreserveDIP delivers packets with destination ip as the VIP.
	LoadBalancerPortMappingFlagsPreserveDIP = LoadBalancerPortMappingFlags(hcnshim.LoadBalancerPortMappingFlagsPreserveDIP)
)

// LoadBalancer is a user-oriented definition of an HostComputeEndpoint in its entirety.
type LoadBalancer struct {
	ID string
	// more documentation | these are subsets of global set of endpoints

	Endpoints []*Endpoint

	IP               string
	Flags            LoadBalancerFlags
	PortMappingFlags LoadBalancerPortMappingFlags
	SourceVip        string
	Protocol         uint32
	Port             int32
	TargetPort       int32
}

// Key returns identifier for diffstore.
func (lb *LoadBalancer) Key() string {
	var protocol string
	switch lb.Protocol {
	case 17:
		protocol = "UDP"
	case 132:
		protocol = "SCTP"
	default:
		protocol = "TCP"
	}

	return fmt.Sprintf("%s/%s", net.JoinHostPort(lb.IP, strconv.Itoa(int(lb.Port))), protocol)
}

// Equal compares if two loadBalancers are equal.
func (lb *LoadBalancer) Equal(other *LoadBalancer) bool {
	// 1. check number of endpoints
	if len(lb.Endpoints) != len(other.Endpoints) {
		return false
	}

	// 2. compare endpoints
	getMatchStrings := func(endpoints []*Endpoint) []string {
		strs := make([]string, 0)
		for _, endpoint := range endpoints {
			strs = append(strs, endpoint.ID)
		}
		return strs
	}

	oldSet := sets.New(getMatchStrings(lb.Endpoints)...)
	newSet := sets.New(getMatchStrings(other.Endpoints)...)
	if !newSet.Equal(oldSet) {
		return false
	}

	// 3. compare flags
	if lb.Flags != other.Flags {
		return false
	}

	// 4. compare flags
	if lb.PortMappingFlags != other.PortMappingFlags {
		return false
	}

	// 5. compare keys
	if lb.Key() != other.Key() {
		return false
	}

	return true
}

// String returns string representation of loadBalancer.
func (lb *LoadBalancer) String() string {
	return fmt.Sprintf("<LoadBalancer Key=%s Endpoints=%d ID=%s>", lb.Key(), len(lb.Endpoints), lb.ID)
}

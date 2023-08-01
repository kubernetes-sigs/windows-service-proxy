package kernelspacefs

import (
	"fmt"
	"k8s.io/klog/v2"
	netutils "k8s.io/utils/net"
	"net"
	"os"
)

// detectNodeIP returns the nodeIP used by the proxier
// The order of precedence is:
// 1. config.bindAddress if bindAddress is not 0.0.0.0 or ::
// 2. the primary ip from the Node object, if set
// 3. if no ip is found it defaults to 127.0.0.1 and IPv4
func detectNodeIP(hostname, bindAddress string) net.IP {
	nodeIP := netutils.ParseIPSloppy(bindAddress)
	if nodeIP == nil {
		klog.V(0).InfoS("Can't determine this node's ip, assuming 127.0.0.1; if this is incorrect, please set the --bind-address flag")
		nodeIP = netutils.ParseIPSloppy("127.0.0.1")
	}
	return nodeIP
}

// getNetworkName returns HostComputeNetwork name
func getNetworkName(hnsNetworkName string) (string, error) {
	if len(hnsNetworkName) == 0 {
		klog.V(3).InfoS("Flag --network-name not set, checking environment variable")
		hnsNetworkName = os.Getenv("KUBE_NETWORK")
		if len(hnsNetworkName) == 0 {
			return "", fmt.Errorf("Environment variable KUBE_NETWORK and network-flag not initialized")
		}
	}
	return hnsNetworkName, nil
}

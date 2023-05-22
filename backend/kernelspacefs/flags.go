package kernelspacefs

import (
	"flag"
	"os"
)

var (
	flagDefaultHostname, _ = os.Hostname()
	enableDSR              = flag.Bool("enable-dsr", true, "Set this flag to enable DSR")
	flagMasqueradeAll      = flag.Bool("masquerade-all", false, "Set this flag to set the masq rule for all traffic")
	flagMasqueradeBit      = flag.Int("masquerade-bit", 14, "iptablesMasqueradeBit is the bit of the iptables fwmark"+" space to mark for SNAT Values must be within the range [0, 31]")
	flagBindAddress        = flag.String("bind-address", "0.0.0.0", "The ip address for the proxy server to serve on (set to '0.0.0.0' for all IPv4 interfaces and '::' for all IPv6 interfaces).")
	flagClusterCIDR        = flag.String("cluster-cidr", "100.244.0.0/24", "cluster IPs CIDR")
	sourceVip              = flag.String("source-vip", "100.244.206.65", "Source VIP")
	flagHostname           = flag.String("hostname", flagDefaultHostname, "hostname")
)

//go:build windows
// +build windows

/*
Copyright 2017-2022 The Kubernetes Authors.

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

package kernelspace

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/tools/events"
	klog "k8s.io/klog/v2"
	netutils "k8s.io/utils/net"
	"sigs.k8s.io/kpng/api/localv1"
	"sigs.k8s.io/kpng/client/backendcmd"
	"sigs.k8s.io/kpng/client/localsink"
	"sigs.k8s.io/kpng/client/localsink/decoder"
	"sigs.k8s.io/kpng/client/localsink/filterreset"
	"sigs.k8s.io/kpng/client/serviceevents"
	healthcheck "sigs.k8s.io/windows-service-proxy/pkg/healthcheck"
)

type Backend struct {
	cfg localsink.Config
}

var (
	_ decoder.Interface = &Backend{}

	proxier       *Proxier
	flag          = &pflag.FlagSet{}
	minSyncPeriod time.Duration
	syncPeriod    time.Duration

	recorder events.EventRecorder

	masqueradeAll = flag.Bool("masquerade-all", false, "Set this flag to set the masq rule for all traffic")
	masqueradeBit = flag.Int("masquerade-bit", 14, "iptablesMasqueradeBit is the bit of the iptables fwmark"+" space to mark for SNAT Values must be within the range [0, 31]")
	bindAddress   = flag.String("bind-address", "0.0.0.0", "The IP address for the proxy server to serve on (set to '0.0.0.0' for all IPv4 interfaces and '::' for all IPv6 interfaces).")

	defaultHostname, _ = os.Hostname()
	hostname           = flag.String("hostname", defaultHostname, "hostname")

	clusterCIDR        = flag.String("cluster-cidr", "100.244.0.0/24", "cluster IPs CIDR")
	sourceVip          = flag.String("source-vip", "100.244.206.65", "Source VIP")
	enableDSR          = flag.Bool("enable-dsr", false, "Set this flag to enable DSR")
	healthzBindAddress = flag.String("healthz-bind-address", "0.0.0.0:10256", "The IP address with port for the health check server to serve on (set to '0.0.0.0:10256' for all IPv4 interfaces and '[::]:10256' for all IPv6 interfaces). Set empty to disable.")

	winkernelConfig KubeProxyWinkernelConfiguration
)

func BindFlags(flags *pflag.FlagSet) {
	flags.AddFlagSet(flag)
}

func (b *Backend) BindFlags(flags *pflag.FlagSet) {
	b.cfg.BindFlags(flags)
	BindFlags(flags)
}

/* init will do all initialization for backend registration */
func init() {
	backendcmd.Register("to-winkernel", func() backendcmd.Cmd { return New() })
}

func New() *Backend {
	return &Backend{}
}

func (s *Backend) Sink() localsink.Sink {
	return filterreset.New(decoder.New(serviceevents.Wrap(s)))
}

func (s *Backend) DeleteEndpoint(namespace, serviceName, key string) {
	proxier.endpointsChanges.EndpointUpdate(namespace, serviceName, key, nil)
}

func (s *Backend) SetService(svc *localv1.Service) {
	klog.V(0).InfoS("SetService -> %v", svc)
	proxier.serviceChanges.Update(svc)
}

func (s *Backend) DeleteService(namespace, name string) {
	proxier.serviceChanges.Delete(namespace, name)
}

func (s *Backend) SetEndpoint(
	namespace,
	serviceName,
	key string,
	endpoint *localv1.Endpoint) {

	proxier.endpointsChanges.EndpointUpdate(namespace, serviceName, key, endpoint)
}

func (s *Backend) Reset() {
	/* noop */
}

func (s *Backend) Setup() {
	var (
		err   error
		chErr chan error
	)

	flag.DurationVar(&syncPeriod, "sync-period-duration", 15*time.Second, "sync period duration")

	// todo(knabben) - implement dualstack
	//proxyMode := getProxyMode(string(config.Mode), WindowsKernelCompatTester{})
	//dualStackMode := getDualStackMode(config.Winkernel.NetworkName, DualStackCompatTester{})
	//_ = dualStackMode
	//_ = proxyMode

	// todo(knabben) - marshal Kubeconfiguration configuration file
	winkernelConfig.EnableDSR = *enableDSR
	winkernelConfig.NetworkName = "" // remove from config? proxier gets network name from KUBE_NETWORK env var
	winkernelConfig.SourceVip = *sourceVip

	nodeIP := detectNodeIP(*hostname, *bindAddress)
	klog.InfoS("Detected node IP", "IP", nodeIP.String())

	klog.Info("Starting Windows Kernel Proxier.")
	klog.InfoS("  Cluster CIDR", "clusterCIDR", *clusterCIDR)
	klog.InfoS("  Enable DSR", "enableDSR", *enableDSR)
	klog.InfoS("  Masquerade all traffic", "masqueradeAll", *masqueradeAll)
	klog.InfoS("  Masquerade bit", "masqueradeBit", *masqueradeBit)
	klog.InfoS("  Node ip", "nodeip", nodeIP.String())
	klog.InfoS("  Source VIP", "sourceVip", *sourceVip)

	nodeRef := &v1.ObjectReference{
		Kind:      "node",
		Name:      *hostname,
		Namespace: "",
	}

	var healthzServer healthcheck.ProxierHealthUpdater
	var healthzPort int
	if len(*healthzBindAddress) > 0 {
		healthzServer = healthcheck.NewProxierHealthServer(*healthzBindAddress, 2*syncPeriod, recorder, nodeRef)
		_, port, _ := net.SplitHostPort(*healthzBindAddress)
		healthzPort, _ = strconv.Atoi(port)
	}

	serveHealthz(healthzServer, chErr)

	proxier, err = NewProxier(
		syncPeriod,
		minSyncPeriod,
		*masqueradeAll,
		*masqueradeBit,
		*clusterCIDR,
		*hostname,
		nodeIP,
		recorder,
		healthzServer,
		winkernelConfig,
		healthzPort)

	if err != nil {
		klog.ErrorS(err, "Failed to create an instance of NewProxier")
		panic("could not initialize proxier")
	}

	go proxier.SyncLoop()
}

// detectNodeIP returns the nodeIP used by the proxier
// The order of precedence is:
// 1. config.bindAddress if bindAddress is not 0.0.0.0 or ::
// 2. the primary IP from the Node object, if set
// 3. if no IP is found it defaults to 127.0.0.1 and IPv4
func detectNodeIP(hostname, bindAddress string) net.IP {
	nodeIP := netutils.ParseIPSloppy(bindAddress)
	if nodeIP == nil {
		klog.V(0).InfoS("Can't determine this node's IP, assuming 127.0.0.1; if this is incorrect, please set the --bind-address flag")
		nodeIP = netutils.ParseIPSloppy("127.0.0.1")
	}
	return nodeIP
}

func serveHealthz(hz healthcheck.ProxierHealthUpdater, errCh chan error) {
	if hz == nil {
		return
	}

	fn := func() {
		err := hz.Run()
		if err != nil {
			klog.ErrorS(err, "Healthz server failed")
			if errCh != nil {
				errCh <- fmt.Errorf("healthz server failed: %v", err)
				// if in hardfail mode, never retry again
				blockCh := make(chan error)
				<-blockCh
			}
		} else {
			klog.ErrorS(nil, "Healthz server returned without error")
		}
	}
	go wait.Until(fn, 5*time.Second, wait.NeverStop)
}

func (s *Backend) Sync() {
	klog.V(0).InfoS("backend.Sync()")
	proxier.setInitialized(true)
	proxier.Sync()
}

func (s *Backend) WaitRequest() (nodeName string, err error) {
	klog.V(0).InfoS("wait request")
	name, _ := os.Hostname()
	return name, nil
}

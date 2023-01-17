//go:build windows
// +build windows

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

package kernelspace

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"os"
	"sigs.k8s.io/kpng/client/backendcmd"
	"sigs.k8s.io/kpng/client/localsink"
	kpngmetrics "sigs.k8s.io/kpng/server/pkg/metrics"
	"sigs.k8s.io/windows-service-proxy/backend/kernelspace/metrics"
	"sigs.k8s.io/windows-service-proxy/pkg/healthcheck"
	"strconv"
	"time"

	klog "k8s.io/klog/v2"
)

var (
	masqueradeAll = flag.Bool("masquerade-all", false, "Set this flag to set the masq rule for all traffic")
	masqueradeBit = flag.Int("masquerade-bit", 14, "iptablesMasqueradeBit is the bit of the iptables fwmark"+" space to mark for SNAT Values must be within the range [0, 31]")
	bindAddress   = flag.String("bind-address", "0.0.0.0", "The IP address for the proxy server to serve on (set to '0.0.0.0' for all IPv4 interfaces and '::' for all IPv6 interfaces).")

	defaultHostname, _ = os.Hostname()
	hostname           = flag.String("hostname", defaultHostname, "hostname")
	clusterCIDR        = flag.String("cluster-cidr", "100.244.0.0/24", "cluster IPs CIDR")
	sourceVip          = flag.String("source-vip", "100.244.206.65", "Source VIP")
	enableDSR          = flag.Bool("enable-dsr", true, "Set this flag to enable DSR")

	healthzBindAddress = flag.String("healthz-bind-address", "0.0.0.0:10256", "The IP address with port for the health check server to serve on (set to '0.0.0.0:10256' for all IPv4 interfaces and '[::]:10256' for all IPv6 interfaces). Set empty to disable.")
	metricsBindAddress = flag.String("metrics-bind-address", "0.0.0.0:10257", "The IP address with port for the metrics server to serve on (set to '0.0.0.0:10257' for all IPv4 interfaces and '[::]:10257' for all IPv6 interfaces). Set empty to disable.")
)

type Backend struct {
	cfg localsink.Config
}

func New() *Backend {
	return &Backend{}
}

/* init will do all initialization for backend registration */
func init() {
	backendcmd.Register("to-winkernel", func() backendcmd.Cmd { return New() })
}

func (s *Backend) Setup() {
	var (
		err   error
		chErr chan error
	)

	// todo(knabben) - marshal Kubeconfiguration configuration file
	winkernelConfig.EnableDSR = *enableDSR
	winkernelConfig.NetworkName = "" // remove from config? proxier gets network name from KUBE_NETWORK env var
	winkernelConfig.SourceVip = *sourceVip

	// todo(knabben) - implement dualstack
	//proxyMode := getProxyMode(string(config.Mode), WindowsKernelCompatTester{})
	//dualStackMode := getDualStackMode(config.Winkernel.NetworkName, DualStackCompatTester{})
	//_ = dualStackMode
	//_ = proxyMode

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
	serveMetrics()

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

	metrics.RegisterMetrics()

	if err != nil {
		klog.ErrorS(err, "Failed to create an instance of NewProxier")
		panic("could not initialize proxier")
	}

	go proxier.SyncLoop()
}

func serveMetrics() {
	ctx := context.Background()
	if len(*metricsBindAddress) != 0 {
		prometheus.MustRegister(kpngmetrics.Kpng_k8s_api_events)
		prometheus.MustRegister(kpngmetrics.Kpng_node_local_events)
		kpngmetrics.StartMetricsServer(*metricsBindAddress, ctx.Done())
	}
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

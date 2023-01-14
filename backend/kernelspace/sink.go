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
	"net"
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/events"
	klog "k8s.io/klog/v2"
	netutils "k8s.io/utils/net"
	"sigs.k8s.io/kpng/api/localv1"
	"sigs.k8s.io/kpng/client/localsink"
	"sigs.k8s.io/kpng/client/localsink/decoder"
	"sigs.k8s.io/kpng/client/localsink/filterreset"
	"sigs.k8s.io/kpng/client/serviceevents"
)

var (
	_ decoder.Interface = &Backend{}

	proxier       *Proxier
	flag          = &pflag.FlagSet{}
	minSyncPeriod time.Duration
	syncPeriod    time.Duration

	recorder events.EventRecorder

	winkernelConfig KubeProxyWinkernelConfiguration
)

func init() {
	flag.DurationVar(&syncPeriod, "sync-period-duration", 15*time.Second, "sync period duration")
}

func BindFlags(flags *pflag.FlagSet) {
	flags.AddFlagSet(flag)
}

func (b *Backend) BindFlags(flags *pflag.FlagSet) {
	b.cfg.BindFlags(flags)
	BindFlags(flags)
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

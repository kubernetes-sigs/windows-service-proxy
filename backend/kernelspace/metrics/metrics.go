/*
Copyright 2017 The Kubernetes Authors.

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

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/kubernetes/pkg/proxy/metrics"
)

var registerMetricsOnce sync.Once // nolint

// RegisterMetrics registers kube-proxy metrics for Windows modes.
func RegisterMetrics() { // nolint
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(metrics.SyncProxyRulesLastQueuedTimestamp)
		prometheus.MustRegister(metrics.SyncProxyRulesLatency)
		prometheus.MustRegister(metrics.SyncProxyRulesLastTimestamp)
		prometheus.MustRegister(metrics.EndpointChangesPending)
		prometheus.MustRegister(metrics.EndpointChangesTotal)
		prometheus.MustRegister(metrics.ServiceChangesPending)
		prometheus.MustRegister(metrics.ServiceChangesTotal)
	})
}

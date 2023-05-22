package kernelspacefs

import (
	"sigs.k8s.io/kpng/api/localv1"
)

type ServiceType string

const (
	ClusterIPService    ServiceType = "ClusterIP"
	NodePortService     ServiceType = "NodePort"
	LoadBalancerService ServiceType = "hostComputeLoadBalancer"
)

// String returns ServiceType as string.
func (st ServiceType) String() string {
	return string(st)
}

// SessionAffinity contains data about assigned session affinity.
type SessionAffinity struct {
	ClientIP *localv1.Service_ClientIP
}

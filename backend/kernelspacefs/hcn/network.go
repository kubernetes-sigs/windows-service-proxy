package hcn

import (
	hcnshim "github.com/Microsoft/hcsshim/hcn"
)

type NetworkType hcnshim.NetworkType

const (
	OverlayNetwork NetworkType = "overlay"
	L2Bridge                   = "L2Bridge"
)

type Network struct {
	Name          string
	Id            string
	Type          NetworkType
	RemoteSubnets []*RemoteSubnetInfo
}

type RemoteSubnetInfo struct {
	DestinationPrefix string
	IsolationID       uint16
	ProviderAddress   string
	DrMacAddress      string
}

func (n *Network) IsOverlay() bool {
	return n.Type == OverlayNetwork
}

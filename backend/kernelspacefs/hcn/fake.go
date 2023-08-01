package hcn

// todo: implement me
// fake implements Interface and is used for mocking
type fake struct {
}

func (f fake) GetNetworkByName(s string) (*Network, error) {
	//TODO implement me
	panic("implement me")
}

func (f fake) GetNetworkByID(s string) (*Network, error) {
	//TODO implement me
	panic("implement me")
}

func (f fake) CreateEndpoint(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) UpdateEndpoint(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) DeleteEndpoint(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) ListEndpoints() ([]*Endpoint, error) {
	//TODO implement me
	panic("implement me")
}

func (f fake) CreateLoadBalancer(loadBalancer *LoadBalancer) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) UpdateLoadBalancer(loadBalancer *LoadBalancer) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) DeleteLoadBalancer(loadBalancer *LoadBalancer) error {
	//TODO implement me
	panic("implement me")
}

func (f fake) ListLoadBalancers() ([]*LoadBalancer, error) {
	//TODO implement me
	panic("implement me")
}

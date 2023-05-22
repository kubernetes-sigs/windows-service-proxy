# FullState Windows Kernel Proxy

## Flow
To program windows proxy we majorly deal in two network components:
1. **HostComputeEndpoints** They represent a virtual port in vSwitch.
2. **HostComputeLoadBalancers** Abstraction which creates VFP rules for Stateful NAT

The idea of fullstate backend is to use a **diffstore** to maintain state (network objects) and track changes. So we create two diffstores:
1. **endpointStore** _for tracking hcn.Endpoints_ 
2. **loadBalancerStore** _for tracking hcn.LoadBalancers_

When we process the fullstate callback, we simply add all the required network objects to achieve the desired network data path in diff stores, after consuming the callback we work on deltas tracked by diffstore and create, update and delete objects accordingly. 

## Files
### 1. **hcn/hcn.go** 
Interface for calling host compute network calls and its concrete implementation

### 2. **hcn/fake.go**
Implementation of the interface for testing

### 3. **hcn/endpoint.go**
Endpoint is a user-oriented definition of an HostComputeEndpoint in its entirety.

### 4. **hcn/loadbalancer.go**
LoadBalancer is a user-oriented definition of an HostComputeLoadBalancer in its entirety.

### 5. **hcn/network.go**
Network is a user-oriented definition of an HostComputeNetwork in its entirety.

### 6. **register.go**
Backend registration with KPNG brain

### 7. **store.go**
1. **endpointStore** maintains state for only remote endpoints
2. **loadBalancerStore** maintains state for all load balancers


### 8. **core.go**
1. **PreRun()**  Setting / initializing things
2. **Callback()** KPNG brain calls this callback with a list of ServiceEndpoints (the desired state) which needs to be programmed in the kernel
3. **addServiceEndpointsForClusterIP()**, **addServiceEndpointsForNodePort()** Logic for programming Kubernetes services for Windows kernel, instead of directly programming the network objects we add all the required objects in diffstore
4. **programHostComputeObjects()** This function has side effects, after consuming the callback, this function works on deltas returned by diffstore and programs the kernel accordingly.    





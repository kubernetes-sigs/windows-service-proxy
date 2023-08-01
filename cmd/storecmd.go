package main

import (
	// kernelspace automatic backend register
	_ "sigs.k8s.io/windows-service-proxy/backend/kernelspace"

	// kernelspacefs automatic backend register
	_ "sigs.k8s.io/windows-service-proxy/backend/kernelspacefs"
)

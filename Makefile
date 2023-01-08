.PHONY: run

run:
	go run ./... kube -v=5 to-local to-winkernel --kubeconfig=${HOME}/.kube/config

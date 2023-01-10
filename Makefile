.PHONY: run

run:
	GOOS=windows go run ./... kube -v=5 to-local to-winkernel --kubeconfig=${HOME}/.kube/config

# windows-service-proxy

Home for Docker files and Helm Chart for build and deploying winkernal Kube-proxy as a HostProcess Container

# Is this the official windows kube proxy ? 

The source code for kube-proxy is in tree.  We don't build kube-proxy here, only package it into a HostProcess Container and provide a Helm chart to easy deployment. 

# Testing

This image will be used in Cluster API for Azure e2e tests and sig-windows CI.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/sig-windows)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-windows)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

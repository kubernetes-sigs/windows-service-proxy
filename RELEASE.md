# Release Process

The Kubernetes Windows Host Proces kube-proxy project is released on an as-needed basis. Due to the unique nature of the using the upstream binaries the release process is a bit different than you might be familiar with.

We need to be able to be able to release kube-proxy images anytime new kubernetes releases are done. We also need to be able to release new versions of all our supported images incase there is a bug in the dockerfile.  There is a `kube-proxy-dockerfile-version` file must be updated if the `Dockerfile` changes. .  To accommodate this release process we have two different ways to release:

1.  If the Dockerfile doesn't change but we want to release a new version of kube-proxy we can update the `kube-proxy-versions` file only.  This will publish a new image with that version to the staging registry. Any images add should be released.

2. If the `Dockerfile` changes then `kube-proxy-dockerfile-version` must be updated and will cause all new images in `kube-proxy-versions` to be published to the staging registry. All of these images should be released.
 
## Release a kube-proxy image

1. Open a PR that adds a kubeproxy version to `kube-proxy-versions` or update the `Dockerfile`  and bump the version in `kube-proxy-dockerfile-version`

1. An OWNER promotes the `gcr.io/gcr.io/k8s-staging-win-svc-proxy/kube-proxy` image built with the tag `docker.io/jsturtevant/kube-proxy:v$kubeproxyversion-$kube-proxy-dockerfile-version`
    1. Follow setup steps for `kpromo` from [here](https://github.com/kubernetes-sigs/promo-tools/blob/main/docs/promotion-pull-requests.md#preparing-environment) if needed
    2. Manually tag the desired container image in the [staging registry](https://console.cloud.google.com/gcr/images/k8s-staging-win-svc-proxy?project=k8s-staging-win-svc-proxy) as `v$kubeproxyversion`.  If there already exists an image with that version then tag it `v$kubeproxyversion-1`
    3. Run `kpromo pr` to open a pull request to have tagged container image promoted from staging to release registries

        ```bash
        kpromo pr --project windows-service-proxy --tag v$kubeproxyversion --reviewers "@jayunit100 @jsturtevant @marosset" --fork {your github username}
        ```

    4. Review / merge image promotion PR
    5. Verify the image is available using `docker pull registry.k8s.io/windows-service-proxy/kube-proxy:v$kubeproxyversion`.  The image is pushed to the release repository via the post submit which can take an hour or two to trigger. View results at https://testgrid.k8s.io/sig-k8s-infra-k8sio#post-k8sio-image-promo

## Release the helm chart
2. An OWNER creates a release with by
    1. Navigating to [releases](https://github.com/kubernetes-sigs/windows-service-proxy/releases) and clicking on `Draft a new release`
    2. Selecting the tag for the current release version
    3. Setting the title of the release to the current release version
    4. Clicking `Auto-generate release notes` button (and editing what was generated as appropriate) 
    5. Adding instructions on how to deploy the current release **to the top of the releaes notes** with the following template:

        To deploy:

        ```bash
        helm repo add windows-service-proxy https://raw.githubusercontent.com/kubernetes-sigs/windows-service-proxy/main/helm/repo
        helm install windows-service-proxy --namespace kube-system --version <helmversion>
        ```

    6. Clicking on `Publish Release`
   
3. Update `image.tag` in `charts/chart.yaml` to $VERSION and create new chart package:
    1. Run `helm package charts/gmsa --destination ./helm/repo`. Make sure the resulting tgz file is in the `helm/repo` folder.
    2. Run `helm repo index helm/repo/` to update the helm index
4. The release issue is closed
5. An announcement email is sent to `kubernetes-sig-windows@googlegroups.com` with the subject `[ANNOUNCE] Kubernetes SIG-Windows Windows kube-proxy HPC image $VERSION is Released`
6. An announcement is posted in `#SIG-windows` in the Kubernetes slack.
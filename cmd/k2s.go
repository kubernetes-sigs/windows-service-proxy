/*
Copyright 2021 The Kubernetes Authors.

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

package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kpng/client/backendcmd"
	"sigs.k8s.io/kpng/client/localsink"
	"sigs.k8s.io/kpng/server/jobs/store2localdiff"

	"github.com/spf13/cobra"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// this depends on the kpng server to run the integrated app
	"sigs.k8s.io/kpng/server/jobs/kube2store"
	"sigs.k8s.io/kpng/server/proxystore"
)

var (
	// kubeConfig is the kubeConfig file for the apiserver
	kubeConfig string

	// kubeServer is the location of the external k8s apiserver.  If this is empty, we resort
	// to in-cluster configuration using internal pod service accounts.
	kubeServer string

	kubeClient = &kubernetes.Clientset{}
	k8sCfg     = &kube2store.K8sConfig{}
)

// kube2storeCmd generates the kube-to-store command, which is the "normal" way to run KPNG,
// wherein you read data in from kubernetes, and push it into a store (file, API, or local backend such as NFT).
func kube2storeCmd() *cobra.Command {
	// kube to * command
	k2sCmd := &cobra.Command{
		Use:   "kube",
		Short: "watch Kubernetes API to the globalv1 state",
	}

	flags := k2sCmd.PersistentFlags()
	flags.StringVar(&kubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster. Defaults to envvar KUBECONFIG.")
	flags.StringVar(&kubeServer, "server", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	// k8sCfg is the configuration of how we interact w/ and watch the K8s APIServer
	k8sCfg.BindFlags(k2sCmd.PersistentFlags())

	ctx := setupGlobal()
	store := proxystore.New()
	run := func() {
		kube2storeCmdRun(ctx, store)
	}

	k2sCmd.AddCommand(ToLocalCmd(ctx, store, kube2storeCmdSetup, run))

	return k2sCmd
}

// LocalCmds uses "reflection", i.e. it depends on the hot-loading of backends when
// the imports are called.  the "Registered" function then adds the backends one at a time.
func LocalCmds(run func(sink localsink.Sink) error) (cmds []*cobra.Command) {
	// sink backends
	for _, useCmd := range backendcmd.Registered() {
		backend := useCmd.New()

		cmd := &cobra.Command{
			Use: useCmd.Use,
			RunE: func(_ *cobra.Command, _ []string) error {
				return run(backend.Sink())
			},
		}

		backend.BindFlags(cmd.Flags())
		klog.Infof("Appending discovered command %v", cmd.Name())
		cmds = append(cmds, cmd)
	}

	return
}

// ToLocalCmd reads from the store, and sends these down to a local backend, which we refer to as a Sink.
// See the LocalCmds implementation to understand how we use reflection to load up the individual backends
// such that their command line options are dynamically accepted here.
func ToLocalCmd(ctx context.Context, store *proxystore.Store, storeProducerJobSetup func() (err error), storeProducerJobRun func()) (cmd *cobra.Command) {
	cmd = &cobra.Command{
		Use: "to-local",
	}

	job := &store2localdiff.Job{}

	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) (err error) {
		job.Store = store
		return
	}

	cmd.AddCommand(LocalCmds(func(sink localsink.Sink) error {
		if storeProducerJobSetup != nil {
			if err := storeProducerJobSetup(); err != nil {
				return err
			}
		}
		go storeProducerJobRun()

		job.Sink = sink
		return job.Run(ctx)
	})...)

	return
}

// kube2storeCmdSetup performs any neccessary setup steps that need to happen
// before the kube2store job starts.
func kube2storeCmdSetup() error {
	if kubeConfig == "" {
		kubeConfig = os.Getenv("KUBECONFIG")
	}
	cfg, err := clientcmd.BuildConfigFromFlags(kubeServer, kubeConfig)
	if err != nil {
		return fmt.Errorf("Error building kubeconfig: %w", err)
	}

	kubeClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("Error building kubernetes clientset: %w", err)
	}
	return nil
}

// kube2storeCmdRun kicks off the kube2store job.
func kube2storeCmdRun(ctx context.Context, store *proxystore.Store) {
	kube2store.Job{
		Kube:   kubeClient,
		Store:  store,
		Config: k8sCfg,
	}.Run(ctx)
}

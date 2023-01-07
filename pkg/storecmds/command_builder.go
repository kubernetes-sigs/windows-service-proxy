package storecmds

import (
	"context"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kpng/client/backendcmd"
	"sigs.k8s.io/kpng/client/localsink"
	"sigs.k8s.io/kpng/server/jobs/store2localdiff"
	"sigs.k8s.io/kpng/server/proxystore"
)

// todo(knabben) - split storecmds/command_builder.go in another module
// so backends do not come imported. Remove these functions later.

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

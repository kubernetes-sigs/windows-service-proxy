/*
Copyright 2023 The Kubernetes Authors.

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
	"flag"
	"os"
	"runtime/pprof"

	"k8s.io/klog/v2"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kpng/server/pkg/proxy"
)

var (
	cpuprofile string
)

func init() {
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
}

// main starts the kpng program by running the command sent by the user.  This is the entry point to kpng!
func main() {
	klog.InitFlags(flag.CommandLine)

	cmd := cobra.Command{
		Use: "kpng",
	}

	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	cmd.AddCommand(kube2storeCmd())

	if err := cmd.Execute(); err != nil {
		klog.Fatal(err)
	}
}

// setupGlobal sets up global processes that need to run regardless of what mode you are running KPNG in.
// this is a grab bag where you put stuff that, one way or other, has to happen.

func setupGlobal() (ctx context.Context) {
	ctx, cancel := context.WithCancel(context.Background())

	// handle exit signals
	go func() {
		proxy.WaitForTermSignal()
		cancel()

		proxy.WaitForTermSignal()
		klog.Fatal("forced exit after second term signal")
		os.Exit(1)
	}()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			klog.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	return
}

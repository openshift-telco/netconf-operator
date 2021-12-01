/*
Copyright 2021. Alexis de Talhouët

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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
	"github.com/adetalhouet/netconf-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(netconfv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(
		&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(
		ctrl.GetConfigOrDie(), ctrl.Options{
			Scheme:                     scheme,
			MetricsBindAddress:         metricsAddr,
			Port:                       9443,
			HealthProbeBindAddress:     probeAddr,
			LeaderElection:             enableLeaderElection,
			LeaderElectionID:           "6c6d728d.adetalhouet.io",
			LeaderElectionResourceLock: "configmaps",
		},
	)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = controllers.AddMountPoint(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MountPoint")
	}

	err = controllers.AddLock(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Lock")
	}

	err = controllers.AddUnlock(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Unlock")
	}

	err = controllers.AddEditConfig(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EditConfig")
	}

	err = controllers.AddGet(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Get")
	}

	err = controllers.AddGetConfig(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GetConfig")
	}

	err = controllers.AddCommit(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Commit")
	}

	err = controllers.AddRPC(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RPC")
	}

	err = controllers.AddEstablishSubscription(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EstablishSubscription")
	}

	err = controllers.AddCreateSubscription(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CreateSubscription")
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

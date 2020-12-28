/*


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
	"errors"
	"flag"
	"github.com/prometheus/common/log"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	lbv1 "smartLB/api/v1"
	"smartLB/controllers"
	"smartLB/controllers/customize_webhook"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = lbv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var localMode bool
	var bindInterface string

	flag.StringVar(&metricsAddr, "metrics-addr", ":8081", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&localMode, "enable-local-mode", true, "whether run this controller on lvs server")
	flag.StringVar(&bindInterface, "bind-interface", "", "which interface the vip will bind")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	if localMode {
		log.Info("SmartLB is running using localMode")
		if bindInterface == "" {
			setupLog.Error(errors.New("the bind interface must be give if local mode enabled"), "program exited")
			os.Exit(1)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "81ebad68.my.domain",
	})

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.SmartLBReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("SmartLB"),
		Scheme:        mgr.GetScheme(),
		Recorder:      mgr.GetEventRecorderFor("smartLB recorder"),
		LocalMode:     localMode,
		BindInterface: bindInterface,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "SmartLB")
		os.Exit(1)
	}

	if err = (&lbv1.SmartLB{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "SmartLB")
		os.Exit(1)
	}

	mgr.GetWebhookServer().Register("/request", &customize_webhook.ReqHandler{})
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

/*
Copyright 2021.

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

package initial

import (
	"flag"
	"github.com/cihub/seelog"
	"github.com/hfeng101/Sunwukong/pkg/zaohua/daofa"
	daofametrics "github.com/hfeng101/Sunwukong/pkg/zaohua/daofa/metrics"
	"github.com/hfeng101/Sunwukong/util/consts"
	"github.com/hfeng101/Sunwukong/util/logger"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/controllers"
	//+kubebuilder:scaffold:imports

	zaohuacontrollers "github.com/hfeng101/Sunwukong/pkg/zaohua/controllers"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(sunwukongv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func initZaohuaParam()(){
	var cpuInitializationPeriod time.Duration
	var initialReadinessDelay time.Duration

	flag.DurationVar(&cpuInitializationPeriod, "cpu-initialization-period", ,"The period after pod start when CPU samples might be skipped.")
	flag.DurationVar(&initialReadinessDelay, "initial-readiness-delay", ,"the period after pod start during which readiness changes will be treated as initial readiness.")
}

func InitialAggrator(role string) {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.StringVar(&logLevel, "log-level", "info", "seelog level,support different level such as debug、info、warn、error，info is the auto value if without setting")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a47d2438.my.domain",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	//设置seelog最低日志等级，默认为info
	setLoggerLevel(logLevel)

	//全局控制
	ctx := ctrl.SetupSignalHandler()

	if role == consts.RootCmdRole {
		if err = (&controllers.HoumaoReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Sunwukong"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Sunwukong")
			os.Exit(1)
		}
	}else if role == consts.ZaohuaCmdRole {
		key := types.NamespacedName{
			os.Getenv("HoumaoNamespace"),
			os.Getenv("HoumaoName"),
		}
		object := &sunwukongv1.Houmao{}
		if err := mgr.GetClient().Get(ctx, key, object); err != nil {
			seelog.Errorf("Get object for key:%v failed, err is %v", key, err.Error())
			return
		}

		//建立RestMetricsClient
		rmc, err := resourceclient.NewForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("create  resourceclient failed, err is %v", err.Error())
			return
		}

		discoveryClient,err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("NewDiscoveryClientForConfig  failed, err is %v", err.Error())
			return
		}

		apiVersionGetter := customclient.NewAvailableAPIsGetter(discoveryClient)
		cmc := customclient.NewForConfig(mgr.GetConfig(), mgr.GetClient().RESTMapper(), apiVersionGetter)
		emc,err := externalclient.NewForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("create externalclient failed, err is %v", err.Error())
			return
		}
		restMetricsClientHandle := daofametrics.NewRestMetricsClient(rmc, cmc, emc)
		calculateHandle := daofa.NewCalculateHandle(restMetricsClientHandle)

		zaohuaHandle := daofa.ZaohuaHandle{
			Client: mgr.GetClient(),
			Object: object,
			Ch: calculateHandle,
		}
		//主流程
		zaohuaMode := os.Getenv("ZaohuaMode")
		go zaohuaHandle.StartShifa(ctx, zaohuaMode)

		if err = (&zaohuacontrollers.ZaohuaController{
			Client: mgr.GetClient(),
			//Log:    ctrl.Log.WithName("controllers").WithName("Zaohua"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Zaohua")
			os.Exit(1)
		}
	}else {
		seelog.Errorf("Initial rool:%v is not valid", role)
		os.Exit(1)
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
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

	//wg := sync.WaitGroup{}
	//wg.Done()
}

func setLoggerLevel(level string) {
	switch level {
	case logger.LOG_LEVEL_DEBUG:
	case logger.LOG_LEVEL_INFO:
	case logger.LOG_LEVEL_WARN:
	case logger.LOG_LEVEL_ERROR:
	default:
		seelog.Errorf("Invalid logger level:%v ", level)
		os.Exit(1)
	}

	logger.SwitchLoggerLevel(level)
}

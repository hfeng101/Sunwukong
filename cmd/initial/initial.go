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
	resourcemetricsclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	custommetricsclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalmetricsclient "k8s.io/metrics/pkg/client/external_metrics"
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

type InitialParam struct {
	Role	string
	CpuInitializationPeriod time.Duration		//pod启动时间
	InitialReadinessDelay time.Duration			//pod ready延时等待时间
	DownScaleStabilizationWindow time.Duration	//缩容保护周期，默认5min
	ScaleTolerance float64						//弹性伸缩容忍度，弹性伸缩的最小触发容忍度，默认0.1，即扩容量比原始增加不够0.1则忽略，否则触发扩容
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(sunwukongv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func InitialAggrator(initParam *InitialParam) {
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

	// 创建controller 聚合管理者
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

	// 施法模块初始化流程
	if initParam.Role == consts.ShifaCmdRole {
		if err = (&controllers.HoumaoReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Sunwukong"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Sunwukong")
			os.Exit(1)
		}

		//TODO：接收扩缩容指令的deamon？
	}else if initParam.Role == consts.ZaohuaCmdRole {	// 造化模块初始化流程
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
		rmc, err := resourcemetricsclient.NewForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("create  resourceclient failed, err is %v", err.Error())
			return
		}

		// HPA所需句柄，用户发现非k8s内置资源的操作
		discoveryClient,err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("NewDiscoveryClientForConfig  failed, err is %v", err.Error())
			return
		}

		// 获取新发现资源的可操作接口句柄
		apiVersionGetter := custommetricsclient.NewAvailableAPIsGetter(discoveryClient)
		cmc := custommetricsclient.NewForConfig(mgr.GetConfig(), mgr.GetClient().RESTMapper(), apiVersionGetter)
		emc,err := externalmetricsclient.NewForConfig(mgr.GetConfig())
		if err != nil {
			seelog.Errorf("create externalclient failed, err is %v", err.Error())
			return
		}
		// metric操作handle
		restMetricsClientHandle := daofametrics.NewRestMetricsClient(rmc, cmc, emc)
		// HPA 计算实施的操作句柄
		calculateHandle := daofa.NewCalculateHandle(mgr.GetClient(), restMetricsClientHandle,  initParam.CpuInitializationPeriod, initParam.InitialReadinessDelay, initParam.DownScaleStabilizationWindow, initParam.ScaleTolerance)
		zaohuaHandle := daofa.ZaohuaHandle{
			Client: mgr.GetClient(),
			Object: object,			//造化关联的对象即猴毛，是传入，还是在里面获取？传入不可接收动态变化
			Ch: calculateHandle,
		}

		//主流程，开始施法造化过程（必须前置，否则后续reconcile失败，触发重启造化流程可能会失败）
		zaohuaMode := os.Getenv(consts.ZaohuaModeBeidong)
		go zaohuaHandle.StartZaohua(ctx, zaohuaMode)

		//辅助流程，监测是否要调整施法法术（即仙气变动），或施法目标（即猴毛更换）
		if err = (&zaohuacontrollers.ZaohuaController{
			Client: mgr.GetClient(),
			//Log:    ctrl.Log.WithName("controllers").WithName("Zaohua"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Zaohua")
			os.Exit(1)
		}

		//TODO：接收扩缩容指令的deamon？
	}else {
		seelog.Errorf("Initial rool:%v is not valid", initParam.Role)
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// 增加健康检查探测支持
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

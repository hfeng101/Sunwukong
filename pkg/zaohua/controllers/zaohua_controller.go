package controllers

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/pkg/zaohua/daofa"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	//"k8s.io/apimachinery/"
	"k8s.io/apimachinery/pkg/runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type ZaohuaController struct {
	client.Client

	Scheme *runtime.Scheme
	Cache cache.Cache
}

var (
	//记录上次的配置
	ObjectSpec = sunwukongv1.HoumaoSpec{}
)

func (z *ZaohuaController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error){
	//检测到ScaleTargetRef、Behavor或Max/Min变化了，重新更新主逻辑体
	//ctx := context.Background()
	key := req.NamespacedName
	object := &sunwukongv1.Houmao{}

	if err := z.Client.Get(ctx, key, object); err != nil {
		seelog.Errorf("Get object from key:%v failed, err is %v", key, err.Error())
		return ctrl.Result{},nil
	}

	//配置变更了，记录新的配置，并发起重启
	if !reflect.DeepEqual(object.Spec, ObjectSpec) {
		ObjectSpec = object.Spec
		if err := daofa.RestartShifa(object);err != nil {
			seelog.Errorf("RestartShifa failed, err is %v", err.Error())
		}
	}

	//通过添加eventhandle来做对比
	//gvk := schema.GroupVersionKind{
	//	Group: sunwukongv1.GroupVersion.Group,
	//	Version: sunwukongv1.GroupVersion.Version,
	//	Kind: "Houmao",
	//}
	//if informer,err := z.Cache.GetInformerForKind(ctx, gvk);err != nil {
	//	//添加一个update处理handle
	//	resourceEventHandle := toolscache.ResourceEventHandlerFuncs{
	//		AddFunc: func(obj interface{}){},
	//		UpdateFunc: func(oldObj interface{}, newObj interface{}){
	//			old := oldObj.(sunwukongv1.Houmao)
	//			new := newObj.(sunwukongv1.Houmao)
	//			if !reflect.DeepEqual(old.Spec, new.Spec){
	//				//如果更新了，就重启
	//			}
	//		},
	//		DeleteFunc: func(obj interface{}){},
	//	}
	//	informer.AddEventHandler(resourceEventHandle)
	//}

	return ctrl.Result{},nil
}

func (z *ZaohuaController) ZaohuaStartup() error {

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (z *ZaohuaController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sunwukongv1.Houmao{}).
		Complete(z)
}

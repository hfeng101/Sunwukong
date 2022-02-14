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

package controllers

import (
	"context"
	"github.com/cihub/seelog"
	"github.com/go-logr/logr"
	"github.com/hfeng101/Sunwukong/pkg/status_updator"
	"github.com/hfeng101/Sunwukong/pkg/xianqi_manager"
	"github.com/hfeng101/Sunwukong/util/consts"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
)

// HoumaoReconciler reconciles a Houmao object
type HoumaoReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=sunwukong.my.domain,resources=houmaoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sunwukong.my.domain,resources=houmaoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sunwukong.my.domain,resources=houmaoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Houmao object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *HoumaoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("houmao", req.NamespacedName)

	objectKey := req.NamespacedName
	object := &sunwukongv1.Houmao{}

	xianqiManagerHandle := xianqi_manager.XianqiManagerHandle{
		r.Client,
		types.NamespacedName{
		req.NamespacedName.Namespace,
		consts.XianqiPrefix + req.NamespacedName.Name,
		},
	}

	// 获取status updator句柄，用于更新猴毛status
	statusUpdateHandle := status_updator.NewStatusUpdateHandle(r.Client, object)

	// 根据key值获取猴毛crd，然后确认是否备好了仙气，若未备好则创建一个，否则，若有必要则更新下配置
	if err := r.Client.Get(ctx, objectKey, object); err != nil {
		seelog.Infof("get object for key:%v failed, err is %v", objectKey, err)

		//猴毛已烧毁，确认仙气是否还在，若在，则毁掉
		if err := xianqiManagerHandle.DestroyXianqi(ctx, objectKey); err != nil {
			seelog.Errorf("DestroyXianqi %v failed, err is %v", objectKey, err)
			return ctrl.Result{}, err
		}
	}else {
		//先确保object没有被删除，是可用状态
		if object.ObjectMeta.DeletionTimestamp.IsZero() {
			//在创建仙气前，先确认给猴毛添加了finalizers
			if err := addFinalizers(ctx, r, object); err != nil {
				seelog.Errorf("addFinalizers failed, err is %v", err)
				return ctrl.Result{}, err
			}

			//TODO:在创建仙气前需先确认scaleTargetRef指定的target是否存在，若不存在，则直接忽略

			// 根据猴毛crd，创建或更新仙气，随时候着准备发起造化
			if err := xianqiManagerHandle.CreateOrUpdateXianqi(ctx, objectKey, object); err != nil {
				seelog.Errorf("CreateOrUpdateXianqi for %v failed, err is %v", objectKey, object)
				return ctrl.Result{}, err
			}

			//处理完后，更新猴毛状态
			if object.Status.ShifaPhase != consts.HoumaoPhaseXianqi {
				if err := statusUpdateHandle.UpdateShifaPhase(ctx, consts.HoumaoPhaseXianqi); err != nil {
					seelog.Errorf("UpdateShifaPhase failed, err is %v", err.Error())
					return ctrl.Result{}, err
				}
			}

			xianqiInfo := sunwukongv1.XianqiInfo{
				Name: consts.XianqiPrefix+objectKey.Name,
				Namespace: objectKey.Namespace,
			}
			//如果有变动，则更新
			if !reflect.DeepEqual(xianqiInfo, object.Status.XianqiInfo) {
				if err := statusUpdateHandle.UpdateXianqiInfo(ctx, &xianqiInfo); err != nil {
					seelog.Errorf("UpdateXianqiInfo failed, err is %v", err.Error())
					return ctrl.Result{}, err
				}
			}

		}else {
			//如果object已被删除，则
			seelog.Infof("get object for key:%v failed, err is %v", objectKey, err)

			//猴毛即将烧毁，确认仙气是否还在，若在，则毁掉
			if err := xianqiManagerHandle.DestroyXianqi(ctx, objectKey); err != nil {
				seelog.Errorf("DestroyXianqi %v failed, err is %v", objectKey, err)
				return ctrl.Result{}, err
			}

			//处理完后，更新孙悟空施法结果
			shifaPhase := consts.HoumaoPhaseDestroy
			if err := statusUpdateHandle.UpdateShifaPhase(ctx, shifaPhase); err != nil {
				seelog.Errorf("UpdateStatus failed, err is %v", err.Error())
				return ctrl.Result{}, err
			}

			// 销毁仙气后，删除finalizers
			if err := delFinalizers(ctx, r, object); err != nil {
				seelog.Errorf("delFinalizers failed, err is %v", err)
				return ctrl.Result{}, err
			}

		}


	}

	return ctrl.Result{}, nil
}

// 支持选举
func (r *HoumaoReconciler) NeedLeaderElection() bool {
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *HoumaoReconciler) SetupWithManager(mgr ctrl.Manager) error {


	return ctrl.NewControllerManagedBy(mgr).
		For(&sunwukongv1.Houmao{}).
		Complete(r)
}

// 为新建的houmao，添加filanlizers，保证后续清理流程可控
func addFinalizers(ctx context.Context, r *HoumaoReconciler, obj *sunwukongv1.Houmao) error {
	//前提是未触发删除
	if obj.ObjectMeta.DeletionTimestamp.IsZero(){
		//确认未添加过finalizer，则添加
		if len(obj.ObjectMeta.Finalizers) == 0 {
			addString(&obj.ObjectMeta.Finalizers, consts.HoumaoFilalizer)
			if err := r.Client.Update(ctx, obj); err != nil {
				seelog.Errorf("addFinalizers failed, err is %v", err)
				return err
			}
		}
	}else {
		seelog.Errorf("This object has been excute deleting operation, can not add finalizer")
	}

	return nil
}

// 对于即将销毁的houmao，清理掉filanlizers，保证后续清理流程顺畅
func delFinalizers(ctx context.Context, r *HoumaoReconciler, obj *sunwukongv1.Houmao) error {
	//确认已触发删除了
	if obj.ObjectMeta.DeletionTimestamp.IsZero(){
		seelog.Errorf("This object didn't been deleted, can not delete finalizer")
	}else {
		if len(obj.ObjectMeta.Finalizers) == 0 {
			obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers, consts.HoumaoFilalizer)
			if err := r.Client.Update(ctx, obj); err != nil {
				seelog.Errorf("delFinalizers failed, err is %v", err)
				return err
			}
		}
	}

	return nil
}

func addString(list *[]string, param string) {
	for _,s := range *list {
		//若存在，直接返回
		if  s == param{
			return
		}
	}
	*list = append(*list, param)
}

func removeString(list []string, param string) (result []string) {
	for _,s := range list {
		//若一样，则跳过
		if s != param {
			result = append(result, s)
		}
	}

	return
}

package xianqi_manager

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/util/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type XianqiManagerHandle struct {
	client.Client
	XianqiKey types.NamespacedName
}

// 创建孙悟空的仙气，准备随时触发造化
func (x *XianqiManagerHandle) CreateOrUpdateXianqi(ctx context.Context, key types.NamespacedName, object *sunwukongv1.Houmao) error{
	//先查看仙气是否已存在
	//xianqiKey := types.NamespacedName{
	//	key.Namespace,
	//	consts.XianqiPrefix + key.Name,
	//}
	xianqiObj := &appsv1.Deployment{}
	if err := x.Get(ctx, x.XianqiKey, xianqiObj); err == nil {
		seelog.Infof("Xianqi:%v has been existed, updating it", x.XianqiKey)
		//如果存在，目前仅检测仙气量，并更新，目标及目标名称变更，并不更新重建
		//TODO：是否可以考虑检测object（即猴毛指定的目标）变化，然后重建仙气
		xianqiObj.Spec.Template.Spec.Containers[0].Resources = object.Spec.XianqiLiang
		if err := x.Client.Update(ctx, xianqiObj); err != nil {
			seelog.Errorf("Xianqi: %v update failed, err is %v", x.XianqiKey, err)
			return err
		}

		seelog.Infof("UpdateXianqi:%v successed", x.XianqiKey)
		return nil
	}

	//如果不存在，则新创建
	labels := map[string]string{
		"Type": "Xianqi",
		"Name": consts.XianqiPrefix+key.Name,
		"Namespace": key.Namespace,
		"HoumaoName": key.Name,
		"HoumaoNamespace": key.Namespace,
	}
	labelSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	replicas := int32(1)
	xianqiObj = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			"Deployment",
			"apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: consts.XianqiPrefix +key.Name,
			Namespace: key.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: labelSelector,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name: "Zaohua",
							Image: consts.XianqiImage,
							Resources: object.Spec.XianqiLiang,	//指定配置
						},
					},
				},
			},

		},
	}

	if err := x.Client.Create(ctx, xianqiObj); err != nil {
		seelog.Errorf("Create Xianqi: %v failed, err is %v", xianqiObj,err)
		return err
	}

	return nil
}

// 调整孙悟空的仙气
func (x *XianqiManagerHandle) UpdateXianqi(ctx context.Context, key types.NamespacedName, object *sunwukongv1.Houmao) error {
	//先查看仙气是否已存在
	//xianqiKey := types.NamespacedName{
	//	key.Namespace,
	//	consts.XianqiPrefix + key.Name,
	//}
	xianqiObj := &appsv1.Deployment{}
	if err := x.Get(ctx, x.XianqiKey, xianqiObj); err != nil {
		seelog.Infof("Xianqi:%v has not been existed, cannot update ", x.XianqiKey)
		return err
	}

	xianqiObj.Spec.Template.Spec.Containers[0].Resources = object.Spec.XianqiLiang
	if err := x.Client.Update(ctx, xianqiObj); err != nil {
		seelog.Errorf("Xianqi: %v update failed, err is %v", x.XianqiKey, err)
		return err
	}

	seelog.Infof("UpdateXianqi:%v successed", x.XianqiKey)
	return nil
}

// 猴毛被烧了，销毁孙悟空的仙气
func (x *XianqiManagerHandle) DestroyXianqi(ctx context.Context, key types.NamespacedName) error {
	//先查看仙气是否已存在
	//xianqiKey := types.NamespacedName{
	//	key.Namespace,
	//	consts.XianqiPrefix + key.Name,
	//}
	xianqiObj := &appsv1.Deployment{}
	if err := x.Get(ctx, x.XianqiKey, xianqiObj); err != nil {
		seelog.Infof("Xianqi:%v has been deleted", xianqiObj)
		return nil
	}

	// 删除
	if err := x.Delete(ctx, xianqiObj); err != nil {
		seelog.Errorf("Delete Xianqi:%v failed, error is %v", xianqiObj, err)
		return err
	}

	return nil
}

package daofa

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/util/consts"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"

	//"go/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	//apimeta "k8s.io/apimachinery/pkg/api/meta"
	//autoscalingv2 "k8s.io/api/autoscaling/v2beta2"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
)

var (
	RestartCh = make(chan struct{})
	HoumaoObject = &sunwukongv1.Houmao{}
)

type ZaohuaHandle struct {
	client.Client
	Object *sunwukongv1.Houmao
}

//type ScaleObject struct {
//	//兼容k8s原生hpa，更新会使得zaohua在下一轮开始前重新加载
//	ScaleTargetRef autoscaling.CrossVersionObjectReference	`json:"scaleTargetRef,omitempty"`
//	Metrics autoscaling.MetricSpec							`json:"metrics,omitempty"`
//	Behavor autoscaling.HorizontalPodAutoscalerBehavior		`json:"behavor"`
//	MinReplicas *int										`json:"minReplicas"`
//	MaxReplicas int											`json:"maxReplicas"`
//
//	//要关注的service，关注对应的endpoints变化或服务质量等
//	ServiceName []string	`json:"serviceName"`
//}

func (z *ZaohuaHandle) StartShifa(ctx context.Context, mode string) error{
	//defer ctx.Done()

	// 不断查看监控数据，并根据条件触发造化
	for {
		// 支持主动、被动两种模式触发弹性伸缩
		switch mode {
		case consts.ZaohuaModeZhudong:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		case consts.ZaohuaModeBeidong:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.scaleForEvent(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		default:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		}


		select{
		case <-ctx.Done():
			// 退出
			seelog.Infof("break out, return")
			return nil
		case <- RestartCh:
			// 重新加载对象或行为
			seelog.Infof("Reload houmao config for next zaohua operation")
			z.Object = HoumaoObject.DeepCopy()
		default:
			// 下一轮走起
			seelog.Infof("next round of zaohua")
		}

	}

	return nil
}

func RestartShifa(houmao *sunwukongv1.Houmao) error{
	defer func(){RestartCh <- struct{}{}}()

	reloadZaohuaHandle(houmao)
	return nil
}

func reloadZaohuaHandle(houmao *sunwukongv1.Houmao) error {
	HoumaoObject.Spec = houmao.Spec
	HoumaoObject.Status = houmao.Status
	return nil
}

// 主动检测指标触发伸缩
func (z *ZaohuaHandle)detectAndScale(ctx context.Context, scaleObject *sunwukongv1.Houmao) error {
	//如果scaleObject没有任何数据，怎么处理，空跑？等待下一轮更新重启
	if reflect.DeepEqual(*scaleObject, sunwukongv1.Houmao{}){
		//等待下一次变更触发
		<- RestartCh
		return nil
	}

	// 根据target获取对应workload的scale
	target := scaleObject.Spec.ScaleTargetRef
	targetGV,err := schema.ParseGroupVersion(scaleObject.Spec.ScaleTargetRef.APIVersion)
	if err != nil {
		seelog.Errorf("ParseGroupVersion failed")
		return err
	}
	targetGVK := schema.GroupVersionKind{
		Group: targetGV.Group,
		Version: targetGV.Version,
		Kind: scaleObject.Spec.ScaleTargetRef.Kind,
	}
	// 获取对应gvk资源列表
	//apimeta.RESTMapping{}
	// 获取scale
	scale, err := z.getScaleForTarget(ctx, scaleObject.Spec.ScaleTargetRef.Name, scaleObject.Namespace, targetGVK)
	if err != nil {
		seelog.Errorf("getScaleForTarget failed, err is %v", err.Error())
		return err
	}

	// 获取metrics
	//metrics := scaleObject.Spec.Metrics
	// 计算
	replicas,err := calculate(ctx, targetGVK, scaleObject.Spec.Metrics, int64(scale.Spec.Replicas),  scaleObject.Spec.MaxReplicas, *(scaleObject.Spec.MinReplicas))
	if err != nil {
		seelog.Errorf("calculate replicas failed, err is %v", err.Error())
		return nil
	}

	// 根据柔性策略，实施伸缩
	if err := execScale(ctx, replicas, target); err != nil {

	}
	//同步service变化


	//更新状态，收尾

	if err:=updateStatus(scaleObject);err != nil {

	}

	return nil
}

// 被动接受push上来的事件，然后判断是否做伸缩
func (z *ZaohuaHandle)scaleForEvent(ctx context.Context, object *sunwukongv1.Houmao) error{

	return nil
}

//
func (z *ZaohuaHandle)getScaleForTarget(ctx context.Context, name string, namespace string, targetGVK schema.GroupVersionKind)(autoscalingv1.Scale,error){
	//select {
	//case <- ctx.Done():
	//	seelog.Errorf("exit, get ctx.Done signal!")
	//	return autoscalingv1.Scale{},nil
	//}

	scaleObject := autoscalingv1.Scale{}
	scaleKey := types.NamespacedName{
		Namespace: namespace,
		Name: name,
	}

	if err := z.Client.Get(ctx, scaleKey, &scaleObject); err != nil {
		seelog.Errorf("get scale for key:%v failed, err is %v", scaleKey, err.Error())
		return autoscalingv1.Scale{},err
	}

	return scaleObject,nil
}

// 根据指标计算伸缩结果
func calculate(ctx context.Context, targetGVK schema.GroupVersionKind, metrics []autoscalingv2beta2.MetricSpec, currentReplicas int64, maxReplicas int64, minReplicas int64)(int64, error) {
	//select{
	//case <-ctx.Done():
	//	seelog.Errorf("exit, get ctx.Done signal!")
	//	return replicas, nil
	//}

	desiredReplicas := currentReplicas
	for index, metric := range(metrics){
		switch metric.Type{
		case consts.PodsMetricSourceType:
			replicas,err := CalculateReplicasWithPodsMetricSourceType(ctx, targetGVK, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithPodsMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
			}
		case consts.ResourceMetricSourceType:
			replicas,err := CalculateReplicasWithResourceMetricSourceType(ctx, targetGVK, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithResourceMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
			}
		case consts.ContainerResourceMetricSourceType:
			replicas,err := CalculateReplicasWithContainerResourceMetricSourcetype(ctx, targetGVK, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithContainerResourceMetricSourcetype failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
			}
		case consts.ObjectMetricSourceType:
			replicas,err := CalculateReplicasWithObjectSourceType(ctx, targetGVK, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithObjectSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
			}
		case consts.ExternalMetricSourceType:
			replicas,err := CalculateExternalMetricSourceType(ctx, targetGVK, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateExternalMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
			}
		}
	}

	if desiredReplicas > maxReplicas {
		desiredReplicas = maxReplicas
	}else if desiredReplicas < minReplicas {
		desiredReplicas = minReplicas
	}

	return desiredReplicas, nil
}

func execScale(ctx context.Context, scaledReplicas int64, target interface{}) error{

	if err := ReviseWithBehavor(); err != nil {

	}

	return nil
}

func syncService()error {

	return nil
}

func updateStatus(object *sunwukongv1.Houmao)error {

	return nil
}
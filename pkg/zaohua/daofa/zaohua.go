package daofa

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/util/consts"
	"google.golang.org/api/websecurityscanner/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/labels"

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

	Ch *CalculateHandle
}

type TimestampedRecommendationRecord struct {
	DesiredReplicas	int64
	Timestamp	time.Time
}

type TimestampedScaleEvent struct {
	ReplicaChange int64
	TimeStamp	time.Time
	OutDated 	bool
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
			// 被动接收指标，触发弹性伸缩
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

	currentReplicas := scale.Spec.Replicas
	//selector,err := labels.Parse(scale.Status.Selector)
	// 获取metrics
	//metrics := scaleObject.Spec.Metrics
	// 计算
	//replicas,err := z.calculate(ctx, targetGVK, scaleObject.Spec.Metrics, int64(scale.Spec.Replicas),  scaleObject.Spec.MaxReplicas, *(scaleObject.Spec.MinReplicas))
	replicas,err := z.calculate(ctx, targetGVK, scaleObject, scale)
	if err != nil {
		seelog.Errorf("calculate replicas failed, err is %v", err.Error())
		return nil
	}

	// 根据柔性策略，实施伸缩
	desiredReplicas, err := z.execScale(ctx, replicas, scale, scaleObject.Spec.Behavor)
	if err != nil {

	}
	//同步service变化
	//if err := updateService(ctx, finalReplicas, )

	//更新状态，收尾
	//scaleObject.Status.CurrentZaohuaResult.CurrentReplicas := currentReplicas
	//scaleObject.Status.CurrentZaohuaResult.DesiredReplicas := desiredReplicas

	zaohuaResult := &sunwukongv1.ZaohuaResult{
		time.Now(),
		int64(currentReplicas),
		desiredReplicas,
	}
	//if scaleObject.Status.Last5thZaohuaResult[4] != sunwukongv1.ZaohuaResult{}

	if err := z.updateStatus(zaohuaResult);err != nil {

	}

	return nil
}

// 被动接受push上来的事件，然后判断是否做伸缩
func (z *ZaohuaHandle)scaleForEvent(ctx context.Context, object *sunwukongv1.Houmao) error{

	return nil
}

//获取scale
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
func (z *ZaohuaHandle)calculate(ctx context.Context, targetGVK schema.GroupVersionKind, scaleObject *sunwukongv1.Houmao, scale autoscalingv1.Scale)(replicas int64, metricType string, metricName string, err error) {
	//select{
	//case <-ctx.Done():
	//	seelog.Errorf("exit, get ctx.Done signal!")
	//	return replicas, nil
	//}

	currentReplicas := int64(scale.Status.Replicas)
	maxReplicas := scaleObject.Spec.MaxReplicas
	minReplicas := scaleObject.Spec.MinReplicas

	namespace := scaleObject.Namespace
	metrics := scaleObject.Spec.Metrics
	podSelector,err := labels.Parse(scale.Status.Selector)
	if err != nil {
		seelog.Errorf("labels.Parse for scale.Status.Selector: %v failed, err is %v", scale.Status.Selector, err.Error())
		return 0,"", "", err
	}

	desiredReplicas := currentReplicas
	//metricType := ""
	//metricName := ""
	for _, metric := range(metrics){
		switch metric.Type{
		case consts.ContainerResourceMetricSourceType:		//容器内的resourse配置指标，直指cpu、mem
			replicas, _, err := z.Ch.CalculateReplicasWithContainerResourceMetricSourceType(ctx, namespace, metric, podSelector, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithContainerResourceMetricSourcetype failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricType = consts.ContainerResourceMetricSourceType
				metricName = metric.ContainerResource.Name.String()
			} //容器内的resourse配置指标，直指cpu、mem
		case consts.ResourceMetricSourceType:		//pod内所有容器的resourse配置指标，直指cpu、mem
			replicas, _, err := z.Ch.CalculateReplicasWithResourceMetricSourceType(ctx, namespace, podSelector, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithResourceMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricType = consts.ResourceMetricSourceType
				metricName = metric.Resource.Name.String()
			}
		case consts.PodsMetricSourceType:		//pod内除resourse配置指标之外，由pod自身提供的自定义metrics指标，比如:QPS
			replicas, _, err := z.Ch.CalculateReplicasWithPodsMetricSourceType(ctx, namespace, podSelector, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithPodsMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricType = consts.ContainerResourceMetricSourceType
				metricName = metric.ContainerResource.Name.String()
			}
		case consts.ObjectMetricSourceType:		//非pod本身服务提供的指标，但是k8s内其他资源object提供的指标，如:ingress，一般object是需要汇聚关联的deployment下所有pods的指标总和
			replicas, _, err := z.Ch.CalculateReplicasWithObjectSourceType(ctx, namespace, podSelector, metric, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithObjectSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricType = consts.ObjectMetricSourceType
				metricName = metric.Object.Metric.Name
			}
		case consts.ExternalMetricSourceType:	//自定义指标，来源于k8s本身无关，指标完全取自外部系统
			replicas, _, err := z.Ch.CalculateReplicasWithExternalMetricSourceType(ctx, namespace, metric, podSelector, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateExternalMetricSourceType failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricType = consts.ExternalMetricSourceType
				metricName = metric.External.Metric.Name
			}
		default:
			seelog.Errorf("the metric type:%v is not supported", metric.Type)
			return desiredReplicas, "", "", seelog.Errorf("the metric type:%v is not supported", metric.Type)
		}
	}

	//保证在[min, max]区间内
	if desiredReplicas > maxReplicas {
		desiredReplicas = maxReplicas
	}else if desiredReplicas < *minReplicas {
		desiredReplicas = *minReplicas
	}

	return desiredReplicas, metricType, metricName, nil
}

func (z *ZaohuaHandle)execScale(ctx context.Context, scaledReplicas int64, scale autoscalingv1.Scale, behavor autoscalingv2beta2.HorizontalPodAutoscalerBehavior) (int64, error){

	// 根据behavor设置，调整弹性伸缩行为
	if behavor.ScaleUp != nil && behavor.ScaleDown != nil {
		finalReplicas,err := ReviseWithBehavor(ctx, scaledReplicas, behavor)
		if  err != nil {
			seelog.Errorf("ReviseWithBehavor for behavor:%v failed, err is %v", behavor, err.Error())
		}

		scale.Spec.Replicas = finalReplicas
	}else {
		scale.Spec.Replicas = scaledReplicas
	}

	// 记录弹性伸缩事件
	storageScaleEvent()

	return scale.Spec.Replicas, nil
}

func syncService()error {

	return nil
}

func (z *ZaohuaHandle)updateStatus(ctx context.Context, zaohuaResult *sunwukongv1.ZaohuaResult)error {

	return nil
}
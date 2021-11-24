package daofa

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/pkg/status_updator"
	"github.com/hfeng101/Sunwukong/util/consts"
	//"google.golang.org/api/websecurityscanner/v1"
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
	OriginReplicas = int32(0)

	RestartZaohuaCh = make(chan struct{})
	HoumaoObject = &sunwukongv1.Houmao{}
	RecommendationRecords []TimestampedRecommendationRecord
	ScaleUpEvents []TimestampedScaleEvent
	ScaleDownEvents []TimestampedScaleEvent
)

type ZaohuaHandle struct {
	client.Client
	Object *sunwukongv1.Houmao

	Ch *CalculateHandle
}

type TimestampedRecommendationRecord struct {
	DesiredReplicas	int32
	Timestamp	time.Time
}

type TimestampedScaleEvent struct {
	ReplicaChange int32
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

func (z *ZaohuaHandle) StartZaohua(ctx context.Context, mode string) error{
	//defer ctx.Done()

	// 不断查看监控数据，并根据条件触发造化，一直死循环？
	for {		//TODO：外循环，还是内循环？是否可以改成wait.util
		// 支持主动、被动两种模式触发弹性伸缩
		// TODO：主被动二选一吗？能否同时支持
		switch mode {
		case consts.ZaohuaModeZhudong:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		case consts.ZaohuaModeBeidong:
			// 被动接收指令或事件，触发弹性伸缩
			if err := z.scaleForEvent(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		default:
			// 默认主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		}


		select{
		case <-ctx.Done():		// 正常退出
			seelog.Infof("break out, return")
			return nil
		case <- RestartZaohuaCh:	// 重新加载对象或行为
			seelog.Infof("Reload houmao config for next zaohua operation")
			z.Object = HoumaoObject.DeepCopy()
		default:	// 出现异常，下一轮走起
			seelog.Infof("next round of zaohua")
		}

	}

	return nil
}

// TODO：是否为独立函数，还需再确认
func RestartZaohua(houmao *sunwukongv1.Houmao) error{
	defer func(){RestartZaohuaCh <- struct{}{}}()

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
		<- RestartZaohuaCh
		return nil
	}

	// 根据target获取对应workload的scale
	//target := scaleObject.Spec.ScaleTargetRef
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

	// 记录扩缩容前的初始副本数
	OriginReplicas = scale.Spec.Replicas

	currentReplicas := scale.Spec.Replicas
	//selector,err := labels.Parse(scale.Status.Selector)
	// 获取metrics
	//metrics := scaleObject.Spec.Metrics
	// 计算
	//replicas,err := z.calculate(ctx, targetGVK, scaleObject.Spec.Metrics, int32(scale.Spec.Replicas),  scaleObject.Spec.MaxReplicas, *(scaleObject.Spec.MinReplicas))
	replicas, metricIndex, metricType, metricName, err := z.calculate(ctx, targetGVK, scaleObject, scale)
	if err != nil {
		seelog.Errorf("calculate replicas failed, err is %v", err.Error())
		return nil
	}

	// 如果计算出来不需要做造化演变，则直接退出
	if replicas == currentReplicas{
		seelog.Infof("Do not scale, return")
		return nil
	}

	// 根据柔性策略，实施伸缩
	behavor := scaleObject.Spec.Behavor
	desiredReplicas, err := z.execScale(ctx, replicas, &scale, &behavor)
	if err != nil {
		seelog.Errorf("execScale failed, err is %v", err.Error())
	}

	//同步service变化
	if err := syncService(ctx, desiredReplicas ); err != nil {
		seelog.Errorf("syncService failed, err is %v", err.Error())
	}

	//更新状态，收尾
	zaohuaResult := &sunwukongv1.ZaohuaResult{
		time.Now(),
		currentReplicas,
		desiredReplicas,
		sunwukongv1.MetricStatus{
			//scaleObject.Spec.Metrics[metricIndex],
			true,
			desiredReplicas-currentReplicas,
			metricIndex,
			metricType,
			metricName,
		},
	}
	//if scaleObject.Status.Last5thZaohuaResult[4] != sunwukongv1.ZaohuaResult{}

	//变更完，将变更结果更新到object中去
	if err := z.updateStatus(ctx, zaohuaResult);err != nil {
		seelog.Errorf("updateStatus failed, err is %v", err.Error())
		return err
	}

	return nil
}

// 被动接受push上来的指令或事件，然后判断是否做伸缩
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
func (z *ZaohuaHandle)calculate(ctx context.Context, targetGVK schema.GroupVersionKind, scaleObject *sunwukongv1.Houmao, scale autoscalingv1.Scale)(replicas int32, metricIndex int32, metricType string, metricName string, err error) {
	//select{
	//case <-ctx.Done():
	//	seelog.Errorf("exit, get ctx.Done signal!")
	//	return replicas, nil
	//}

	currentReplicas := int32(scale.Status.Replicas)
	maxReplicas := scaleObject.Spec.MaxReplicas
	minReplicas := scaleObject.Spec.MinReplicas

	namespace := scaleObject.Namespace
	metrics := scaleObject.Spec.Metrics
	podSelector,err := labels.Parse(scale.Status.Selector)
	if err != nil {
		seelog.Errorf("labels.Parse for scale.Status.Selector: %v failed, err is %v", scale.Status.Selector, err.Error())
		return 0,0,"", "", err
	}

	desiredReplicas := currentReplicas
	//metricType := ""
	//metricName := ""
	for index, metric := range(metrics){
		switch metric.Type{
		case consts.ContainerResourceMetricSourceType:		//容器内的resourse配置指标，直指cpu、mem
			replicas, _, err := z.Ch.CalculateReplicasWithContainerResourceMetricSourceType(ctx, namespace, metric, podSelector, currentReplicas)
			if err != nil {
				seelog.Errorf("CalculateReplicasWithContainerResourceMetricSourcetype failed, err is %v", err.Error())
				continue
			}
			if desiredReplicas < replicas {
				desiredReplicas = replicas
				metricIndex = int32(index)
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
				metricIndex = int32(index)
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
				metricIndex = int32(index)
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
				metricIndex = int32(index)
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
				metricIndex = int32(index)
				metricType = consts.ExternalMetricSourceType
				metricName = metric.External.Metric.Name
			}
		default:
			seelog.Errorf("the metric type:%v is not supported", metric.Type)
			return desiredReplicas, 0,"", "", seelog.Errorf("the metric type:%v is not supported", metric.Type)
		}
	}

	//保证在[min, max]区间内
	if desiredReplicas > maxReplicas {
		desiredReplicas = maxReplicas
	}else if desiredReplicas < *minReplicas {
		desiredReplicas = *minReplicas
	}

	return desiredReplicas, metricIndex, metricType, metricName, nil
}

func (z *ZaohuaHandle)execScale(ctx context.Context, scaledReplicas int32, scale *autoscalingv1.Scale, behavor *autoscalingv2beta2.HorizontalPodAutoscalerBehavior) (int32, error){
	preReplicas := scale.Spec.Replicas
	finalReplicas := scaledReplicas
	// 根据behavor设置，调整弹性伸缩行为
	if behavor.ScaleUp != nil && behavor.ScaleDown != nil {
		if scaledReplicas > scale.Spec.Replicas{
			finalReplicas,err := ReviseWithBehavor(ctx, scaledReplicas, scale.Spec.Replicas, behavor, &RecommendationRecords, &ScaleUpEvents)
			if  err != nil {
				seelog.Errorf("ReviseWithBehavor for behavor:%v failed, err is %v", behavor, err.Error())
				return 0, err
			}

			scale.Spec.Replicas = finalReplicas
		}else {
			finalReplicas,err := ReviseWithBehavor(ctx, scaledReplicas, scale.Spec.Replicas, behavor, &RecommendationRecords, &ScaleDownEvents)
			if  err != nil {
				seelog.Errorf("ReviseWithBehavor for behavor:%v failed, err is %v", behavor, err.Error())
				return 0, err
			}

			scale.Spec.Replicas = finalReplicas
		}

	}

	scale.Spec.Replicas = finalReplicas

	// 下发造化演变结果
	//scaleUpdateOptions := client.UpdateOptions{
	//
	//}
	if err := z.Client.Update(ctx, scale, &client.UpdateOptions{}); err != nil {
		seelog.Errorf("Update scale failed, err is %v", err.Error())
		return 0, err
	}

	// 记录弹性伸缩事件
	storageScaleEvent(behavor, preReplicas, finalReplicas)

	return scale.Spec.Replicas, nil
}

// 造化后更新关联的service
func syncService(ctx context.Context, replicas int32)error {

	return nil
}

func (z *ZaohuaHandle)updateStatus(ctx context.Context, zaohuaResult *sunwukongv1.ZaohuaResult)error {
	zaohuaRecord := z.Object.Status.ZaohuaRecord
	preZaohuaResult := zaohuaRecord.CurrentZaohuaResult
	zaohuaRecord.CurrentZaohuaResult = *zaohuaResult
	lenOfLast5thZaohuaResult := len(zaohuaRecord.Last5thZaohuaResult)
	if lenOfLast5thZaohuaResult < 5 {
		// 留存过去的5次造化记录
		zaohuaRecord.Last5thZaohuaResult[lenOfLast5thZaohuaResult] = preZaohuaResult
	}else {
		//留新去旧,先往前挪，再补充最后一个
		for i := 0; i < lenOfLast5thZaohuaResult - 1; i++ {
			zaohuaRecord.Last5thZaohuaResult[i] = zaohuaRecord.Last5thZaohuaResult[i+1]
		}
		zaohuaRecord.Last5thZaohuaResult[lenOfLast5thZaohuaResult - 1] = preZaohuaResult
	}

	// 如果是第一次扩缩容，则把原始副本数记录下来
	if OriginReplicas != 0 && zaohuaRecord.OriginReplicas == 0 {
		zaohuaRecord.OriginReplicas = OriginReplicas
	}

	statusUpdateHandle := status_updator.NewStatusUpdateHandle(z.Client, z.Object)
	if err := statusUpdateHandle.UpdateZaohuaRecord(ctx, &zaohuaRecord); err != nil {
		seelog.Errorf("UpdateZaohuaRecord failed, err is %v", err.Error())
		return err
	}

	return nil
}

func storageScaleEvent(behavor *autoscalingv2beta2.HorizontalPodAutoscalerBehavior,  preReplicas, newReplicas int32) {
	if behavor == nil {
		seelog.Errorf("there is no behavor, return ")
		return
	}

	var oldSampleIndex int
	var longestPolicyPeriod int32
	foundOldSample := false
	if newReplicas > preReplicas {
		longestPolicyPeriod = getLongestPolicyPeriod(behavor.ScaleUp)

		markScaleUpEventsOutdated(longestPolicyPeriod)

		replicaChange := newReplicas - preReplicas

		for i, scaleUpEvent := range ScaleUpEvents {
			if scaleUpEvent.OutDated {
				foundOldSample = true
				oldSampleIndex = i
			}
		}

		newEvent := TimestampedScaleEvent{replicaChange, time.Now(), false}
		if foundOldSample {
			ScaleUpEvents[oldSampleIndex] = newEvent
		}else {
			ScaleUpEvents = append(ScaleUpEvents, newEvent)
		}
	}else {
		longestPolicyPeriod = getLongestPolicyPeriod(behavor.ScaleDown)
		markScaleDownEventsOutdated(longestPolicyPeriod)
		replicaChange := preReplicas - newReplicas

		for i, scaleDownEvent := range ScaleDownEvents {
			if scaleDownEvent.OutDated {
				foundOldSample = true
				oldSampleIndex = i
			}
		}

		newEvent := TimestampedScaleEvent{replicaChange, time.Now(), false}
		if foundOldSample {
			ScaleDownEvents[oldSampleIndex] = newEvent
		}else {
			ScaleDownEvents = append(ScaleDownEvents, newEvent)
		}
	}

	return
}

func getLongestPolicyPeriod(scaleRules *autoscalingv2beta2.HPAScalingRules)(longestPolicyPeriod int32) {

	for _, policy := range(scaleRules.Policies) {
		if policy.PeriodSeconds > longestPolicyPeriod {
			longestPolicyPeriod = policy.PeriodSeconds
		}
	}

	return longestPolicyPeriod
}

func markScaleUpEventsOutdated(longestPolicyPeriod int32){
	period := time.Second * time.Duration(longestPolicyPeriod)
	cutoff := time.Now().Add(-period)
	for i, event := range(ScaleUpEvents){
		if event.TimeStamp.Before(cutoff) {
			ScaleUpEvents[i].OutDated = true
		}
	}

	return
}

func markScaleDownEventsOutdated(longestPolicyPeriod int32){
	period := time.Second * time.Duration(longestPolicyPeriod)
	cutoff := time.Now().Add(-period)
	for i, event := range(ScaleDownEvents) {
		if event.TimeStamp.Before(cutoff) {
			ScaleDownEvents[i].OutDated = true
		}
	}

	return
}

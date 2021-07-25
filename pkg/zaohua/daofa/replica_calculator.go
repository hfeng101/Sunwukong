package daofa

import (
	"context"
	"github.com/cihub/seelog"
	metricsclient "github.com/hfeng101/Sunwukong/pkg/zaohua/daofa/metrics"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"math"

	//"k8s.io/kubernetes/pkg/controller/podautoscaler/metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
)

type CalculateHandle struct{
	client.Client
	RestMetricsClientHandle *metricsclient.RestMetricsClient
	CpuInitializationPeriod time.Duration
	InitialReadinessDelay time.Duration
	DownScaleStabilizationWindow time.Duration
	ScaleTolerance float64
}

func NewCalculateHandle(restMetricsClientHandle *metricsclient.RestMetricsClient, clientHandle client.Client, cpuInitializationPeriod time.Duration, initialReadinessDelay time.Duration, downScaleStabilizationWindow time.Duration, scaleTolerance float64) *CalculateHandle {
	return &CalculateHandle{
		clientHandle,
		restMetricsClientHandle,
		cpuInitializationPeriod,
		initialReadinessDelay,
		downScaleStabilizationWindow,
		scaleTolerance,
	}
}

//根据给定的PodsMetric计算出弹性伸缩的结果，根据podSelector找到对应的pod实例
func (c *CalculateHandle) CalculateReplicasWithPodsMetricSourceType(ctx context.Context, namespace string, podSelector labels.Selector, metric autoscalingv2beta2.MetricSpec, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	//if metric.
	//metricSelector,err := labels.Parse(metric.Pods.Metric.Selector.String())
	metricSelector,err := metav1.LabelSelectorAsSelector(metric.Pods.Metric.Selector)

	podMetricsInfo, timestamp, err := c.RestMetricsClientHandle.GetRawMetric(metric.Pods.Metric.Name, namespace, podSelector, metricSelector)
	if err != nil {
		seelog.Errorf("GetRawMetric failed, err is %v", err.Error())
		return 0, time.Time{}, nil
	}

	//replicaCount, utilization, err := c.calculatePlainMetricReplicas(podMetricsInfo, currentReplicas, metric.Pods.Target.AverageValue.MilliValue(), namespace, podSelector, "", v1.ResourceName(""))
	replicaCount, _, err := c.calculatePlainMetricReplicas(podMetricsInfo, currentReplicas, metric.Pods.Target.AverageValue.MilliValue(), namespace, podSelector, "", v1.ResourceName(""))

	//if replicaCount
	replicas = replicaCount

	return replicas, time.Time{}, nil
}

func (c * CalculateHandle)calculatePlainMetricReplicas(metrics metricsclient.PodMetricsInfo, currentReplicas int32, targetUtilization int64, namespace string, podSelector labels.Selector, container string, resource v1.ResourceName)(replicaCount int32, utilization int64, err error){
	ctx := context.Background()
	objectList := &v1.PodList{}
	//objectList := &appsv1beta2.Deployment{}
	listOptions := client.ListOptions{
		Namespace: namespace,
		LabelSelector: podSelector,
	}

	if err := c.Client.List(ctx, objectList, &listOptions); err != nil {
		seelog.Errorf("List pods failed, err is %v", err.Error())
		return 0,0, err
	}

	if len(objectList.Items) == 0 {
		seelog.Errorf("no pods returned by selector:%v", podSelector.String())
		return 0,0, seelog.Errorf("no pods returned by selector:%v", podSelector.String())
	}

	readyPodCount, unreadyPods, missingPods, ignoredPods := groupPods(objectList, metrics, v1.ResourceName(""), c.CpuInitializationPeriod, c.InitialReadinessDelay)

	//清除未ready的metric指标数据
	for _,podName := range(unreadyPods.UnsortedList()){
		delete(metrics, podName)
	}
	//清除未ready的metric指标数据
	for _,podName := range(ignoredPods.UnsortedList()){
		delete(metrics, podName)
	}

	//requests, err := calculatePodRequests(objectList, container, resource)
	//if err != nil {
	//	seelog.Errorf("calculatePodRequests failed, err is %v", err.Error())
	//	return 0, 0, seelog.Errorf("calculatePodRequests failed, err is %v", err.Error())
	//}

	if len(metrics) == 0 {
		seelog.Errorf("without any available metrics for ready pods")
		return 0, 0, seelog.Errorf("without any available metrics for ready pods")
	}

	// 计算资源当前利用率与目标的比率，以及当前利用率
	//usageRatio, utilization, rawUtilization, err := metricsclient.GetResourceUtilizationRatio(metrics, requests, targetUtilization)
	usageRatio, utilization := metricsclient.GetMetricUtilizationRatio(metrics, targetUtilization)

	//如果有未ready的pod，并且利用率与目标的比率大于1，则直接执行弹性伸缩，不再对unreadyPods做动态平衡调整
	rebalanceIgnored := len(unreadyPods) >0 && usageRatio > 1.0

	//如果做再平衡，且pod指标都全
	if !rebalanceIgnored && len(missingPods) == 0 {
		//计算弹性系数在容忍度内，则不调整了，直接返回
		if math.Abs(1.0-usageRatio) <= c.ScaleTolerance{
			return int32(math.Ceil(usageRatio * float64(readyPodCount))), utilization, nil
		}
	}

	//如果存在未获取到指标的pod
	if len(missingPods) > 0 {
		//若利用率与目标的比率小于1，则把该pod的指标值默认调整为最大为目标值
		if usageRatio < 1.0 {
			for podName := range(missingPods){
				metrics[podName] = metricsclient.PodMetric{Value: targetUtilization}
			}
		}else {
			//若且利用率与目标的比率大于1，则把该pod的指标值默认调整为最小为0，忽略该pod
			for podName := range missingPods {
				metrics[podName] = metricsclient.PodMetric{Value: 0}
			}
		}
	}

	//如果不对missingPods做再平衡调整，则直接忽略unreadyPods
	if rebalanceIgnored{
		for podName:= range unreadyPods{
			metrics[podName] = metricsclient.PodMetric{Value: 0}
		}
	}

	//调整后重新计算
	newUsageRatio, _ := metricsclient.GetMetricUtilizationRatio(metrics, targetUtilization)

	//如果再容忍度内，或前后两次计算结果不一致，则直接返回
	if math.Abs(1.0-newUsageRatio) <= c.ScaleTolerance || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		return currentReplicas, utilization, nil
	}

	//计算返回结果
	newReplicas := int32(math.Ceil(newUsageRatio * float64(len(metrics))))
	if (newUsageRatio < 1.0 && newReplicas > currentReplicas) || (newUsageRatio > 1.0 && newReplicas < currentReplicas) {
		return currentReplicas, utilization, nil
	}

	return newReplicas, utilization, nil
}

//步骤说明
//1. 获取所有指标，匹配label
//2. 获取pod列表，并将pod分类成unreadyPods, missingPods, ignoredPods
//3. 根据指标，计算使用率，及与目标值的比率，判断是否触发scale动作
//4. 根据实况判断是否要做平衡调整，以及实时做出调整
//5. 重新计算最终结果，及调整后的实例数
func (c *CalculateHandle)CalculateReplicasWithResourceMetricSourceType(ctx context.Context, namespace string, podSelector labels.Selector, metric autoscalingv2beta2.MetricSpec, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	//desiredReplicas := int32(0)
	//获取实例及其指标数据
	resource := metric.Resource.Name
	//container := metric.Resource
	podMetrics, timestamp, err := c.RestMetricsClientHandle.GetResouceMetric(resource, namespace, podSelector, "container")
	if err != nil {
		seelog.Errorf("GetResouceMetric failed, err is %v", err)
		return 0, time.Time{}, err
	}

	if len(podMetrics) == 0{
		seelog.Errorf("without any available metrics for ready pods")
		return 0, time.Time{}, seelog.Errorf("without any available metrics for ready pods")
	}

	objectList := &v1.PodList{}
	//objectList := &appsv1beta2.Deployment{}
	listOptions := client.ListOptions{
		Namespace: namespace,
		LabelSelector: podSelector,
	}
	if err := c.Client.List(ctx, objectList, &listOptions); err != nil {
		seelog.Errorf("List pods failed, err is %v", err.Error())
		return 0,time.Time{}, err
	}
	if len(objectList.Items) == 0 {
		seelog.Errorf("no pods returned by selector:%v", podSelector.String())
		return 0,time.Time{}, seelog.Errorf("no pods returned by selector:%v", podSelector.String())
	}

	readyPodCount, unreadyPods, missingPods, ignoredPods := groupPods(objectList, podMetrics, resource, c.CpuInitializationPeriod, c.InitialReadinessDelay)
	//清除undreadyPod的metric指标数据
	for _,podName := range(unreadyPods.UnsortedList()){
		delete(podMetrics, podName)
	}
	//清除ignoredPod的metric指标数据
	for _,podName := range(ignoredPods.UnsortedList()){
		delete(podMetrics, podName)
	}

	requests, err := calculatePodRequests(objectList, "", resource)
	if err != nil {
		seelog.Errorf("calculatePodRequests failed, err is %v", err)
		return 0, time.Time{}, err
	}

	targetUtilization := metric.Resource.Target.AverageValue.MilliValue()
	//usageRatio, utilization, rawUtilization, err := metricsclient.GetResourceUtilizationRatio(podMetrics, requests, targetUtilization)
	usageRatio, _, _, err := metricsclient.GetResourceUtilizationRatio(podMetrics, requests, targetUtilization)
	if err != nil {
		seelog.Errorf("GetResourceUtilizationRatio failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	rebalanceIgnored := len(unreadyPods)>0 && usageRatio>1.0

	if !rebalanceIgnored && len(missingPods) == 0 {
		//计算弹性系数在容忍度内，则不调整了，直接返回
		if math.Abs(1.0-usageRatio) <= c.ScaleTolerance{
			return currentReplicas,  timestamp, nil
		}
		return int32(math.Ceil(usageRatio * float64(readyPodCount))), timestamp, nil
	}

	if len(missingPods) > 0 {
		if usageRatio < 1.0 {
			for podName := range(missingPods){
				podMetrics[podName] = metricsclient.PodMetric{Value: targetUtilization}
			}
		}else {
			//若且利用率与目标的比率大于1，则把该pod的指标值默认调整为最小为0，忽略该pod
			for podName := range missingPods {
				podMetrics[podName] = metricsclient.PodMetric{Value: 0}
			}
		}
	}

	//如果不对missingPods做再平衡调整，则直接忽略unreadyPods
	if rebalanceIgnored{
		for podName:= range unreadyPods{
			podMetrics[podName] = metricsclient.PodMetric{Value: 0}
		}
	}

	//调整后重新计算
	newUsageRatio, _ := metricsclient.GetMetricUtilizationRatio(podMetrics, targetUtilization)

	//如果再容忍度内，或前后两次计算结果不一致，则直接返回
	if math.Abs(1.0-newUsageRatio) <= c.ScaleTolerance || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		//return currentReplicas, int32(utilization), nil
		return currentReplicas, timestamp, nil
	}

	//计算返回结果
	newReplicas := int32(math.Ceil(newUsageRatio * float64(len(podMetrics))))
	if (newUsageRatio < 1.0 && newReplicas > currentReplicas) || (newUsageRatio > 1.0 && newReplicas < currentReplicas) {
		//return currentReplicas, utilization, nil
		return currentReplicas, timestamp, nil
	}

	//return newReplicas, utilization, nil
	return newReplicas, timestamp, nil
}

func (c *CalculateHandle)CalculateReplicasWithContainerResourceMetricSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, podSelector labels.Selector, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	//获取指标数据
	resourceName := metric.ContainerResource.Name
	container := metric.ContainerResource.Container
	podMetrics, timestamp, err := c.RestMetricsClientHandle.GetResouceMetric(resourceName, namespace, podSelector, container)
	if err != nil {
		seelog.Errorf("GetResouceMetric failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}
	if len(podMetrics) == 0 {
		seelog.Errorf("")
		return 0, time.Time{}, seelog.Errorf("without any available metrics for ready pods")
	}

	//获取pod列表
	objectList := &v1.PodList{}
	listOptions := &client.ListOptions{
		LabelSelector: podSelector,
		Namespace: namespace,
	}
	if err := c.Client.List(ctx, objectList, listOptions); err != nil {
		seelog.Errorf("List object failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}
	if len(objectList.Items) == 0 {
		seelog.Errorf("no pods returned by selector:%v", podSelector.String())
		return 0, time.Time{}, seelog.Errorf("no pods returned by selector:%v", podSelector.String())
	}

	//将pod梳理分类
	readyPodCount, unreadyPods, missingPods, ignoredPods := groupPods(objectList, podMetrics, "", c.CpuInitializationPeriod, c.InitialReadinessDelay)

	//清除undreadyPod的metric指标数据
	for _,podName := range(unreadyPods.UnsortedList()){
		delete(podMetrics, podName)
	}
	//清除ignoredPod的metric指标数据
	for _,podName := range(ignoredPods.UnsortedList()){
		delete(podMetrics, podName)
	}

	//获取pod列表里指定的request信息
	podRequests, err := calculatePodRequests(objectList, container, "")
	if err != nil {
		seelog.Errorf("calculatePodRequests failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	//计算利用率及与其目标值的比率
	targetAverageUtilization := metric.ContainerResource.Target.AverageValue.MilliValue()
	//usageRatio, utilization, rawAverageValue, err := metricsclient.GetResourceUtilizationRatio(podMetrics, podRequests, targetAverageUtilization, )
	usageRatio, _, _, err := metricsclient.GetResourceUtilizationRatio(podMetrics, podRequests, targetAverageUtilization, )
	if err != nil {
		seelog.Errorf("GetResourceUtilizationRatio failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	//重新平衡调整，如果usageRatio>1.0则预示着要触发弹性伸缩，但len(unreadyPods)>0则表示还有未准备好的pod，需要对unreadyPod做调整平衡后再做决策
	rebalanceIgnored := len(unreadyPods)>0 && usageRatio>1.0

	//如果不需要重新平衡调整，且无pod丢失指标，则判断返回
	if !rebalanceIgnored && len(missingPods) == 0 {
		//计算弹性系数在容忍度内，则不调整了，直接返回
		if math.Abs(1.0-usageRatio) <= c.ScaleTolerance{
			return currentReplicas, timestamp, nil
		}
		return int32(math.Ceil(usageRatio * float64(readyPodCount))), timestamp, nil
	}

	//如果有pod丢失指标
	if len(missingPods) > 0 {
		if usageRatio < 1.0 {
			for podName := range(missingPods){
				podMetrics[podName] = metricsclient.PodMetric{Value: targetAverageUtilization}
			}
		}else {
			//若且利用率与目标的比率大于1，则把该pod的指标值默认调整为最小为0，忽略该pod
			for podName := range missingPods {
				podMetrics[podName] = metricsclient.PodMetric{Value: 0}
			}
		}
	}

	//如果不对missingPods做再平衡调整，则直接忽略unreadyPods
	if rebalanceIgnored{
		for podName:= range unreadyPods{
			podMetrics[podName] = metricsclient.PodMetric{Value: 0}
		}
	}

	//调整后重新计算
	newUsageRatio, _ := metricsclient.GetMetricUtilizationRatio(podMetrics, targetAverageUtilization)

	//如果再容忍度内，或前后两次计算结果不一致，则直接返回
	if math.Abs(1.0-newUsageRatio) <= c.ScaleTolerance || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		return currentReplicas, time.Time{}, nil
	}

	//计算最终决策结果
	newReplicas := int32(math.Ceil(newUsageRatio * float64(len(podMetrics))))
	if (newUsageRatio < 1.0 && newReplicas > currentReplicas) || (newUsageRatio > 1.0 && newReplicas < currentReplicas) {
		return currentReplicas, time.Time{}, nil
	}

	return newReplicas,timestamp, nil
}

func (c *CalculateHandle)CalculateReplicasWithObjectSourceType(ctx context.Context, namespace string, podSelector labels.Selector, metric autoscalingv2beta2.MetricSpec, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	if metric.Object.Target.Type == autoscalingv2beta2.ValueMetricType {
		replicas, timestamp, err = c.getReplicasWithObjectSourceType(ctx, namespace, metric, podSelector, currentReplicas)
		if err != nil {
			seelog.Errorf("getReplicasWithObjectSourceType failed, err is %v", err.Error())
			return 0, time.Time{}, err
		}
	}else if metric.Object.Target.Type == autoscalingv2beta2.AverageValueMetricType {
		replicas, timestamp, err = c.getReplicasForPerPodWithObjectSourceType(ctx, namespace, metric, currentReplicas)
		if err != nil {
			seelog.Errorf("getReplicasForPerPodWithObjectSourceType failed, err is %v", err.Error())
			return 0, time.Time{}, err
		}
	}else if metric.Object.Target.Type == autoscalingv2beta2.UtilizationMetricType {
		replicas = currentReplicas
	}else {
		seelog.Errorf("metric.Object.Target.Type:%v is not supported", metric.Object.Target.Type)
		return 0, time.Time{}, seelog.Errorf("metric.Object.Target.Type:%v is not supported", metric.Object.Target.Type)
	}

	return replicas,timestamp, nil
}

//按object累计总值评估
func (c *CalculateHandle)getReplicasWithObjectSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, podSelector labels.Selector, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	metricSelector,err := metav1.LabelSelectorAsSelector(metric.Object.Metric.Selector)
	if err != nil {
		seelog.Errorf("LabelSelectorAsSelector failed, err is %v", err)
		return 0, time.Time{}, err
	}

	objectRef := &metric.Object.DescribedObject
	metricName := metric.Object.Metric.Name
	utilization, timestamp, err := c.RestMetricsClientHandle.GetObjectMetric(metricName, namespace, objectRef, metricSelector)
	if err != nil {
		seelog.Errorf("GetObjectMetric failed, err is %v", err.Error())
		return 0,  time.Time{}, err
	}

	targetUtilization := metric.Object.Target.Value.MilliValue()
	usageRatio := float64(utilization)/float64(targetUtilization)
	c.getUsageRatioReplicas(ctx, currentReplicas, usageRatio, namespace, podSelector)

	return replicas,timestamp, nil
}

func (c *CalculateHandle)getUsageRatioReplicas(ctx context.Context, CurrentReplicas int32, usageRatio float64, namespace string, selector labels.Selector)(replicas int32, timestamp time.Time, err error) {
	if CurrentReplicas != 0 {
		if math.Abs(1.0-usageRatio) <= c.ScaleTolerance {
			return CurrentReplicas, timestamp, nil
		}

		readyPodCount := int32(0)
		readyPodCount, err = c.getReadyPodsCount(ctx, namespace, selector)
		if err != nil {
			seelog.Errorf("getReadyPodsCount failed, err is %v", err.Error())
			return 0, time.Time{}, err
		}
		replicas = int32(math.Ceil(usageRatio * float64(readyPodCount)))
	}else {
		replicas = int32(math.Ceil(usageRatio))
	}

	return replicas, timestamp, nil
}

func (c *CalculateHandle)getReadyPodsCount(ctx context.Context, namespace string, podSelector labels.Selector)(readyPodCount int32, err error){
	objectList := &v1.PodList{}
	//objectList := &appsv1beta2.Deployment{}
	listOptions := client.ListOptions{
		Namespace: namespace,
		LabelSelector: podSelector,
	}
	if err := c.Client.List(ctx, objectList, &listOptions); err != nil {
		seelog.Errorf("List pods failed, err is %v", err.Error())
		return 0, err
	}
	if len(objectList.Items) == 0 {
		seelog.Errorf("no pods returned by selector:%v", podSelector.String())
		return 0, seelog.Errorf("no pods returned by selector:%v", podSelector.String())
	}

	readyPodCount = 0
	for _, pod := range(objectList.Items) {
		if pod.Status.Phase == v1.PodRunning && podutil.IsPodReady(&pod) {
			readyPodCount++
		}
	}

	return readyPodCount, nil
}

//按object对应pod列表的平均值评估
func (c *CalculateHandle)getReplicasForPerPodWithObjectSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	metricSelector,err := metav1.LabelSelectorAsSelector(metric.Object.Metric.Selector)
	if err != nil {
		seelog.Errorf("LabelSelectorAsSelector failed, err is %v", err)
		return 0, time.Time{}, err
	}

	objectRef := &metric.Object.DescribedObject
	metricName := metric.Object.Metric.Name
	utilization, timestamp, err := c.RestMetricsClientHandle.GetObjectMetric(metricName, namespace, objectRef, metricSelector)
	if err != nil {
		seelog.Errorf("GetObjectMetric failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	replicas = currentReplicas
	targetAverageUtilization := metric.Object.Target.AverageValue.MilliValue()
	usageRatio := float64(utilization)/(float64(targetAverageUtilization)*float64(currentReplicas))
	if  math.Abs(1.0-usageRatio)> c.ScaleTolerance {
		replicas = int32(math.Ceil(float64(utilization)/float64(targetAverageUtilization)))
	}

	//utilization := int32(math.Ceil(float64(utilization)/float64(currentReplicas)))

	return replicas,timestamp, nil
}

func (c *CalculateHandle)CalculateReplicasWithExternalMetricSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, podSelector labels.Selector, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	if metric.Object.Target.Type == autoscalingv2beta2.ValueMetricType {
		replicas, timestamp, err = c.getReplicasWithExternalMetricSourceType(ctx, namespace, metric, podSelector,currentReplicas)
		if err != nil {
			seelog.Errorf("getReplicasWithExternalMetricSourceType failed, err is %v", err.Error())
			return 0, time.Time{}, err
		}
	}else if metric.Object.Target.Type == autoscalingv2beta2.AverageValueMetricType {
		replicas, timestamp, err = c.getReplicasForPerPodWithExternalMetricSourceType(ctx, namespace, metric, podSelector,currentReplicas)
		if err != nil {
			seelog.Errorf("getReplicasForPerPodWithExternalMetricSourceType failed, err is %v", err.Error())
			return 0, time.Time{}, err
		}
	}else if metric.Object.Target.Type == autoscalingv2beta2.UtilizationMetricType {
		replicas = currentReplicas
	}else {
		seelog.Errorf("metric.Object.Target.Type:%v is not supported", metric.Object.Target.Type)
		return 0, time.Time{}, seelog.Errorf("metric.Object.Target.Type:%v is not supported", metric.Object.Target.Type)
	}

	return replicas, timestamp, nil
}

func (c *CalculateHandle) getReplicasWithExternalMetricSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, podSelector labels.Selector, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	metricSelector,err := metav1.LabelSelectorAsSelector(metric.External.Metric.Selector)
	if err != nil {
		seelog.Errorf("LabelSelectorAsSelector failed, err is %v", err)
		return 0, time.Time{}, err
	}
	metricName := metric.External.Metric.Name
	metrics, timestamp, err := c.RestMetricsClientHandle.GetExternalMetric(metricName, namespace, metricSelector)
	if err != nil {
		seelog.Errorf("GetExternalMetric failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	utilization := int64(0)
	for _,value := range(metrics) {
		utilization += value
	}

	replicas = currentReplicas
	targetUtilization := metric.External.Target.Value.MilliValue()
	usageRatio := float64(utilization)/float64(targetUtilization)
	replicas, timestamp, err = c.getUsageRatioReplicas(ctx, currentReplicas, usageRatio, namespace, podSelector)
	if err != nil {
		seelog.Errorf("getUsageRatioReplicas failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	return replicas, timestamp, nil
}

func (c *CalculateHandle)getReplicasForPerPodWithExternalMetricSourceType(ctx context.Context, namespace string, metric autoscalingv2beta2.MetricSpec, podSelector labels.Selector, currentReplicas int32)(replicas int32, timestamp time.Time, err error){
	metricSelector,err := metav1.LabelSelectorAsSelector(metric.External.Metric.Selector)
	if err != nil {
		seelog.Errorf("LabelSelectorAsSelector failed, err is %v", err)
		return 0, time.Time{}, err
	}

	metricName := metric.External.Metric.Name
	metrics, timestamp, err := c.RestMetricsClientHandle.GetExternalMetric(metricName, namespace, metricSelector)
	if err != nil {
		seelog.Errorf("GetExternalMetric failed, err is %v", err.Error())
		return 0, time.Time{}, err
	}

	utilization := int64(0)
	for _,value := range(metrics) {
		utilization += value
	}

	replicas = currentReplicas
	targetAverageUtilization := metric.External.Target.AverageValue.MilliValue()
	usageRatio := float64(utilization)/(float64(targetAverageUtilization) * float64(currentReplicas))
	if math.Abs(1.0 - usageRatio) > 0 {
		replicas = int32(math.Ceil(float64(utilization)/float64(targetAverageUtilization)))
	}
	//计算调度前的使用情况
	utilization = int64(math.Ceil(float64(utilization)/float64(currentReplicas)))
	
	return replicas, timestamp, nil
}

//获取指定container name及resource的总累计值，如：depoloyment A 里面container name为"test"的容器，该容器配置resource的request下的cpu累计值
func calculatePodRequests(podList *v1.PodList, container string, resource v1.ResourceName)(map[string]int64, error) {
	requests := make(map[string]int64, len(podList.Items))
	for _,pod := range podList.Items {
		podSum := int64(0)
		for _,c := range(pod.Spec.Containers){
			if container == "" || container == c.Name{
				if containerRequest,ok := c.Resources.Requests[resource]; ok {
					podSum += containerRequest.MilliValue()
				}
			}else {
				seelog.Errorf("missing request for resource:%v", resource)
				return nil, seelog.Errorf("missing request for resource:%v", resource)
			}
		}

		requests[pod.Name] = podSum
	}

	return requests,nil
}

func groupPods(podList *v1.PodList, metrics metricsclient.PodMetricsInfo, resource v1.ResourceName, cpuInitializationPeriod, delayOfInitialReadinessStatus time.Duration) (readyPodCount int, unreadyPods, missingPods, ignoredPods sets.String){
	//未就绪pod列表
	unreadyPods = sets.NewString()
	//无metric数据的pod列表
	missingPods = sets.NewString()
	//挂掉或异常状态的pod列表
	ignoredPods = sets.NewString()

	for _,pod := range podList.Items{
		if pod.DeletionTimestamp != nil || pod.Status.Phase != v1.PodFailed{
			ignoredPods.Insert(pod.Name)
			continue
		}

		if pod.Status.Phase == v1.PodPending{
			unreadyPods.Insert(pod.Name)
			continue
		}

		metric, found := metrics[pod.Name]
		if !found {
			missingPods.Insert(pod.Name)
			continue
		}

		if resource == v1.ResourceCPU {
			var unready bool
			_, condition := podutil.GetPodCondition(&pod.Status, v1.PodReady)
			if condition == nil || pod.Status.StartTime == nil {
				unready = true
			}else {
				//如果pod已经正常启动
				if pod.Status.StartTime.Add(cpuInitializationPeriod).After(time.Now()) {
					// 如果pod当前状态未ready或者本统计周期内指标空缺，则仍未unready
					unready = condition.Status == v1.ConditionFalse || metric.Timestamp.Before(condition.LastTransitionTime.Time.Add(metric.Window))
				}else {
					// pod未正常启动，如果pod当前状态未ready且从启动一直未ready，则仍未unready
					unready = condition.Status == v1.ConditionFalse && pod.Status.StartTime.Add(delayOfInitialReadinessStatus).After(condition.LastTransitionTime.Time)
				}
			}

			if unready {
				unreadyPods.Insert(pod.Name)
				continue
			}
		}
		readyPodCount++
	}

	return readyPodCount, unreadyPods, missingPods, ignoredPods
}


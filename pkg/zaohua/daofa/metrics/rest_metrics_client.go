package metrics

import (
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"


	"time"
)

type RestMetricsClient struct{
	*resourceMetricsClient
	*customMetricsClient
	*externalMetricsClient
}

type resourceMetricsClient struct{
	client resourceclient.NodeMetricsesGetter
}

type customMetricsClient struct{
	client customclient.AvailableAPIsGetter
}

type externalMetricsClient struct{
	client externalclient.NamespacedMetricsGetter
}

func GetResouceMetric(resouce v1.ResourceName, namespace string, selector labels.Selector, container string)(PodMetricsInfo, time.Time, error){

	return PodMetricsInfo{}, time.Now(), nil
}

func getPodMetrics(rawMetrics []metricsapi.PodMetrics, resource v1.ResourceName)PodMetricsInfo {

	return PodMetricsInfo{}
}

func GetRawMetric(metricName string, namespace string, selector labels.Selector, metricSelector labels.Selector)(PodMetricsInfo, time.Time, error){

	return PodMetricsInfo{}, time.Now(), nil
}

func GetObjectMetric(metricName string, namespace string, objectRef *autoscalingv2beta2.CrossVersionObjectReference, selector labels.Selector)(int64, time.Time, error){

	return 0, time.Now(), nil
}

func GetExternalMetric(metricName string, namespace string, selector labels.Selector)([]int64, time.Time, error){

	return []int64{0,1,2}, time.Now(), nil
}
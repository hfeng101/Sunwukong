package metrics

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"time"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

type PodMetric struct {
	Timestamp time.Time
	Window	time.Duration
	Value	int64
}

type PodMetricsInfo map[string]PodMetric

type MetricsInterface interface {
	GetResourceMetric(resource v1.ResourceName, namespace string, selector labels.Selector, container string)(PodMetricsInfo, time.Time, error)
	GetRawMetric(metricName string, namespace string, selector labels.Selector, metricSelector labels.Selector)(PodMetricsInfo, time.Time, error)
	GetObjectMetric(metricName string, namespace string, objectRef *autoscalingv2beta2.CrossVersionObjectReference, metricSelector labels.Selector)(int32, time.Time, error)
	GetExternalMetric(metricName string, namespace string, selector labels.Selector)([]int32, time.Time, error)
}
package metrics

import (
	"context"
	"github.com/cihub/seelog"
	"github.com/hfeng101/Sunwukong/util/consts"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/apimachinery/pkg/util/cache"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	customclient "k8s.io/metrics/pkg/client/custom_metrics"
	externalclient "k8s.io/metrics/pkg/client/external_metrics"

	customapi "k8s.io/metrics/pkg/apis/custom_metrics/v1beta2"
	"time"
)

type resourceMetricsClient struct{
	client resourceclient.PodMetricsesGetter
}

type customMetricsClient struct{
	client customclient.CustomMetricsClient
}

type externalMetricsClient struct{
	client externalclient.ExternalMetricsClient
}

type RestMetricsClient struct{
	*resourceMetricsClient
	*customMetricsClient
	*externalMetricsClient
}

func NewRestMetricsClient(resourceClient resourceclient.PodMetricsesGetter, customClient customclient.CustomMetricsClient, externalClient externalclient.ExternalMetricsClient)*RestMetricsClient {
	return &RestMetricsClient{
		&resourceMetricsClient{resourceClient},
		&customMetricsClient{customClient},
		&externalMetricsClient{externalClient},
	}
}



func (c *resourceMetricsClient) GetResouceMetric(resource v1.ResourceName, namespace string, selector labels.Selector, containerName string)(PodMetricsInfo, time.Time, error){
	metrics, err := c.client.PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		seelog.Errorf("unable to fetch metrics from resource metrics API:%v", err.Error())
		return nil, time.Time{}, seelog.Errorf("unable to fetch metrics from resource metrics API:%v", err.Error())
	}

	if len(metrics.Items) == 0 {
		seelog.Errorf("no metrics returned from resource metrics API")
		return nil, time.Time{},seelog.Errorf("no metrics returned from resource metrics API")
	}

	var res PodMetricsInfo
	if containerName != "" {
		res, err = getContainerMetrics(metrics.Items, resource, containerName)
		if err != nil {
			seelog.Errorf("failed to get container metrics , err is %v", err.Error())
			return nil, time.Time{}, seelog.Errorf("failed to get container metrics , err is %v", err.Error())
		}
	}else {
		res = getPodMetrics(metrics.Items, resource)
	}

	timestamp := metrics.Items[0].Timestamp.Time

	return res, timestamp, nil
}

func getContainerMetrics(rawMetrics []metricsapi.PodMetrics, resource v1.ResourceName, containerName string)(PodMetricsInfo, error){
	res := make(PodMetricsInfo, len(rawMetrics))
	for _, m := range rawMetrics {
		containerFound := false
		for _, c := range m.Containers{
			if c.Name == containerName{
				containerFound = true
				if val, resFound := c.Usage[resource]; resFound {
					res[m.Name] = PodMetric{
						Timestamp: m.Timestamp.Time,
						Window: m.Window.Duration,
						Value: val.MilliValue(),
					}
				}

				break
			}
		}

		if  !containerFound{
			return nil, seelog.Errorf("container:%v not present in metrics for pod:%v/%v", containerName, m.Namespace, m.Name)
		}
	}

	return res, nil
}

func getPodMetrics(rawMetrics []metricsapi.PodMetrics, resource v1.ResourceName)PodMetricsInfo {
	res := make(PodMetricsInfo, len(rawMetrics))

	for _, m := range(rawMetrics) {
		podSum := int64(0)
		missing := len(m.Containers) == 0
		for _, c := range(m.Containers) {
			resValue, found := c.Usage[resource]
			if !found {
				missing = true
				seelog.Infof("missing resource metric %v for %v/%v", resource, m.Namespace, m.Name)
				break
			}
			podSum += resValue.MilliValue()
		}

		if !missing {
			res[m.Name] = PodMetric{
				Timestamp: m.Timestamp.Time,
				Window: m.Window.Duration,
				Value: podSum,
			}
		}
	}

	return res
}

func (c *customMetricsClient) GetRawMetric(metricName string, namespace string, selector labels.Selector, metricSelector labels.Selector)(PodMetricsInfo, time.Time, error){
	metrics,err := c.client.NamespacedMetrics(namespace).GetForObjects(schema.GroupKind{Kind: "Pod"}, selector, metricName, metricSelector)
	if err != nil {
		seelog.Errorf("unable to fetch metrics from custom metrics API: %v", err.Error())
		return nil, time.Time{}, seelog.Errorf("unable to fetch metrics from custom metrics API: %v", err.Error())
	}

	if len(metrics.Items) ==0 {
		seelog.Errorf("no metrics returned from custom metrics API")
		return nil, time.Time{}, seelog.Errorf("no metrics returned from custom metrics API")
	}

	res := make(PodMetricsInfo, len(metrics.Items))
	for _, m := range(metrics.Items) {
		window := consts.MetricServerDefaultMetricWindow
		if m.WindowSeconds != nil {
			window = time.Duration(*m.WindowSeconds) * time.Second
		}
		res[m.DescribedObject.Name] = PodMetric{
			Timestamp: m.Timestamp.Time,
			Window: window,
			Value: int64(m.Value.MilliValue()),
		}

		m.Value.MilliValue()
	}

	timestamp := metrics.Items[0].Timestamp.Time

	return res, timestamp, nil
}

func (c *customMetricsClient)GetObjectMetric(metricName string, namespace string, objectRef *autoscalingv2beta2.CrossVersionObjectReference, metricSelector labels.Selector)(int64, time.Time, error){
	gvk := schema.FromAPIVersionAndKind(objectRef.APIVersion, objectRef.Kind)
	var metricValue *customapi.MetricValue
	var err error
	if gvk.Kind == "Namespace" && gvk.Group == ""{
		metricValue, err = c.client.RootScopedMetrics().GetForObject(gvk.GroupKind(), namespace, metricName, metricSelector)
	}else {
		metricValue, err = c.client.NamespacedMetrics(namespace).GetForObject(gvk.GroupKind(), objectRef.Name, metricName, metricSelector)
	}

	if err != nil {
		seelog.Errorf("unable to fetch metrics from custom metrics API:%v", err.Error())
		return 0, time.Time{}, seelog.Errorf("unable to fetch metrics from custom metrics API:%v", err.Error())
	}

	return metricValue.Value.MilliValue(), metricValue.Timestamp.Time, nil
}

func (c *externalMetricsClient)GetExternalMetric(metricName string, namespace string, selector labels.Selector)([]int64, time.Time, error){
	metrics, err := c.client.NamespacedMetrics(namespace).List(metricName, selector)
	if err != nil {
		seelog.Errorf("unable to fetch metrics from external metric API:%v", err.Error())
		return []int64{}, time.Time{}, seelog.Errorf("unable to fetch metrics from external metric API:%v", err.Error())
	}

	if len(metrics.Items) == 0 {
		seelog.Errorf("no metrics returned from external metrics API")
		return []int64{}, time.Time{}, seelog.Errorf("no metrics returned from external metrics API")
	}

	res := make([]int64, 0)
	for _, m := range metrics.Items {
		res = append(res, m.Value.MilliValue())
	}
	timestamp := metrics.Items[0].Timestamp.Time

	return res, timestamp, nil
}
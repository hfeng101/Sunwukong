package consts

import "time"

const (
	ObjectMetricSourceType = "Object"
	PodsMetricSourceType = "Pods"
	ResourceMetricSourceType = "Resource"
	ContainerResourceMetricSourceType = "ContainerResource"
	ExternalMetricSourceType = "External"


	MetricServerDefaultMetricWindow = time.Minute
)

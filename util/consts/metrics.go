package consts

import "time"

const (
	ObjectMetricSourceType = "Object"						//k8s内置资源的指标
	PodsMetricSourceType = "Pods"							//pod的独立指标
	ResourceMetricSourceType = "Resource"					//pod下所有container的resource综合指标
	ContainerResourceMetricSourceType = "ContainerResource"	//pod下指定container的resource指标，如：cpu、mem
	ExternalMetricSourceType = "External"					//k8s外部自定义指标


	MetricServerDefaultMetricWindow = time.Minute
)

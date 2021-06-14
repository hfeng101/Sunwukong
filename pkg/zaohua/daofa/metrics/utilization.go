package metrics

import "github.com/cihub/seelog"

// 获取指定指标集的平均值，及其相对目标值的比率，如cpu使用率，及其与实际目标值的比率，示例：cpu总值/request总值 * 100%=cpu平均利用率，再对比目标值比率
func GetResourceUtilizationRatio(metrics PodMetricsInfo, requests map[string]int64, targetUtilization int64)(utilizationRatio float64, currentUtilization int32, rawAverageValue int64, err error) {
	metricsTotal := int64(0)
	requestsTotal := int64(0)
	numEntries := 0

	for podName, metric := range(metrics) {
		request, hasRequest := requests[podName]
		if !hasRequest {
			continue
		}

		metricsTotal += metric.Value
		requestsTotal += request
		numEntries++
	}

	if requestsTotal == 0 {
		seelog.Errorf("no metrics for request list")
		return 0, 0, 0, seelog.Errorf("no metrics for request list")
	}
	currentUtilization = int32((metricsTotal * 100)/requestsTotal)
	utilizationRatio = float64(currentUtilization)/float64(targetUtilization)
	rawAverageValue = metricsTotal/int64(numEntries)

	return utilizationRatio, currentUtilization, rawAverageValue, nil
}

// 获取指标的平均值，及其相对目标值的比率，比率大于1.0则表示超标，需要做弹性伸缩
func GetMetricUtilizationRatio(metrics PodMetricsInfo, targetUtilization int64)(utilizationRatio float64, currentUtilization int64) {
	metricsTotal := int64(0)

	for _,metric := range metrics {
		metricsTotal += metric.Value
	}
	
	currentUtilization = metricsTotal / int64(len(metrics))

	return float64(currentUtilization)/float64(targetUtilization), currentUtilization
}

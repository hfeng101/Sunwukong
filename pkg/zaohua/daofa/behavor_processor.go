package daofa

import (
	"context"
	"github.com/cihub/seelog"
	"github.com/hfeng101/Sunwukong/util/consts"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"math"
	"time"
)

type ReviseInfo struct {
	ScaleUpBehavor *autoscalingv2beta2.HPAScalingRules
	ScaleDownBehavor *autoscalingv2beta2.HPAScalingRules
	//MinReplicas	int32
	//MaxReplicas	int32
	CurrentReplicas	int32
	DesiredReplicas	int32
}

// 基于Behavor做弹性伸缩修订调整
func ReviseWithBehavor(ctx context.Context, desiredReplicas int32, currentReplicas int32, behavor *autoscalingv2beta2.HorizontalPodAutoscalerBehavior, recommendationRecords *[]TimestampedRecommendationRecord, scaleEvents *[]TimestampedScaleEvent)(scaledReplicas int32, err error){
	// 确保scaleDown的稳定窗口要初始化, 默认300s
	if behavor != nil && behavor.ScaleDown != nil && behavor.ScaleDown.StabilizationWindowSeconds == nil {
		*behavor.ScaleDown.StabilizationWindowSeconds = consts.StabilizationWindowSeconds
	}

	reviseInfo := ReviseInfo{
		behavor.ScaleUp,
		behavor.ScaleDown,
		currentReplicas,
		desiredReplicas,
	}

	//稳定窗口内做优选
	//stabilizationedRecommendation, reason, err := stabilizationRecommendationWithBehavors(reviseInfo, recommendationRecords)
	stabilizationedRecommendation, _, err := stabilizationRecommendationWithBehavors(reviseInfo, recommendationRecords)
	if err != nil {
		seelog.Errorf("stabilizationRecommendationBehavors failed, err is %v", err.Error())
		return 0, err
	}

	reviseInfo.DesiredReplicas = stabilizationedRecommendation
	//if stabilizationedRecommendation !=

	//scaledReplicas, reason, message := convertDesiredReplicasWithBehaviorRate(reviseInfo, scaleEvents)
	scaledReplicas, _, err = convertDesiredReplicasWithBehaviorRate(reviseInfo, scaleEvents)
	if err != nil {
		seelog.Errorf("convertDesiredReplicasWithBehaviorRate failed, err is %v", err.Error())
		return 0, err
	}

	return scaledReplicas,nil
}


//func maybeInitScaleDownStabilizationWindow(behavor *autoscalingv2beta2.HorizontalPodAutoscalerBehavior) {
//	if behavor != nil && behavor.ScaleDown != nil && behavor.ScaleDown.StabilizationWindowSeconds == nil {
//		behavor.ScaleDown.StabilizationWindowSeconds = consts.StabilizationWindowSeconds
//	}
//}

// 根据扩容/缩容策略，在稳定窗口内做优选，决策出最终结果
func stabilizationRecommendationWithBehavors(reviseInfo ReviseInfo, recommendationRecords *[]TimestampedRecommendationRecord)(recommendation int32, reason string, err error) {
	recommendation = reviseInfo.DesiredReplicas
	foundOldSample := false
	oldSampleIndex := int32(0)
	scaleDelaySeconds := int32(0)
	var recommendationAction func(int32, int32)(int32)

	if reviseInfo.DesiredReplicas >= reviseInfo.CurrentReplicas {
		scaleDelaySeconds = *reviseInfo.ScaleUpBehavor.StabilizationWindowSeconds
		recommendationAction = min
		reason = "ScaleUpStabilized"
		//message := "recent recommendations were lower than current one, applying the lowest recent recommendation"
	}else {
		scaleDelaySeconds = *reviseInfo.ScaleDownBehavor.StabilizationWindowSeconds
		recommendationAction = max
		reason = "ScaleUpStabilized"
		//message := "recent recommendations were lower than current one, applying the lowest recent recommendation"
	}

	maxDelaySeconds := max(int32(*reviseInfo.ScaleUpBehavor.StabilizationWindowSeconds), int32(*reviseInfo.ScaleDownBehavor.StabilizationWindowSeconds))
	cutoff := time.Now().Add(-time.Second * time.Duration(scaleDelaySeconds))
	obsoluteCutoff := time.Now().Add(-time.Second * time.Duration(maxDelaySeconds))

	for i,rec := range(*recommendationRecords) {
		//在稳定时间窗口之内，则当前建议有效
		if rec.Timestamp.After(cutoff) {
			recommendation = recommendationAction(rec.DesiredReplicas, recommendation)
		}

		//TODO: 有什么用呢？
		//在过时截止时间之前的最后一次记录更新了
		if rec.Timestamp.Before(obsoluteCutoff) {
			foundOldSample = true
			oldSampleIndex = int32(i)
		}
		//if rec.DesiredReplicas >= reviseInfo.DesiredReplicas
	}

	if foundOldSample{
		(*recommendationRecords)[oldSampleIndex] = TimestampedRecommendationRecord{recommendation, time.Now()}
	}else {
		*recommendationRecords = append(*recommendationRecords, TimestampedRecommendationRecord{recommendation, time.Now()})
	}

	return recommendation, reason, nil
}

//按Behavor策略（如：按pod数扩缩或按比例扩缩，扩缩上限和快慢节奏控制）决策出最终结果
func convertDesiredReplicasWithBehaviorRate(reviseInfo ReviseInfo, scaleEvents *[]TimestampedScaleEvent)(replicas int32, reason string, err error) {
	if reviseInfo.DesiredReplicas > reviseInfo.DesiredReplicas{
		scaleUpLimit := calculateScaleUpLimitWithScalingRules(reviseInfo.CurrentReplicas, scaleEvents, reviseInfo.ScaleUpBehavor)
		//if scaleUpLimit < reviseInfo.CurrentReplicas {
		//	scaleUpLimit = reviseInfo.CurrentReplicas
		//}

		if reviseInfo.DesiredReplicas > scaleUpLimit {
			replicas = scaleUpLimit
			reason = "scaleUpLimit"
		}else {
			replicas = reviseInfo.DesiredReplicas
			reason = "DesiredWithinRange"
		}
	}else if reviseInfo.DesiredReplicas < reviseInfo.CurrentReplicas{
		scaleDownLimit := calculateScaleDownLimitWithScalingRules(reviseInfo.CurrentReplicas, scaleEvents, reviseInfo.ScaleUpBehavor)

		//if scaleDownLimit > reviseInfo.CurrentReplicas{
		//	scaleDownLimit = reviseInfo.CurrentReplicas
		//}

		if reviseInfo.DesiredReplicas < scaleDownLimit {
			replicas = scaleDownLimit
			reason = "scaleDownLimit"
		}else {
			replicas = reviseInfo.DesiredReplicas
			reason = "DesiredWithinRange"
		}
	}

	return replicas, reason, nil
}

func calculateScaleUpLimitWithScalingRules(currentReplicas int32, scaleEvents *[]TimestampedScaleEvent, scaleUpBehavor *autoscalingv2beta2.HPAScalingRules)(scaleUpLimit int32){
	proposedReplicas := int32(0)
	var selectPolicyFunc func(int32, int32)int32

	if *scaleUpBehavor.SelectPolicy == autoscalingv2beta2.DisabledPolicySelect{
		scaleUpLimit = currentReplicas
		return
	}else if *scaleUpBehavor.SelectPolicy == autoscalingv2beta2.MinPolicySelect {
		scaleUpLimit = math.MinInt32
		selectPolicyFunc = min
	}else {
		scaleUpLimit = math.MaxInt32
	}

	for _, policy :=  range(scaleUpBehavor.Policies) {
		replicasAddedInCurrentStabiliztionWindow := getReplicasChangePerScalePolicyPeriod(policy.PeriodSeconds, scaleEvents)
		startReplicasInCurrentStabiliztionWindow := currentReplicas - replicasAddedInCurrentStabiliztionWindow
		if policy.Type == autoscalingv2beta2.PodsScalingPolicy {
			proposedReplicas = startReplicasInCurrentStabiliztionWindow + int32(policy.Value)
		}else if policy.Type == autoscalingv2beta2.PercentScalingPolicy {
			proposedReplicas = int32(math.Ceil(float64(startReplicasInCurrentStabiliztionWindow) * (1 + float64(policy.Value)/100)))
		}
		scaleUpLimit = selectPolicyFunc(scaleUpLimit, proposedReplicas)
	}

	// 若要扩容的上限低于当前实例数，以当前为准
	if scaleUpLimit < currentReplicas{
		scaleUpLimit = currentReplicas
	}

	return scaleUpLimit
}

func calculateScaleDownLimitWithScalingRules(currentReplicas int32, scaleEvents *[]TimestampedScaleEvent, scaleDownBehavor *autoscalingv2beta2.HPAScalingRules)(scaleDownLimit int32){
	proposedReplicas := int32(0)
	var selectPolicyFunc func(int32, int32)int32

	if *scaleDownBehavor.SelectPolicy == autoscalingv2beta2.DisabledPolicySelect{
		scaleDownLimit = currentReplicas
		return
	}else if *scaleDownBehavor.SelectPolicy == autoscalingv2beta2.MinPolicySelect {
		scaleDownLimit = math.MaxInt32	//初始为最大值
		selectPolicyFunc = min
	}else {
		scaleDownLimit = math.MinInt32	//初始为最小值
	}

	for _, policy :=  range(scaleDownBehavor.Policies) {
		replicasAddedInCurrentStabiliztionWindow := getReplicasChangePerScalePolicyPeriod(policy.PeriodSeconds, scaleEvents)
		startReplicasInCurrentStabiliztionWindow := currentReplicas - replicasAddedInCurrentStabiliztionWindow
		if policy.Type == autoscalingv2beta2.PodsScalingPolicy {
			proposedReplicas = startReplicasInCurrentStabiliztionWindow + int32(policy.Value)
		}else if policy.Type == autoscalingv2beta2.PercentScalingPolicy {
			proposedReplicas = int32(math.Ceil(float64(startReplicasInCurrentStabiliztionWindow) * (1 + float64(policy.Value)/100)))
		}
		//根据SelectPolicy决策选优
		scaleDownLimit = selectPolicyFunc(scaleDownLimit, proposedReplicas)
	}

	// 若要缩容的下限超过当前实例数，以当前为准
	if scaleDownLimit > currentReplicas{
		scaleDownLimit = currentReplicas
	}

	return scaleDownLimit
}

// 获取每个伸缩周期内的变更量总和
func getReplicasChangePerScalePolicyPeriod(policyPeriodSeconds int32, scaleEvents *[]TimestampedScaleEvent)(replicas int32){
	period := time.Second * time.Duration(policyPeriodSeconds)
	cutoff := time.Now().Add(-period)

	for _,scaleEvent := range(*scaleEvents) {
		if scaleEvent.TimeStamp.After(cutoff){
			replicas += scaleEvent.ReplicaChange
		}
	}
	return replicas
}

func max(a, b int32) int32 {
	if a <= b {
		return a
	}
	return b
}

func min(a, b int32) int32 {
	if a <= b {
		return a
	}
	return b
}
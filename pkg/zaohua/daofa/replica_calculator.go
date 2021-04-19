package daofa

import (
	"context"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/runtime/schema"
)


func CalculateReplicasWithPodsMetricSourceType(ctx context.Context, targetGVK schema.GroupVersionKind, metric autoscalingv2beta2.MetricSpec, currentReplicas int64)(int64, error){
	return 0,nil
}


func CalculateReplicasWithResourceMetricSourceType(ctx context.Context, targetGVK schema.GroupVersionKind, metric autoscalingv2beta2.MetricSpec, currentReplicas int64)(int64, error){
	return 0,nil
}

func CalculateReplicasWithContainerResourceMetricSourcetype(ctx context.Context, targetGVK schema.GroupVersionKind, metric autoscalingv2beta2.MetricSpec, currentReplicas int64)(int64, error){
	return 0,nil
}

func CalculateReplicasWithObjectSourceType(ctx context.Context, targetGVK schema.GroupVersionKind, metric autoscalingv2beta2.MetricSpec, currentReplicas int64)(int64, error){
	return 0,nil
}

func CalculateExternalMetricSourceType(ctx context.Context, targetGVK schema.GroupVersionKind, metric autoscalingv2beta2.MetricSpec, currentReplicas int64)(int64, error){
	return 0,nil
}
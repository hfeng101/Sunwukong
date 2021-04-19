package daofa

import (
	"context"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

func ReviseWithBehavor(ctx context.Context, replicas int64, bahavor autoscalingv2beta2.HorizontalPodAutoscalerBehavior)(error){

	

	return nil
}
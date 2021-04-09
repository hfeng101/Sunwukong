package daofa

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var RestartCh = make(chan struct{})

type ZaohuaHandle struct {
	client.Client
	Object *ScaleObject
}

type ScaleObject struct {
	//兼容k8s原生hpa，更新会使得zaohua在下一轮开始前重新加载
	ScaleTargetRef autoscaling.CrossVersionObjectReference	`json:"scaleTargetRef,omitempty"`
	Metrics autoscaling.MetricSpec							`json:"metrics,omitempty"`
	Behavor autoscaling.HorizontalPodAutoscalerBehavior		`json:"behavor"`
	MinReplicas *int										`json:"minReplicas"`
	MaxReplicas int											`json:"maxReplicas"`

	//要关注的service，关注对应的endpoints变化或服务质量等
	ServiceName []string	`json:"serviceName"`
}

func (z *ZaohuaHandle) StartShifa(ctx context.Context, ) error{
	//defer ctx.Done()

	// 不断查看监控数据，并根据条件触发造化
	for {
		scaler(ctx, z.Object)

		select{
		case <-ctx.Done():
			// 退出
			seelog.Infof("break out, return")
			return nil
		case <- RestartCh:
			// 重新加载对象或行为
			seelog.Infof("Reload houmao config for next zaohua operation")
		default:
			// 下一轮走起
			seelog.Infof("next round of zaohua")
		}

	}

	return nil
}

func RestartShifa(houmao *sunwukongv1.Houmao) error{
	defer func(){RestartCh <- struct{}{}}()

	reloadZaohuaHandle(houmao)
	return nil
}

func reloadZaohuaHandle(houmao *sunwukongv1.Houmao) error {

	return nil
}

func scaler(ctx context.Context, scaleObject *ScaleObject) error {
	//如果scaleObject没有任何数据，怎么处理，空跑？等待下一轮更新重启
	//if *scaleObject == ScaleObject{}{
	//		<- RestartCh
	//}

	return nil
}
package daofa

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"github.com/hfeng101/Sunwukong/util/consts"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var (
	RestartCh = make(chan struct{})
	HoumaoObject = &sunwukongv1.Houmao{}
)

type ZaohuaHandle struct {
	client.Client
	Object *sunwukongv1.Houmao
}

//type ScaleObject struct {
//	//兼容k8s原生hpa，更新会使得zaohua在下一轮开始前重新加载
//	ScaleTargetRef autoscaling.CrossVersionObjectReference	`json:"scaleTargetRef,omitempty"`
//	Metrics autoscaling.MetricSpec							`json:"metrics,omitempty"`
//	Behavor autoscaling.HorizontalPodAutoscalerBehavior		`json:"behavor"`
//	MinReplicas *int										`json:"minReplicas"`
//	MaxReplicas int											`json:"maxReplicas"`
//
//	//要关注的service，关注对应的endpoints变化或服务质量等
//	ServiceName []string	`json:"serviceName"`
//}

func (z *ZaohuaHandle) StartShifa(ctx context.Context, mode string) error{
	//defer ctx.Done()

	// 不断查看监控数据，并根据条件触发造化
	for {
		// 支持主动、被动两种模式触发弹性伸缩
		switch mode {
		case consts.ZaohuaModeZhudong:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		case consts.ZaohuaModeBeidong:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.scaleForEvent(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		default:
			// 主动拉取指标，检测并触发弹性伸缩
			if err := z.detectAndScale(ctx, z.Object);err != nil {
				seelog.Errorf("scaler failed, time is %v, err is %v", time.Now().UTC(), err.Error())
			}
		}


		select{
		case <-ctx.Done():
			// 退出
			seelog.Infof("break out, return")
			return nil
		case <- RestartCh:
			// 重新加载对象或行为
			seelog.Infof("Reload houmao config for next zaohua operation")
			z.Object = HoumaoObject.DeepCopy()
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
	HoumaoObject.Spec = houmao.Spec
	HoumaoObject.Status = houmao.Status
	return nil
}

// 主动检测指标触发伸缩
func (z *ZaohuaHandle)detectAndScale(ctx context.Context, scaleObject *sunwukongv1.Houmao) error {
	//如果scaleObject没有任何数据，怎么处理，空跑？等待下一轮更新重启
	if reflect.DeepEqual(*scaleObject, sunwukongv1.Houmao{}){
		//等待下一次变更触发
		<- RestartCh
		return nil
	}

	// 获取target
	target := scaleObject.Spec.ScaleTargetRef
	// 获取metrics
	metrics := scaleObject.Spec.Metrics
	// 计算
	replicas,err := calculate(ctx, target, metrics, scaleObject.Status.CurrentZaohuaResult.CurrentReplicas,  scaleObject.Spec.MaxReplicas, *(scaleObject.Spec.MinReplicas))
	if err != nil {
		seelog.Errorf("calculate replicas failed, err is %v", err.Error())
		return nil
	}

	// 根据柔性策略，实施伸缩
	if err := execScale(ctx, replicas, target); err != nil {

	}
	//同步service变化


	//更新状态，收尾


	return nil
}

// 被动接受push上来的事件，然后判断是否做伸缩
func (z *ZaohuaHandle)scaleForEvent(ctx context.Context, object *sunwukongv1.Houmao) error{

	return nil
}

// 根据指标计算伸缩结果
func calculate(ctx context.Context, target interface{}, metrics interface{}, currentReplicas int64, maxReplicas int64, minReplicas int64)(int64, error) {

	return currentReplicas, nil
}

func execScale(ctx context.Context, scaledReplicas int64, target interface{}) error{
	return nil
}

func syncService()error {

	return nil
}

func updateStatus(object *sunwukongv1.Houmao)error {

	return nil
}
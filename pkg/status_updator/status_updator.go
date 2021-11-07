package status_updator

import (
	"context"
	"encoding/json"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type StatusUpdateHandle struct{
	client.Client
	object *sunwukongv1.Houmao
}

var (
	Lock sync.Mutex
	Handle *StatusUpdateHandle
)

// 初始化
func NewStatusUpdateHandle(clientHandle client.Client, object *sunwukongv1.Houmao) (*StatusUpdateHandle){
	Handle = &StatusUpdateHandle{
		clientHandle,
		object,
	}

	return &StatusUpdateHandle{
		clientHandle,
		object,
	}
}

// 加锁竞获
func GetStatusUpdateHandle() *StatusUpdateHandle{
	Lock.Lock()
	defer Lock.Unlock()

	return Handle
}

// 更新状态
func (s *StatusUpdateHandle) UpdateStatus(ctx context.Context, status *sunwukongv1.HoumaoStatus) error{
	//s.object.Status = status
	if reflect.DeepEqual(s.object.Status, status){
		seelog.Infof("status is same,do not update status")
		return nil
	}

	////TODO: 如何只更新status，而非全局更新
	//s.object.Status = status
	//
	//if err := s.Client.Update(ctx, s.object);err != nil {
	//	seelog.Errorf("update status for %v failed, err is %v", status, err)
	//	return err
	//}

	//若用patch，就可以不用考虑锁问题，不同的role更新不同的status信息
	patch,err := json.Marshal(status)
	if err != nil {
		seelog.Errorf("status:%v doing json marshal failed", status)
		return err
	}
	patchInfo := client.RawPatch(types.StrategicMergePatchType, patch)

	if err := s.Client.Patch(ctx, s.object, patchInfo);err != nil {
		seelog.Errorf("Patch status:%v for object:%v failed", status, s.object)
		return err
	}

	return nil
}

// 仅更新仙气
func (s *StatusUpdateHandle) UpdateShifaPhase(ctx context.Context, shifaPhase string) error {
	if reflect.DeepEqual(s.object.Status.ShifaPhase, shifaPhase){
		seelog.Infof("status is same,do not update status")
		return nil
	}

	//若用patch，就可以不用考虑锁问题，不同的role更新不同的status信息
	patch,err := json.Marshal(shifaPhase)
	if err != nil {
		seelog.Errorf("status:%v doing json marshal failed", shifaPhase)
		return err
	}
	patchInfo := client.RawPatch(types.StrategicMergePatchType, patch)

	if err := s.Client.Patch(ctx, s.object, patchInfo);err != nil {
		seelog.Errorf("Patch status:%v for object:%v failed", shifaPhase, s.object)
		return err
	}

	return nil
}

// 仅更新仙气
func (s *StatusUpdateHandle) UpdateXianqiInfo(ctx context.Context, xianqiInfo *sunwukongv1.XianqiInfo) error {
	if reflect.DeepEqual(s.object.Status.XianqiInfo, xianqiInfo){
		seelog.Infof("status is same,do not update status")
		return nil
	}

	//若用patch，就可以不用考虑锁问题，不同的role更新不同的status信息
	patch,err := json.Marshal(xianqiInfo)
	if err != nil {
		seelog.Errorf("status:%v doing json marshal failed", xianqiInfo)
		return err
	}
	patchInfo := client.RawPatch(types.StrategicMergePatchType, patch)

	if err := s.Client.Patch(ctx, s.object, patchInfo);err != nil {
		seelog.Errorf("Patch status:%v for object:%v failed", xianqiInfo, s.object)
		return err
	}

	return nil
}

// 更新造化结果及记录
func (s *StatusUpdateHandle) UpdateZaohuaRecord(ctx context.Context, zaohuaRecord *sunwukongv1.ZaohuaRecord) error {
	if reflect.DeepEqual(s.object.Status.ZaohuaRecord, zaohuaRecord){
		seelog.Infof("status is same,do not update status")
		return nil
	}

	//若用patch，就可以不用考虑锁问题，不同的role更新不同的status信息
	patch,err := json.Marshal(zaohuaRecord)
	if err != nil {
		seelog.Errorf("status:%v doing json marshal failed", zaohuaRecord)
		return err
	}
	patchInfo := client.RawPatch(types.StrategicMergePatchType, patch)

	if err := s.Client.Patch(ctx, s.object, patchInfo);err != nil {
		seelog.Errorf("Patch status:%v for object:%v failed", zaohuaRecord, s.object)
		return err
	}

	return nil
}

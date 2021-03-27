package status_updator

import (
	"context"
	"github.com/cihub/seelog"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
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
func NewStatusUpdateHandle(clientHandle client.Client, object *sunwukongv1.Houmao){
	Handle = &StatusUpdateHandle{
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

func (s *StatusUpdateHandle) UpdateStatus(ctx context.Context, status sunwukongv1.HoumaoStatus) error{
	//s.object.Status = status
	if reflect.DeepEqual(s.object.Status, status){
		seelog.Infof("status is same,do not update status")
		return nil
	}

	//TODO: 如何只更新status，而非全局更新
	if err := s.Client.Update(ctx, s.object);err != nil {
		seelog.Errorf("update status for %v failed, err is %v", status, err)
		return err
	}

	return nil
}


func (s *StatusUpdateHandle) UpdateXianqiInfo(ctx context.Context, xianqiInfo sunwukongv1.XianqiInfo) error {

	return nil
}

func (s *StatusUpdateHandle) UpdatePhase(ctx context.Context, phase string) error {

	return nil
}
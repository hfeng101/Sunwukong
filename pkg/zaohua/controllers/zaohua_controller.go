package controllers

import (
	"context"
	sunwukongv1 "github.com/hfeng101/Sunwukong/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	//"k8s.io/apimachinery/"
	"k8s.io/apimachinery/pkg/runtime"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type ZaohuaController struct {
	client.Client

	Scheme *runtime.Scheme
}

func (z *ZaohuaController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error){

	return ctrl.Result{},nil
}

func (z *ZaohuaController) ZaohuaStartup() error {

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (z *ZaohuaController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sunwukongv1.Houmao{}).
		Complete(z)
}
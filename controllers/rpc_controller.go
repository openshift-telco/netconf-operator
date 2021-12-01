/*
Copyright 2021. Alexis de Talhouët

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/adetalhouet/go-netconf/netconf/message"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	"github.com/redhat-cop/operator-utils/pkg/util"

	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
)

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=rpcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=rpcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=rpcs/finalizers,verbs=update

// RPCReconciler reconciles a RPC object
type RPCReconciler struct {
	util.ReconcilerBase
}

// AddRPC Add creates a new MountPoint Controller and adds it to the Manager.
func AddRPC(mgr manager.Manager) error {
	return addRPC(mgr, newRPCReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(rpcControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling RPC")

	// Fetch the CRD instance
	instance := &netconfv1.RPC{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("RPC resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get RPC")
		return r.ManageError(ctx, instance, err)
	}

	// Managing CR validation
	if ok, err := r.isValid(instance); !ok {
		return r.ManageErrorWithRequeue(ctx, instance, err, 2*time.Second)
	}

	// Managing CR Initialization
	if ok := r.isInitialized(instance); !ok {
		err := r.GetClient().Update(context.Background(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(ctx, instance, err)
		}
		return reconcile.Result{}, nil
	}

	// Managing CR Finalization
	if util.IsBeingDeleted(instance) {
		//if !util.HasFinalizer(instance, finalizer) {
		//	return reconcile.Result{}, nil
		//}
		//err := r.manageCleanUpLogic(instance)
		//
		//if err != nil {
		//	log.Error(err, "unable to delete instance", "instance", instance)
		//	return r.ManageError(ctx, instance, err)
		//}
		//util.RemoveFinalizer(instance, finalizer)
		//err = r.GetClient().Update(context.Background(), instance)
		//if err != nil {
		//	log.Error(err, "unable to update instance", "instance", instance)
		//	return r.ManageError(ctx, instance, err)
		//}
		return reconcile.Result{}, nil
	}

	// Managing Logic
	err = r.manageOperatorLogic(instance, log)
	if err != nil {
		return r.ManageError(ctx, instance, err)
	}

	return r.ManageSuccess(ctx, instance)
}

func (r *RPCReconciler) isInitialized(obj metav1.Object) bool {
	_, ok := obj.(*netconfv1.RPC)
	if !ok {
		return false
	}
	//if util.HasFinalizer(mountPoint, finalizer) {
	//	return true
	//}
	//util.AddFinalizer(mountPoint, finalizer)
	return true

}

func (r *RPCReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.RPC)
	if !ok {
		return false, fmt.Errorf("%s not an RPC object", obj.GetName())
	}

	exists := CheckMountPointExists(
		r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint},
	)
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	return true, nil
}

//func (r *RPCReconciler) manageCleanUpLogic(RPC *netconfv1.RPC) error {
//
//	// TODO NOOP
//
//	return nil
//}

func (r *RPCReconciler) manageOperatorLogic(obj *netconfv1.RPC, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Send RPC %s.", obj.Spec.MountPoint, obj.Name))

	s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]
	reply, err := s.SyncRPC(message.NewRPC(obj.Spec.XML), obj.Spec.Timeout)

	if err != nil || reply.Errors != nil {
		log.Info(fmt.Sprintf("%s: Failed to send RPC %s.", obj.Spec.MountPoint, obj.Name))
		obj.Status = "failed"
		obj.RpcReply = reply.RawReply
		return err
	}

	log.Info(fmt.Sprintf("%s: Successfully executed RPC %s operation.", obj.Spec.MountPoint, obj.Name))
	obj.Status = "success"
	obj.RpcReply = reply.Data

	return nil
}

func newRPCReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &RPCReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(rpcControllerName),
			mgr.GetAPIReader(),
		),
	}
}

func addRPC(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(rpcControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.RPC{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

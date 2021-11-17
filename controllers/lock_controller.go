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
	"errors"
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

	"github.com/redhat-cop/operator-utils/pkg/util"

	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
)

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=locks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=locks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=locks/finalizers,verbs=update

// LockReconciler reconciles a Lock object
type LockReconciler struct {
	util.ReconcilerBase
}

// AddLock Add creates a new MountPoint Controller and adds it to the Manager.
func AddLock(mgr manager.Manager) error {
	return addLock(mgr, newLockReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LockReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(lockControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Lock")

	// Fetch the CRD instance
	instance := &netconfv1.Lock{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Lock resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Lock")
		return r.ManageError(ctx, instance, err)
	}

	// Managing CR validation
	if ok, err := r.isValid(instance); !ok {
		return r.ManageError(ctx, instance, err)
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
		if !util.HasFinalizer(instance, finalizer) {
			return reconcile.Result{}, nil
		}
		err := r.manageCleanUpLogic(instance)

		if err != nil {
			log.Error(err, "unable to delete instance", "instance", instance)
			return r.ManageError(ctx, instance, err)
		}
		util.RemoveFinalizer(instance, finalizer)
		err = r.GetClient().Update(context.Background(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(ctx, instance, err)
		}
		return reconcile.Result{}, nil
	}

	// Managing MountPoint Logic
	err = r.manageOperatorLogic(instance, log)
	if err != nil {
		return r.ManageError(ctx, instance, err)
	}

	return r.ManageSuccess(ctx, instance)
}

func (r *LockReconciler) isInitialized(obj metav1.Object) bool {
	mountPoint, ok := obj.(*netconfv1.Lock)
	if !ok {
		return false
	}
	if util.HasFinalizer(mountPoint, finalizer) {
		return true
	}
	util.AddFinalizer(mountPoint, finalizer)
	return false

}

func (r *LockReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.Lock)
	if !ok {
		return false, errors.New("not an Lock object")
	}

	exists := CheckMountPointExists(r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint})
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	return true, nil
}

func (r *LockReconciler) manageCleanUpLogic(lock *netconfv1.Lock) error {

	// Unlock the session
	s := Sessions[types.NamespacedName{Namespace: lock.Namespace, Name: lock.Spec.MountPoint}.String()]
	_, err := s.ExecRPC(message.NewUnlock(lock.Spec.Target))

	return err
}

func (r *LockReconciler) manageOperatorLogic(lock *netconfv1.Lock, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Send Lock on %s datastore.", lock.Spec.MountPoint, lock.Spec.Target))

	s := Sessions[types.NamespacedName{Namespace: lock.Namespace, Name: lock.Spec.MountPoint}.String()]
	reply, err := s.ExecRPC(message.NewLock(lock.Spec.Target))
	if err != nil {
		log.Error(err, fmt.Sprintf("%s: Failed to Lock.", lock.Spec.MountPoint))
		lock.Status.Status = "failed"
		lock.Status.RpcReply = reply.RawReply
		return err
	}
	log.Info(fmt.Sprintf("%s: Successfully executed lock operation.", lock.Spec.MountPoint))

	lock.Status.Status = "success"
	lock.Status.RpcReply = reply.Data

	return nil
}

func newLockReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &LockReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(lockControllerName),
			mgr.GetAPIReader(),
		),
	}
}

func addLock(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(lockControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.Lock{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

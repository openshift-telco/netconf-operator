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

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=editconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=editconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=editconfigs/finalizers,verbs=update

// EditConfigReconciler reconciles a EditConfig object
type EditConfigReconciler struct {
	util.ReconcilerBase
}

// AddEditConfig Add creates a new MountPoint Controller and adds it to the Manager.
func AddEditConfig(mgr manager.Manager) error {
	return addEditConfig(mgr, newEditConfigReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EditConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(editConfigControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling EditConfig")

	// Fetch the CRD instance
	instance := &netconfv1.EditConfig{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("EditConfig resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get EditConfig")
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

func (r *EditConfigReconciler) isInitialized(obj metav1.Object) bool {
	mountPoint, ok := obj.(*netconfv1.EditConfig)
	if !ok {
		return false
	}
	if util.HasFinalizer(mountPoint, finalizer) {
		return true
	}
	util.AddFinalizer(mountPoint, finalizer)
	return false

}

func (r *EditConfigReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.EditConfig)
	if !ok {
		return false, errors.New("not an EditConfig object")
	}

	exists := CheckMountPointExists(r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint})
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	return true, nil
}

func (r *EditConfigReconciler) manageCleanUpLogic(editConfig *netconfv1.EditConfig) error {
	// TODO NOOP
	return nil
}

func (r *EditConfigReconciler) manageOperatorLogic(editConfig *netconfv1.EditConfig, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: EditConfig for %s datastore with %s operation.", editConfig.Spec.MountPoint,
		editConfig.Spec.Target, editConfig.Spec.Operation))

	if !editConfig.Spec.DependsOn.IsNil() {
		err := validateDependency(r.ReconcilerBase, editConfig.Namespace, editConfig.Spec.DependsOn)
		if err != nil {
			log.Error(err, "Failed to validate dependency.")
			return err
		}
	}

	s := Sessions[types.NamespacedName{Namespace: editConfig.Namespace, Name: editConfig.Spec.MountPoint}.String()]
	reply, err := s.ExecRPC(message.NewEditConfig(editConfig.Spec.Target, editConfig.Spec.Operation, editConfig.Spec.XML))
	if err != nil {
		log.Error(err, "%s: Failed to EditConfig.", editConfig.Spec.MountPoint)
		editConfig.Status.Status = "failed"
		editConfig.Status.RpcReply = reply.RawReply
		return err
	}
	log.Info(fmt.Sprintf("%s: Successfully executed edit-config operation.", editConfig.Spec.MountPoint))

	editConfig.Status.Status = "success"
	editConfig.Status.RpcReply = reply.Data

	return nil
}

func newEditConfigReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &EditConfigReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(editConfigControllerName),
			mgr.GetAPIReader(),
		),
	}
}

func addEditConfig(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(editConfigControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.EditConfig{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

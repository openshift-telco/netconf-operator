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
	"github.com/go-logr/logr"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
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

	netconfv1 "github.com/openshift-telco/netconf-operator/api/v1"
)

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=getconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=getconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=getconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=events,verbs=get;list;watch;create;update;patch;delete

// GetConfigReconciler reconciles a GetConfig object
type GetConfigReconciler struct {
	util.ReconcilerBase
}

// AddGetConfig Add creates a new MountPoint Controller and adds it to the Manager.
func AddGetConfig(mgr manager.Manager) error {
	return addGetConfig(mgr, newGetConfigReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GetConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(getConfigControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling GetConfig")

	// Fetch the CRD instance
	instance := &netconfv1.GetConfig{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("GetConfig resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get GetConfig")
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
		return reconcile.Result{}, nil
	}

	err = r.manageOperatorLogic(instance, log)
	if err != nil {
		return r.ManageError(ctx, instance, err)
	}

	return r.ManageSuccess(ctx, instance)
}

func (r *GetConfigReconciler) isInitialized(obj metav1.Object) bool {
	_, ok := obj.(*netconfv1.GetConfig)
	if !ok {
		return false
	}
	return true

}

func (r *GetConfigReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.GetConfig)
	if !ok {
		return false, errors.New("not an GetConfig object")
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

func (r *GetConfigReconciler) manageOperatorLogic(obj *netconfv1.GetConfig, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Send GetConfig %s for %s datastore.", obj.Spec.MountPoint, obj.Name, obj.Spec.Target))

	// TODO implement filtering
	s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]
	reply, err := s.SyncRPC(message.NewGetConfig(obj.Spec.Target, "", ""), obj.Spec.Timeout)

	if err != nil || reply.Errors != nil {
		log.Info(
			fmt.Sprintf(
				"%s: Failed to GetConfig for %s datastore %s.", obj.Spec.MountPoint, obj.Spec.Target, obj.Name,
			),
		)
		obj.Status = "failed"
		obj.RpcReply = reply.RawReply
		return err
	}

	log.Info(fmt.Sprintf("%s: Successfully executed GetConfig operation %s.", obj.Spec.MountPoint, obj.Name))
	obj.Status = "success"
	obj.RpcReply = reply.Data
	return nil
}

func newGetConfigReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &GetConfigReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(getConfigControllerName),
			mgr.GetAPIReader(),
		),
	}
}

func addGetConfig(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(getConfigControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.GetConfig{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

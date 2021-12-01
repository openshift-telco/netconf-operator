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

//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=commits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=commits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=commits/finalizers,verbs=update

// CommitReconciler reconciles a Commit object
type CommitReconciler struct {
	util.ReconcilerBase
}

// AddCommit Add creates a new MountPoint Controller and adds it to the Manager.
func AddCommit(mgr manager.Manager) error {
	return addCommit(mgr, newCommitReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CommitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(commitControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Commit")

	// Fetch the CRD instance
	instance := &netconfv1.Commit{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Commit resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Commit")
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

func (r *CommitReconciler) isInitialized(obj metav1.Object) bool {
	_, ok := obj.(*netconfv1.Commit)
	if !ok {
		return false
	}
	return true
}

func (r *CommitReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.Commit)
	if !ok {
		return false, fmt.Errorf("%s is not an Commit object", obj.GetName())
	}

	exists := CheckMountPointExists(
		r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint},
	)
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	if !instance.Spec.DependsOn.IsNil() {
		err := validateDependency(r.ReconcilerBase, instance.Namespace, instance.Spec.DependsOn)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (r *CommitReconciler) manageOperatorLogic(obj *netconfv1.Commit, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Execute Commit operation %s.", obj.Spec.MountPoint, obj.Name))

	s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]
	reply, err := s.SyncRPC(message.NewCommit(), obj.Spec.Timeout)

	if err != nil || reply.Errors != nil {
		log.Info(fmt.Sprintf("%s: Failed to Commit %s", obj.Spec.MountPoint, obj.Name))
		obj.Status = "failed"
		obj.RpcReply = reply.RawReply
		return err
	}

	log.Info(fmt.Sprintf("%s: Successfully executed Commit operation %s.", obj.Spec.MountPoint, obj.Name))
	obj.Status = "success"
	obj.RpcReply = reply.Data
	return nil

}

func newCommitReconciler(mgr manager.Manager) reconcile.Reconciler {
	c := &CommitReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(commitControllerName),
			mgr.GetAPIReader(),
		),
	}
	return c
}

func addCommit(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New(commitControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.Commit{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

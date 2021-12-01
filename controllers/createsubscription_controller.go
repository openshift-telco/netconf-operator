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
	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/redhat-cop/operator-utils/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	netconfv1 "github.com/openshift-telco/netconf-operator/api/v1"
)

//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=createsubscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=createsubscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=createsubscriptions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// CreateSubscriptionReconciler reconciles a CreateSubscription object
type CreateSubscriptionReconciler struct {
	util.ReconcilerBase
	recorder record.EventRecorder
}

// AddCreateSubscription creates a new MountPoint Controller and adds it to the Manager.
func AddCreateSubscription(mgr manager.Manager) error {
	return addCreateSubscription(mgr, newCreateSubscriptionReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CreateSubscriptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(createSubscriptionControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling CreateSubscription")

	// Fetch the CRD instance
	instance := &netconfv1.CreateSubscription{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("CreateSubscription resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get CreateSubscription")
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

func (r *CreateSubscriptionReconciler) isInitialized(obj metav1.Object) bool {
	_, ok := obj.(*netconfv1.CreateSubscription)
	if !ok {
		return false
	}
	return true

}

func (r *CreateSubscriptionReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.CreateSubscription)
	if !ok {
		return false, fmt.Errorf("%s is not an CreateSubscription object", obj.GetName())
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

func (r *CreateSubscriptionReconciler) manageOperatorLogic(obj *netconfv1.CreateSubscription, log logr.Logger) error {
	identifier := obj.GetNamespacedName()

	log.Info(fmt.Sprintf("%s: Create NETCONF subscription %s.", obj.Spec.MountPoint, obj.Name))

	callback := func(event netconf.Event) {
		notification := event.Notification()
		// sends a K8S event
		r.recorder.Eventf(
			obj, "Normal", fmt.Sprintf("NetconfNotification-%s", identifier), fmt.Sprintf("%s", notification.RawReply),
		)
	}

	s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]
	err := s.CreateNotificationStream(
		obj.Spec.Timeout, obj.Spec.StopTime, obj.Spec.StartTime, obj.Spec.Stream, callback,
	)
	if err != nil {
		obj.Status = "failed"
		return err
	}

	obj.Status = "subscribed"
	return nil
}

func newCreateSubscriptionReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &CreateSubscriptionReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(),
			mgr.GetEventRecorderFor(createSubscriptionControllerName),
			mgr.GetAPIReader(),
		),
		recorder: mgr.GetEventRecorderFor(createSubscriptionControllerName),
	}
}

func addCreateSubscription(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(createSubscriptionControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.CreateSubscription{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

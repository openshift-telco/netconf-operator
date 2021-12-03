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
	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
	netconfv1 "github.com/openshift-telco/netconf-operator/api/v1"
	"github.com/redhat-cop/operator-utils/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=establishsubscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=establishsubscriptions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.openshift-telco.io,resources=establishsubscriptions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// EstablishSubscriptionReconciler reconciles a EstablishSubscription object
type EstablishSubscriptionReconciler struct {
	util.ReconcilerBase
	recorder record.EventRecorder
}

// AddEstablishSubscription Add creates a new MountPoint Controller and adds it to the Manager.
func AddEstablishSubscription(mgr manager.Manager) error {
	return addEstablishSubscription(mgr, newEstablishSubscriptionReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EstablishSubscriptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(establishSubscriptionControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling EstablishSubscription")

	// Fetch the CRD instance
	instance := &netconfv1.EstablishSubscription{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("EstablishSubscription resource not found. Since object must be deleted triggering clean up logic")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get EstablishSubscription")
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
		if !util.HasFinalizer(instance, establishSubscriptionFinalizer) {
			return reconcile.Result{}, nil
		}
		err := r.manageCleanUpLogic(instance)

		if err != nil {
			log.Error(err, "unable to delete instance", "instance", instance)
			return r.ManageError(ctx, instance, err)
		}
		util.RemoveFinalizer(instance, establishSubscriptionFinalizer)
		err = r.GetClient().Update(context.Background(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(ctx, instance, err)
		}
		return reconcile.Result{}, nil
	}

	err = r.manageOperatorLogic(instance, log)
	if err != nil {
		return r.ManageError(ctx, instance, err)
	}

	return r.ManageSuccess(ctx, instance)
}

func (r *EstablishSubscriptionReconciler) isInitialized(obj metav1.Object) bool {
	instance, ok := obj.(*netconfv1.EstablishSubscription)
	if !ok {
		return false
	}
	if util.HasFinalizer(instance, establishSubscriptionFinalizer) {
		return true
	}
	util.AddFinalizer(instance, establishSubscriptionFinalizer)
	return false

}

func (r *EstablishSubscriptionReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.EstablishSubscription)
	if !ok {
		return false, errors.New("not an EstablishSubscription object")
	}

	exists := CheckMountPointExists(
		r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint},
	)
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	err := ValidateXML(instance.Spec.XML, message.EstablishSubscription{})
	if err != nil {
		return false, err
	}

	return true, nil
}
func (r *EstablishSubscriptionReconciler) manageCleanUpLogic(obj *netconfv1.EstablishSubscription) error {
	subID := obj.SubscriptionID
	if subID != "" {
		s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]
		s.Listener.Remove(subID)

		// FIXME move to proper operation
		delSub := fmt.Sprintf(
			"<delete-subscription xmlns=\"urn:ietf:params:xml:ns:yang:ietf-event-notifications\"><subscription-id>%s</subscription-id></delete-subscription>",
			subID,
		)
		_, err := s.SyncRPC(message.NewRPC(delSub), 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *EstablishSubscriptionReconciler) manageOperatorLogic(
	obj *netconfv1.EstablishSubscription, log logr.Logger,
) error {
	log.Info(fmt.Sprintf("%s: Establish NETCONF subscription %s.", obj.Spec.MountPoint, obj.Name))

	s := Sessions[obj.GetMountPointNamespacedName(obj.Spec.MountPoint)]

	identifier := obj.GetNamespacedName()

	// In case of update to the CR, check if the identified EstablishSubscription exists already.
	// If so, do not proceed with new registration, skip instead.
	if obj.SubscriptionID != "" {
		log.Info(
			fmt.Sprintf(
				"%s: Skip EstablishSubscription for %s as it already exists with subscriptionID %s",
				obj.Spec.MountPoint, identifier, obj.SubscriptionID,
			),
		)
		return nil
	}

	reply, err := s.SyncRPC(message.NewEstablishSubscription(obj.Spec.XML), obj.Spec.Timeout)

	if err != nil || reply.Errors != nil {
		log.Info(
			fmt.Sprintf(
				"%s: Failed to Establish NETCONF Subscription %s with error %d.", obj.Spec.MountPoint, obj.Name, err,
			),
		)
		obj.Status = "failed"
		if reply != nil {
			obj.RpcReply = reply.RawReply
		}
		return err
	}

	log.Info(
		fmt.Sprintf(
			"%s: Successfully Established %s NETCONF Subscription for stream.",
			obj.Spec.MountPoint, obj.GetNamespacedName(),
		),
	)

	// Define callback for EstablishSubscription
	notificationCallback := func(event netconf.Event) {
		notification := event.Notification()
		if obj.Spec.KafkaSink.Enabled {
			err := SendToKafka(notification.RawReply, obj.Spec.KafkaSink)
			if err != nil {
				return
			}
		} else {
			// sends a K8S event
			r.recorder.Eventf(
				obj, "Normal", "NewEstablishSubscriptionNotification",
				fmt.Sprintf("%s", notification.RawReply),
			)
		}
	}

	// Register a new listener for upcoming NETCONF notifications for
	// that particular stream, identified by its subscriptionID
	s.Listener.Register(reply.SubscriptionID, notificationCallback)

	obj.Status = "subscribed"
	obj.SubscriptionID = reply.SubscriptionID

	return nil
}

func newEstablishSubscriptionReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &EstablishSubscriptionReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(),
			mgr.GetEventRecorderFor(establishSubscriptionControllerName),
			mgr.GetAPIReader(),
		),
		recorder: mgr.GetEventRecorderFor(establishSubscriptionControllerName),
	}
}

func addEstablishSubscription(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(establishSubscriptionControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.EstablishSubscription{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

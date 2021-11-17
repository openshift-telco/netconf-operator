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
	"github.com/adetalhouet/go-netconf/netconf"
	"github.com/adetalhouet/go-netconf/netconf/message"
	"github.com/go-logr/logr"
	"github.com/go-xmlfmt/xmlfmt"
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
	"strings"
	"sync"

	ctrl "sigs.k8s.io/controller-runtime"

	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
)

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=notifications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=notifications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=notifications/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// ActiveNotifications hold a map of active notification streams,
// identified by their CR NamespacedName (either CreateSubscription or
// EstablishSubscription). When the CR gets deleted, it is set to `false`
// and then removed from the map.
var ActiveNotifications = make(map[string]bool)

var ActiveStreams = make(map[string]bool)

var notificationWG sync.WaitGroup
var wgCount = 0

// NotificationReconciler reconciles a Notification object
type NotificationReconciler struct {
	util.ReconcilerBase
	recorder record.EventRecorder
}

// AddNotification Add creates a new MountPoint Controller and adds it to the Manager.
func AddNotification(mgr manager.Manager) error {
	return addNotification(mgr, newNotificationReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NotificationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(notificationControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Notification")

	// Fetch the CRD instance
	instance := &netconfv1.Notification{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Notification resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Notification")
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

func (r *NotificationReconciler) isInitialized(obj metav1.Object) bool {
	mountPoint, ok := obj.(*netconfv1.Notification)
	if !ok {
		return false
	}
	if util.HasFinalizer(mountPoint, finalizer) {
		return true
	}
	util.AddFinalizer(mountPoint, finalizer)
	return false

}

func (r *NotificationReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*netconfv1.Notification)
	if !ok {
		return false, errors.New("not an Notification object")
	}

	exists := CheckMountPointExists(
		r.ReconcilerBase,
		types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.MountPoint},
	)
	if !exists {
		return false, fmt.Errorf("MountPoint %s doesn't exists", instance.Spec.MountPoint)
	}

	if len(instance.Spec.EstablishSubscription) == 0 && len(instance.Spec.CreateSubscription) == 0 {
		return false, fmt.Errorf("either establishSubscription and/or createSubscription need to be provided")
	}

	for _, v := range instance.Spec.EstablishSubscription {
		err := ValidateXML(v.XML, message.EstablishSubscription{})
		if err != nil {
			return false, err
		}
	}

	return true, nil
}
func (r *NotificationReconciler) manageCleanUpLogic(notification *netconfv1.Notification) error {

	// this sets the conditions so receiveNotificationAsync for the according object release its goroutine
	ActiveNotifications[notification.GetNamespacedName()] = false

	for _, n := range notification.Spec.CreateSubscription {
		delete(ActiveStreams, notification.GetIdentifier(n.Name))
	}

	for _, e := range notification.Spec.EstablishSubscription {
		delete(ActiveStreams, notification.GetIdentifier(e.Name))
	}

	// Release all the retained threads
	// FIXME weak impl as race condition could occur and we could end up with negative delta
	//for i:=0; i<wgCount; i++ {
	//	notificationWG.Done()
	//}

	// TODO Wait for notification threads to finish
	// notificationWG.Wait()

	return nil
}

func (r *NotificationReconciler) manageOperatorLogic(notification *netconfv1.Notification, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Registering NETCONF subscription.", notification.Spec.MountPoint))

	s := Sessions[types.NamespacedName{Namespace: notification.Namespace, Name: notification.Spec.MountPoint}.String()]

	for _, n := range notification.Spec.CreateSubscription {

		// In case of update to the CR, check if the identified notification exists already.
		// If so, do not register proceed with new registration, skip instead.
		if _, ok := ActiveStreams[notification.GetIdentifier(n.Name)]; ok {
			continue
		}
		reply, err := s.ExecRPC(message.NewCreateSubscription(n.StopTime, n.StartTime, n.Stream))
		if err != nil {
			log.Error(
				err, fmt.Sprintf(
					"%s: Failed to Create %s NETCONF Subscription for stream %s.", notification.Spec.MountPoint, n.Name,
					n.Stream,
				),
			)
			notification.Status.Status = "failed"
			notification.Status.RpcReply = reply.RawReply
			return err
		}
		log.Info(
			fmt.Sprintf(
				"%s: Successfully Created %s NETCONF Subscription for stream `%s`.", notification.Spec.MountPoint,
				n.Name, n.Stream,
			),
		)

		ActiveStreams[notification.GetIdentifier(n.Name)] = true
	}

	for _, e := range notification.Spec.EstablishSubscription {

		// In case of update to the CR, check if the identified notification exists already.
		// If so, do not register proceed with new registration, skip instead.
		if _, ok := ActiveNotifications[notification.GetIdentifier(e.Name)]; ok {
			continue
		}

		reply, err := s.ExecRPC(message.NewEstablishSubscription(e.XML))
		// discard error related to notifications received before rpc-reply
		if err != nil && !strings.ContainsAny(err.Error(), "discarding received message") {
			log.Error(
				err,
				fmt.Sprintf("%s: Failed to Establish NETCONF Subscription %s.", notification.Spec.MountPoint, e.Name),
			)
			notification.Status.Status = "failed"
			notification.Status.RpcReply = reply.RawReply
			return err
		}
		log.Info(
			fmt.Sprintf(
				"%s: Successfully Establish NETCONF Subscription %s.", notification.Spec.MountPoint, e.Name,
			),
		)

		ActiveStreams[notification.GetIdentifier(e.Name)] = true
	}

	r.registerNotificationGoRoutine(notification, s)
	notification.Status.Status = "subscribed"

	return nil
}

func (r *NotificationReconciler) registerNotificationGoRoutine(
	notification *netconfv1.Notification, s *netconf.Session,
) {
	identifier := notification.GetNamespacedName()
	wgCount++
	notificationWG.Add(1)
	go func() {
		defer notificationWG.Done()
		defer func() {
			wgCount--
		}()
		receiveNotificationAsync(r.recorder, identifier, notification, s)
	}()

	ActiveNotifications[identifier] = true
}

func receiveNotificationAsync(
	recorder record.EventRecorder, identifier string, instance *netconfv1.Notification, s *netconf.Session,
) {

	condition := ActiveNotifications[identifier]
	for ok := true; ok; ok = condition {
		rawXML, err := s.Transport.Receive()
		if err != nil {
			panic(err)
		}

		var rawReply = string(rawXML)
		fmt.Printf("%+v", rawReply)
		if strings.Contains(rawReply, "<rpc-reply") {
			fmt.Printf("Ignore <rpc-reply - expecting only <notification")
			continue
		}
		fmt.Printf("%+v", rawReply)

		notification, err := message.NewNotification(rawXML)

		prettyXML := xmlfmt.FormatXML(notification.Data, "\t", "  ")

		recorder.Eventf(
			instance, "Normal", "NewNotification",
			fmt.Sprintf(
				"Received NETCONF notification for %s registration.\nNotification data:\n%s", identifier, prettyXML,
			),
		)
	}
}

func newNotificationReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &NotificationReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(),
			mgr.GetEventRecorderFor(notificationControllerName),
			mgr.GetAPIReader(),
		),
		recorder: mgr.GetEventRecorderFor(notificationControllerName),
	}
}

func addNotification(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(notificationControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.Notification{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

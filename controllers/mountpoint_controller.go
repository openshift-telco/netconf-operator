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
	"golang.org/x/crypto/ssh"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"

	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
	"github.com/redhat-cop/operator-utils/pkg/util"
)

//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=mountpoints,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=mountpoints/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netconf.adetalhouet.io,resources=mountpoints/finalizers,verbs=update

// MountPointReconciler reconciles a MountPoint object
type MountPointReconciler struct {
	util.ReconcilerBase
}

// AddMountPoint Add creates a new MountPoint Controller and adds it to the Manager.
func AddMountPoint(mgr manager.Manager) error {
	return addMountPoint(mgr, newMountPointReconciler(mgr))
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MountPointReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var log = logf.Log.WithName(mountPointControllerName)

	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling MountPoint")

	// Fetch the CRD instance
	instance := &netconfv1.MountPoint{}
	err := r.GetClient().Get(context.Background(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("MountPoint resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get MountPoint")
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

		// remove cached session from inventory
		delete(Sessions, instance.GetNamespacedName())

		return reconcile.Result{}, nil
	}

	// Managing MountPoint Logic
	err = r.manageOperatorLogic(instance, log)
	if err != nil {
		return r.ManageError(ctx, instance, err)
	}

	return r.ManageSuccess(ctx, instance)
}

func (r *MountPointReconciler) isInitialized(obj metav1.Object) bool {
	mountPoint, ok := obj.(*netconfv1.MountPoint)
	if !ok {
		return false
	}
	if util.HasFinalizer(mountPoint, finalizer) {
		return true
	}
	util.AddFinalizer(mountPoint, finalizer)
	return false

}

func (r *MountPointReconciler) isValid(obj metav1.Object) (bool, error) {
	_, ok := obj.(*netconfv1.MountPoint)
	if !ok {
		return false, errors.New("not an MountPoint object")
	}

	return true, nil
}

func (r *MountPointReconciler) manageCleanUpLogic(mountPoint *netconfv1.MountPoint) error {
	s := Sessions[mountPoint.GetNamespacedName()]
	// Close NETCONF session. If fails, kill the session.
	rpc, err := s.ExecRPC(message.NewCloseSession())
	if err != nil || !rpc.Ok {
		// If there is a failure here, there is nothing we can do.
		_, _ = s.ExecRPC(message.NewKillSession(string(rune(s.SessionID))))
		return err
	}
	return s.Close()
}

func (r *MountPointReconciler) manageOperatorLogic(mountPoint *netconfv1.MountPoint, log logr.Logger) error {
	log.Info(fmt.Sprintf("%s: Create Netconf connection to %s.", mountPoint.Name, mountPoint.Spec.Target))

	sshConfig := &ssh.ClientConfig{
		User:            mountPoint.Spec.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(mountPoint.Spec.Username)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var session *netconf.Session
	var err error

	// Establish SSH session
	if mountPoint.Spec.Timeout == 0 {
		session, err = netconf.DialSSH(mountPoint.Spec.Target, sshConfig)
	} else {
		timeout := time.Duration(mountPoint.Spec.Timeout) * time.Second
		session, err = netconf.DialSSHTimeout(mountPoint.Spec.Target, sshConfig, timeout)
	}
	if err != nil {
		log.Error(err, fmt.Sprintf("%s: Failed to established SSH connection to %s.",
			mountPoint.Name, mountPoint.Spec.Target))
		mountPoint.Status.Status = "failed"
		return err
	}

	// Send our hello using default capabilities + additional capabilities, as defined in the CR.
	capabilities := netconf.DefaultCapabilities
	for _, capability := range mountPoint.Spec.AdditionalCapabilities {
		capabilities = append(capabilities, capability)
	}

	err = session.SendHello(&message.Hello{Capabilities: capabilities})
	if err != nil {
		log.Error(err, fmt.Sprintf("%s: Failed to send hello-message", mountPoint.Name))
		mountPoint.Status.Status = "failed"
		return err
	}

	Sessions[mountPoint.GetNamespacedName()] = session

	log.Info(fmt.Sprintf("%s: Successfully connected.", mountPoint.Name))

	mountPoint.Status.Status = "connected"
	mountPoint.Status.Capabilities = session.Capabilities

	return nil
}

func newMountPointReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &MountPointReconciler{
		ReconcilerBase: util.NewReconcilerBase(
			mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(mountPointControllerName),
			mgr.GetAPIReader(),
		),
	}
}

func addMountPoint(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(mountPointControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &netconfv1.MountPoint{}}, &handler.EnqueueRequestForObject{},
		util.ResourceGenerationOrFinalizerChangedPredicate{},
	)
	if err != nil {
		return err
	}

	return nil
}

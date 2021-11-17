package controllers

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/adetalhouet/go-netconf/netconf"
	netconfv1 "github.com/adetalhouet/netconf-operator/api/v1"
	"github.com/redhat-cop/operator-utils/pkg/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const mountPointControllerName = "MountPoint_crd"
const commitControllerName = "Commit_crd"
const getControllerName = "Get_crd"
const getConfigControllerName = "GetConfig_crd"
const editConfigControllerName = "EditConfig_crd"
const lockControllerName = "Lock_crd"
const unlockControllerName = "Unlock_crd"
const rpcControllerName = "RPC_crd"
const notificationControllerName = "Notification_crds"
const establishSubscriptionControllerName = "EstablishSubscription_crd"
const finalizer = "io.adetalhouet.netconf.finalizer"

// Sessions hold the active SSH session to Netconf servers.
// The key is the NamespacedName of the MountPoint object referred in the CR
var Sessions = make(map[string]*netconf.Session)

// CheckMountPointExists validates a MountPoint, defined by its namespacedName, exists
func CheckMountPointExists(r util.ReconcilerBase, namespacedName types.NamespacedName) bool {
	// Check MountPoint CRD exists
	instance := &netconfv1.MountPoint{}
	err := r.GetClient().Get(context.Background(), namespacedName, instance)
	if err != nil {
		return false
	}

	// Check MountPoint session exists
	if _, ok := Sessions[instance.GetNamespacedName()]; ok {
		//TODO check session is still connected, or reconnect session
	} else {
		return false
	}
	return true
}

// ValidateXML checks a provided string can be properly unmarshall in the specified struct
func ValidateXML(data string, dataStruct interface{}) error {
	err := xml.Unmarshal([]byte(data), &dataStruct)
	if err != nil {
		return fmt.Errorf("provided XML is not valid: %s. \n%s", data, err)
	}
	return nil
}

func validateDependency(r util.ReconcilerBase, namespace string, dep netconfv1.DependsOn) error {

	var instance client.Object

	switch dep.Kind {
	case "Commit":
		instance = &netconfv1.Commit{}
	case "EditConfig":
		instance = &netconfv1.EditConfig{}
	case "Lock":
		instance = &netconfv1.Lock{}
	default:
		return fmt.Errorf(
			"invalid dependendy. Only Commit, EditConfig and Lock are supported. %s was provided", dep.Kind,
		)
	}
	err := validateExist(r, namespace, dep, instance)
	if err != nil {
		return err
	}

	switch dep.Kind {
	case "Commit":
		i, _ := instance.(*netconfv1.Commit)
		return validateStatus(i.Status.Status, dep.Name, namespace)
	case "EditConfig":
		i, _ := instance.(*netconfv1.EditConfig)
		return validateStatus(i.Status.Status, dep.Name, namespace)
	case "Lock":
		i, _ := instance.(*netconfv1.Lock)
		return validateStatus(i.Status.Status, dep.Name, namespace)
	}

	return nil
}

func validateExist(r util.ReconcilerBase, namespace string, dep netconfv1.DependsOn, instance client.Object) error {
	err := r.GetClient().Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: dep.Name}, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("provided resource %s not found in namespace %s", dep.Name, namespace)
		}
		return fmt.Errorf("failed to read resource %s from namespace %s", dep.Name, namespace)
	}
	return nil
}

func validateStatus(status string, name string, namespace string) error {
	if status == "success" {
		return nil
	}
	return fmt.Errorf("Dependent resource %s from namespace %s is in %s state", name, namespace, status)
}

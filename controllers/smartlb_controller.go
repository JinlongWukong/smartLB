/*


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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	lbv1 "smartLB/api/v1"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("SmartLB controller")
var Events = make(chan event.GenericEvent, 10)

// SmartLBReconciler reconciles a SmartLB object
type SmartLBReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *SmartLBReconciler) deleteExternalDependency(smartlb *lbv1.SmartLB, svc *corev1.Service) error {
	log.Info("deleting the internal/external dependencies")
	output, _ := json.Marshal(smartlb.Status)
	log.Info(string(output))

	// remove ExernalIPs from service
	svc.Spec.ExternalIPs = []string{}
	if err := r.Client.Update(context.Background(), svc); err != nil {
		log.Error(err, "Unable to remove service external IP")
		return err
	}
	log.Info("Remove service externalIPs successfully")

	// remove Loadbalancer configuration from external if existed
	if uri := smartlb.Spec.Subscribe; uri != "" {
		client := &http.Client{}
		req, _ := http.NewRequest("DELETE", uri, bytes.NewBuffer(output))
		req.Header.Set("Content-type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Error(err, "http request failed")
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			log.Info("Remove external Loadbalance configuration successfully")
		} else {
			return fmt.Errorf("wrong status-code returned from external Loadbalancer")
		}
	}

	log.Info("the internal/external dependencies were deleted")

	return nil
}

// +kubebuilder:rbac:groups=lb.my.domain,resources=smartlbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lb.my.domain,resources=smartlbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lb.my.domain,resources=smartlbs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=endpoints/status,verbs=get

func (r *SmartLBReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("smartlb", req.NamespacedName)

	smartlb := &lbv1.SmartLB{}

	if err := r.Get(ctx, req.NamespacedName, smartlb); err != nil {
		log.Error(err, "unable to fetch smartLB")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	objkey := client.ObjectKey{
		Namespace: smartlb.Spec.Namespace,
		Name:      smartlb.Spec.Service,
	}

	// fetch service info
	svc := &corev1.Service{}
	if err := r.Get(ctx, objkey, svc); err != nil {
		log.Error(err, "unable to fetch service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ports := make([]int32, 0)
	for _, port := range svc.Spec.Ports {
		ports = append(ports, port.Port)
	}
	log.Info("Service Ports Found: " + fmt.Sprint(ports))

	// name of your custom finalizer
	myFinalizerName := "cleanup"
	if smartlb.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(smartlb.ObjectMeta.Finalizers, myFinalizerName) {
			smartlb.ObjectMeta.Finalizers = append(smartlb.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Client.Update(ctx, smartlb); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if containsString(smartlb.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteExternalDependency(smartlb, svc); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			smartlb.ObjectMeta.Finalizers = removeString(smartlb.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Client.Update(ctx, smartlb); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Our finalizer has finished, so the reconciler can do nothing.
		return reconcile.Result{}, nil
	}

	// fetch endpoints info
	endpoint := &corev1.Endpoints{}
	if err := r.Get(ctx, objkey, endpoint); err != nil {
		log.Error(err, "unable to fetch endpoints")
		return ctrl.Result{}, err
	}

	//generate node status
	appendIfMissing := func(slice []lbv1.NodeStatus, i lbv1.NodeStatus) []lbv1.NodeStatus {
		for _, ele := range slice {
			if reflect.DeepEqual(ele, i) {
				return slice
			}
		}
		return append(slice, i)
	}
	nodeInfo := []lbv1.NodeStatus{}
	for _, subnet := range endpoint.Subsets {
		for _, address := range subnet.Addresses {
			nodeInfo = appendIfMissing(nodeInfo, lbv1.NodeStatus{IP: *address.NodeName, Port: ports})
			log.Info("Node IP found: " + *address.NodeName)
		}
	}

	//update service.spec externalIps
	service := []string{smartlb.Spec.Vip}
	if !reflect.DeepEqual(svc.Spec.ExternalIPs, service) {
		svc.Spec.ExternalIPs = service
		if err := r.Client.Update(ctx, svc); err != nil {
			log.Error(err, "Unable to update service")
			return ctrl.Result{}, err
		}
		log.Info("Service externalIps was updated")
		r.Recorder.Event(smartlb, "Normal", "Updated",
			fmt.Sprintf("service %s was updated with a ExternalIP %s", svc.Name, smartlb.Spec.Vip))
	}
	//update smartlb status
	if !reflect.DeepEqual(smartlb.Status.NodeList, nodeInfo) ||
		!reflect.DeepEqual(smartlb.Status.ExternalIP, smartlb.Spec.Vip) {
		smartlb.Status.NodeList = nodeInfo
		smartlb.Status.ExternalIP = smartlb.Spec.Vip
		if err := r.Client.Status().Update(ctx, smartlb); err != nil {
			log.Error(err, "unable to update smartlb status")
			return ctrl.Result{}, err
		} else {
			log.Info("smartlb status was updated")
		}
	}

	output, _ := json.Marshal(smartlb.Status)
	log.Info(string(output))
	//send to external lb
	if uri := smartlb.Spec.Subscribe; uri != "" {
		resp, err := http.Post(uri, "application/json", bytes.NewBuffer(output))
		if err != nil {
			log.Error(err, "Http request failed")
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 == http.StatusOK/100 {
			log.Info("External LB configure successfully")
		} else {
			log.Info("External LB configure failed", "Return status code: ", resp.StatusCode)
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *SmartLBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Define a mapping from the object in the event to one or more
	// objects to Reconcile

	svcMapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			lb := &lbv1.SmartLBList{}
			if err := r.Client.List(context.Background(), lb); err != nil {
				log.Error(err, "List smartLB items failed!")
				return nil
			}
			for _, item := range lb.Items {
				if item.Spec.Service == a.Meta.GetName() &&
					item.Spec.Namespace == a.Meta.GetNamespace() {
					return []reconcile.Request{
						{NamespacedName: types.NamespacedName{
							Name:      item.ObjectMeta.Name,
							Namespace: item.ObjectMeta.Namespace,
						}},
					}
				}
			}
			return nil
		})
	//External generic event, return all CRs
	requestMapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			lb := &lbv1.SmartLBList{}
			if err := r.Client.List(context.Background(), lb); err != nil {
				log.Error(err, "List smartLB items failed!")
				return nil
			}
			requests := make([]reconcile.Request, 0)
			for _, item := range lb.Items {
				temp := reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      item.ObjectMeta.Name,
					Namespace: item.ObjectMeta.Namespace,
				}}
				requests = append(requests, temp)
			}
			return requests
		})

	p := predicate.Funcs{
		//If service defined in CRs, pass
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.Info("Update event")
			lb := &lbv1.SmartLBList{}
			if err := r.Client.List(context.Background(), lb); err != nil {
				log.Error(err, "List smartLB items failed!")
				return false
			}
			for _, item := range lb.Items {
				if item.Spec.Service == e.MetaNew.GetName() &&
					item.Spec.Namespace == e.MetaNew.GetNamespace() {
					return true
				}
			}
			return false
		},
		//If service defined in CRs, pass
		CreateFunc: func(e event.CreateEvent) bool {
			log.Info("Creat event")
			lb := &lbv1.SmartLBList{}
			if err := r.Client.List(context.Background(), lb); err != nil {
				log.Error(err, "List smartLB items failed!")
				return false
			}
			for _, item := range lb.Items {
				if item.Spec.Service == e.Meta.GetName() &&
					item.Spec.Namespace == e.Meta.GetNamespace() {
					return true
				}
			}
			return false
		},
		//External event, always return true currently
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			log.Info("Generic event")
			return true
		},
	}
	servicePrct := builder.WithPredicates(p)

	//Watch self
	//Watch services
	//Watch channel wire to external request
	return ctrl.NewControllerManagedBy(mgr).
		For(&lbv1.SmartLB{}).
		Watches(&source.Kind{Type: &corev1.Service{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: svcMapFn,
			},
			servicePrct,
		).
		Watches(&source.Channel{Source: Events},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: requestMapFn,
			},
			servicePrct,
		).
		Complete(r)
}

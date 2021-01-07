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

package v1

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"smartLB/controllers/ipam"
)

// log is for logging in this package.
var log = logf.Log.WithName("smartlb-webhook")

func (r *SmartLB) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-lb-my-domain-v1-smartlb,mutating=true,failurePolicy=fail,groups=lb.my.domain,resources=smartlbs,verbs=create;update,versions=v1,name=msmartlb.kb.io

var _ webhook.Defaulter = &SmartLB{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *SmartLB) Default() {
	log.Info("default", "name", r.Name)

	// Apply ip of not given
	if r.Spec.Vip == "" {
		if ip, err := ipam.VipPool.Apply(r.Spec.String()); err != nil {
			log.Error(err, "apply ip failed")
		} else {
			log.Info("Apply IP returned", "address: ", ip)
			r.Spec.Vip = ip
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-lb-my-domain-v1-smartlb,mutating=false,failurePolicy=fail,groups=lb.my.domain,resources=smartlbs,versions=v1,name=vsmartlb.kb.io

var _ webhook.Validator = &SmartLB{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateCreate() error {
	log.Info("validate create", "name", r.Name)

	// Check Vip is given
	if r.Spec.Vip == "" {
		return fmt.Errorf("VIP should not be empty")
	}

	// Make sure address unique
	if err := ipam.VipPool.MarkOwner(r.Spec.Vip, r.Spec.String()); err != nil {
		return err
	}

	// Release address if error
	if err := r.ValidateSmartLB(); err != nil {
		log.Info("Error occur, address will be released", "address: ", r.Spec.Vip)
		ipam.VipPool.ReleaseOwner(r.Spec.Vip, r.Spec.String())
		return err
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateUpdate(old runtime.Object) error {
	log.Info("validate update", "name", r.Name)

	//var oldSmartLB = old.(*SmartLB)

	// The object is being deleted, return
	if !r.DeletionTimestamp.IsZero() {
		return nil
	}

	// Check Vip is given
	if r.Spec.Vip == "" {
		return fmt.Errorf("VIP should not be empty")
	}

	// Make sure address unique
	if err := ipam.VipPool.MarkOwner(r.Spec.Vip, r.Spec.String()); err != nil {
		return err
	}

	return r.ValidateSmartLB()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateDelete() error {
	log.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

// Generate part of validate
func (r *SmartLB) ValidateSmartLB() error {
	log.Info("validate smartLB", "name", r.Name)

	// Check subscriber is reachable
	if uri := r.Spec.Subscribe; uri != "" {
		resp, err := http.Get(uri)
		if err != nil {
			log.Error(err, "Subscribe uri check failed!")
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			log.Info("Subscribe uri check passed!")
		} else {
			log.Info("Subscribe uri check failed! ", "unexpected status code", resp.StatusCode)
			err = fmt.Errorf("subscribe uri return status code: %v", resp.StatusCode)
			return err
		}
	}

	return nil
}

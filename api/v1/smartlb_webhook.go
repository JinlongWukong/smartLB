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
	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var smartlblog = logf.Log.WithName("smartlb-resource")

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
	smartlblog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-lb-my-domain-v1-smartlb,mutating=false,failurePolicy=fail,groups=lb.my.domain,resources=smartlbs,versions=v1,name=vsmartlb.kb.io

var _ webhook.Validator = &SmartLB{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateCreate() error {
	smartlblog.Info("validate create", "name", r.Name)

	return r.ValidateSmartLB()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateUpdate(old runtime.Object) error {
	smartlblog.Info("validate update", "name", r.Name)

	return r.ValidateSmartLB()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SmartLB) ValidateDelete() error {
	smartlblog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

// Generate part of validate
func (r *SmartLB) ValidateSmartLB() error {
	smartlblog.Info("validate smartLB", "name", r.Name)

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
			return fmt.Errorf("subscribe uri status code unexpected! %v", resp.StatusCode)
		}
	}

	return nil
}

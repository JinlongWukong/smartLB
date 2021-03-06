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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// +kubebuilder:subresource:status
// SmartLBSpec defines the desired state of SmartLB
type SmartLBSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Service   string `json:"service"`
	Namespace string `json:"namespace"`
	Vip       string `json:"vip"`
	Scheduler string `json:"scheduler,omitempty"`
	Subscribe string `json:"subscribe,omitempty"`
}

func (slb SmartLBSpec) String() string {
	return fmt.Sprintf("%s,%s", slb.Namespace, slb.Service)
}

// SmartLBStatus defines the observed state of SmartLB
type SmartLBStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nodes      []string     `json:"ipaddress"`
	ExternalIP string       `json:"vip"`
	Ports      []PortStatus `json:"ports"`
}

type PortStatus struct {
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

func (ps PortStatus) String() string {
	return ps.Protocol + strconv.Itoa(int(ps.Port))
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SmartLB is the Schema for the smartlbs API
// +kubebuilder:printcolumn:name="Vip",type=string,JSONPath=`.status.vip`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type SmartLB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SmartLBSpec   `json:"spec,omitempty"`
	Status SmartLBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SmartLBList contains a list of SmartLB
type SmartLBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SmartLB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SmartLB{}, &SmartLBList{})
}

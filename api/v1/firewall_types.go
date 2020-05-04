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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FireWallSpec defines the desired state of FireWall
type FireWallSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	DeploymentName     string `json:"deploymentName"`
	SecondOperatorName string `json:"secondOperatorName"`
}

// FireWallStatus defines the observed state of FireWall
type FireWallStatus struct {
	// DeploymentReplica è opzionale, quindi aggiungo omitempty
	DeploymentReplicas int32 `json:"deploymentReplicas,omitempty"`
}

// +kubebuilder:object:root=true

// FireWall is the Schema for the firewalls API
type FireWall struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FireWallSpec   `json:"spec,omitempty"`
	Status FireWallStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FireWallList contains a list of FireWall
type FireWallList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FireWall `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FireWall{}, &FireWallList{})
}

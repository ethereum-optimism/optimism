/*
Copyright 2021.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActorSpec defines the desired state of Actor
type ActorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image                 string          `json:"image,omitempty"`
	L1URL                 string          `json:"l1_url"`
	L2URL                 string          `json:"l2_url"`
	PrivateKey            *Valuer         `json:"private_key,omitempty"`
	AddressManagerAddress string          `json:"address_manager_address"`
	TestFilename          string          `json:"test_filename,omitempty"`
	Concurrency           int             `json:"concurrency,omitempty"`
	RunForMS              int             `json:"run_for_ms,omitempty"`
	RunCount              int             `json:"run_count,omitempty"`
	ThinkTimeMS           int             `json:"think_time_ms,omitempty"`
	Env                   []corev1.EnvVar `json:"env,omitempty"`
}

// ActorStatus defines the observed state of Actor
type ActorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Actor is the Schema for the actors API
type Actor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActorSpec   `json:"spec,omitempty"`
	Status ActorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ActorList contains a list of Actor
type ActorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Actor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Actor{}, &ActorList{})
}

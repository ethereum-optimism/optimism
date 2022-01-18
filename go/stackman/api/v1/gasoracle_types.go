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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GasOracleSpec defines the desired state of GasOracle
type GasOracleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image            string      `json:"image,omitempty"`
	L1URL            string      `json:"l1_url,omitempty"`
	L1TimeoutSeconds int         `json:"l1_timeout_seconds,omitempty"`
	L2URL            string      `json:"l2_url,omitempty"`
	L2TimeoutSeconds int         `json:"l2_timeout_seconds,omitempty"`
	Env              []v1.EnvVar `json:"env,omitempty"`
}

// GasOracleStatus defines the observed state of GasOracle
type GasOracleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GasOracle is the Schema for the gasoracles API
type GasOracle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GasOracleSpec   `json:"spec,omitempty"`
	Status GasOracleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GasOracleList contains a list of GasOracle
type GasOracleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GasOracle `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GasOracle{}, &GasOracleList{})
}

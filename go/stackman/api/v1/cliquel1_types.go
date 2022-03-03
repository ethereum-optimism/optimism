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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CliqueL1Spec defines the desired state of CliqueL1
type CliqueL1Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Image            string     `json:"image,omitempty"`
	GenesisFile      *Valuer    `json:"genesis_file,omitempty"`
	SealerPrivateKey *Valuer    `json:"sealer_private_key"`
	SealerAddress    string     `json:"sealer_address,omitempty"`
	ChainID          int        `json:"chain_id,omitempty"`
	DataPVC          *PVCConfig `json:"data_pvc,omitempty"`
	AdditionalArgs   []string   `json:"additional_args,omitempty"`
}

// CliqueL1Status defines the observed state of CliqueL1
type CliqueL1Status struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CliqueL1 is the Schema for the cliquel1s API
type CliqueL1 struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CliqueL1Spec   `json:"spec,omitempty"`
	Status CliqueL1Status `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CliqueL1List contains a list of CliqueL1
type CliqueL1List struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CliqueL1 `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CliqueL1{}, &CliqueL1List{})
}

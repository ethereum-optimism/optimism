package v1

import (
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Valuer struct {
	Value     string               `json:"value,omitempty"`
	ValueFrom *corev1.EnvVarSource `json:"value_from,omitempty"`
}

func (v *Valuer) String() string {
	out, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func (v *Valuer) EnvVar(name string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:      name,
		Value:     v.Value,
		ValueFrom: v.ValueFrom,
	}
}

type PVCConfig struct {
	Name    string             `json:"name"`
	Storage *resource.Quantity `json:"storage,omitempty"`
}

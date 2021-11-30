package v1

import corev1 "k8s.io/api/core/v1"

type Valuer struct {
	Value     string               `json:"value,omitempty"`
	ValueFrom *corev1.EnvVarSource `json:"value_from,omitempty"`
}

func (v *Valuer) EnvVar(name string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:      name,
		Value:     v.Value,
		ValueFrom: v.ValueFrom,
	}
}

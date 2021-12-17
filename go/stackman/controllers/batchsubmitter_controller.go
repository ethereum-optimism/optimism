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

package controllers

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// BatchSubmitterReconciler reconciles a BatchSubmitter object
type BatchSubmitterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=batchsubmitters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=batchsubmitters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=batchsubmitters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BatchSubmitter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *BatchSubmitterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.BatchSubmitter{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("batch submitter resource not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting batch submitter")
		return ctrl.Result{}, err
	}

	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.entrypointsCfgMap(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "batch-submitter-entrypoints"), &corev1.ConfigMap{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	statefulSet := &appsv1.StatefulSet{}
	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.statefulSet(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "batch-submitter"), statefulSet)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	argsHash := r.statefulSetArgsHash(crd)
	if statefulSet.Labels["args_hash"] != argsHash {
		err := r.Update(ctx, r.statefulSet(crd))
		if err != nil {
			lgr.Error(err, "error updating batch submitter statefulSet")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "batch-submitter"), &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BatchSubmitterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.BatchSubmitter{}).
		Complete(r)
}

func (r *BatchSubmitterReconciler) labels() map[string]string {
	return map[string]string{
		"app": "batch-submitter",
	}
}

func (r *BatchSubmitterReconciler) entrypointsCfgMap(crd *stackv1.BatchSubmitter) *corev1.ConfigMap {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "batch-submitter-entrypoints", r.labels()),
		Data: map[string]string{
			"entrypoint.sh": BatchSubmitterEntrypoint,
		},
	}
	ctrl.SetControllerReference(crd, cfgMap, r.Scheme)
	return cfgMap
}

func (r *BatchSubmitterReconciler) statefulSetArgsHash(crd *stackv1.BatchSubmitter) string {
	h := md5.New()
	h.Write([]byte(crd.Spec.Image))
	h.Write([]byte(crd.Spec.L1URL))
	h.Write([]byte(strconv.Itoa(crd.Spec.L1TimeoutSeconds)))
	h.Write([]byte(crd.Spec.L2URL))
	h.Write([]byte(strconv.Itoa(crd.Spec.L2TimeoutSeconds)))
	h.Write([]byte(crd.Spec.DeployerURL))
	h.Write([]byte(strconv.Itoa(crd.Spec.DeployerTimeoutSeconds)))
	for _, ev := range crd.Spec.Env {
		h.Write([]byte(ev.String()))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (r *BatchSubmitterReconciler) statefulSet(crd *stackv1.BatchSubmitter) *appsv1.StatefulSet {
	replicas := int32(1)
	defaultMode := int32(0o777)
	labels := r.labels()
	labels["args_hash"] = r.statefulSetArgsHash(crd)
	initContainers := []corev1.Container{
		{
			Name:            "wait-for-l1",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				Hostify(crd.Spec.L1URL),
				"-t",
				strconv.Itoa(crd.Spec.L1TimeoutSeconds),
			},
		},
		{
			Name:            "wait-for-l2",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				Hostify(crd.Spec.L2URL),
				"-t",
				strconv.Itoa(crd.Spec.L2TimeoutSeconds),
			},
		},
	}
	baseEnv := []corev1.EnvVar{
		{
			Name:  "L1_NODE_WEB3_URL",
			Value: crd.Spec.L1URL,
		},
		{
			Name:  "L2_NODE_WEB3_URL",
			Value: crd.Spec.L2URL,
		},
		{
			Name:  "RUN_METRICS_SERVER",
			Value: "true",
		},
		{
			Name:  "METRICS_HOSTNAME",
			Value: "0.0.0.0",
		},
		{
			Name:  "METRICS_PORT",
			Value: "9090",
		},
	}
	if crd.Spec.DeployerURL != "" {
		initContainers = append(initContainers, corev1.Container{
			Name:            "wait-for-deployer",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				Hostify(crd.Spec.DeployerURL),
				"-t",
				strconv.Itoa(crd.Spec.DeployerTimeoutSeconds),
			},
		})
		baseEnv = append(baseEnv, []corev1.EnvVar{
			{
				Name:  "ROLLUP_STATE_DUMP_PATH",
				Value: "http://deployer:8081/state-dump.latest.json",
			},
			{
				Name:  "DEPLOYER_URL",
				Value: crd.Spec.DeployerURL,
			},
		}...)
	}
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "batch-submitter", labels),
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "batch-submitter",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: r.labels(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyAlways,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:            "batch-submitter",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							WorkingDir:      "/opt/optimism/packages/batch-submitter",
							Command: []string{
								"/bin/sh",
								"/opt/entrypoints/entrypoint.sh",
								"npm",
								"run",
								"start",
							},
							Env: append(baseEnv, crd.Spec.Env...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "entrypoints",
									MountPath: "/opt/entrypoints",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 9090,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "entrypoints",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: ObjectName(crd.ObjectMeta, "batch-submitter-entrypoints"),
									},
									DefaultMode: &defaultMode,
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, statefulSet, r.Scheme)
	return statefulSet
}

func (r *BatchSubmitterReconciler) service(crd *stackv1.BatchSubmitter) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "batch-submitter", r.labels()),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "batch-submitter",
			},
			Ports: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: 9090,
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

const BatchSubmitterEntrypoint = `
#!/bin/sh

if [ -n "$DEPLOYER_URL" ]; then
	echo "Loading addresses from $DEPLOYER_URL."
	ADDRESSES=$(curl --fail --show-error --silent "$DEPLOYER_URL/addresses.json")
	export ADDRESS_MANAGER_ADDRESS=$(echo $ADDRESSES | jq -r ".AddressManager")
fi

exec "$@"
`

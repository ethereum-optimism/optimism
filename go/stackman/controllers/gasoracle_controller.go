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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// GasOracleReconciler reconciles a GasOracle object
type GasOracleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=gasoracles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=gasoracles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=gasoracles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GasOracle object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *GasOracleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.GasOracle{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("gas oracle resource not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting gas oracle")
		return ctrl.Result{}, err
	}

	deployment := &appsv1.Deployment{}
	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.deployment(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "gas-oracle"), deployment)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	argsHash := r.deploymentArgsHash(crd)
	if deployment.Labels["args_hash"] != argsHash {
		err := r.Update(ctx, r.deployment(crd))
		if err != nil {
			lgr.Error(err, "error updating gas oracle deployment")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GasOracleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.GasOracle{}).
		Complete(r)
}

func (r *GasOracleReconciler) labels() map[string]string {
	return map[string]string{
		"app": "gas-oracle",
	}
}

func (r *GasOracleReconciler) deploymentArgsHash(crd *stackv1.GasOracle) string {
	h := md5.New()
	h.Write([]byte(crd.Spec.Image))
	h.Write([]byte(crd.Spec.L1URL))
	h.Write([]byte(strconv.Itoa(crd.Spec.L1TimeoutSeconds)))
	h.Write([]byte(crd.Spec.L2URL))
	h.Write([]byte(strconv.Itoa(crd.Spec.L2TimeoutSeconds)))
	for _, ev := range crd.Spec.Env {
		h.Write([]byte(ev.String()))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (r *GasOracleReconciler) deployment(crd *stackv1.GasOracle) *appsv1.Deployment {
	replicas := int32(1)
	labels := r.labels()
	labels["args_hash"] = r.deploymentArgsHash(crd)
	baseEnv := []corev1.EnvVar{
		{
			Name:  "GAS_PRICE_ORACLE_ETHEREUM_HTTP_URL",
			Value: crd.Spec.L1URL,
		},
		{
			Name:  "GAS_PRICE_ORACLE_LAYER_TWO_HTTP_URL",
			Value: crd.Spec.L2URL,
		},
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "gas-oracle", labels),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "gas-oracle",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.labels(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					InitContainers: []corev1.Container{
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
					},
					Containers: []corev1.Container{
						{
							Name:            "gpo",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Env:             append(baseEnv, crd.Spec.Env...),
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, deployment, r.Scheme)
	return deployment
}

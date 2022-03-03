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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// ActorReconciler reconciles a Actor object
type ActorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=actors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=actors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=actors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Actor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *ActorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.Actor{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("actor resource not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting actor")
		return ctrl.Result{}, err
	}

	deployment := &appsv1.Deployment{}
	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.deployment(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "actor"), deployment)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	argsHash := r.argsHash(crd)
	if deployment.Labels["args_hash"] != argsHash {
		err := r.Update(ctx, r.deployment(crd))
		if err != nil {
			lgr.Error(err, "error updating actor deployment")
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "actor"), &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ActorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.Actor{}).
		Complete(r)
}

func (r *ActorReconciler) labels(crd *stackv1.Actor) map[string]string {
	return map[string]string{
		"actor": crd.ObjectMeta.Name,
	}
}

func (r *ActorReconciler) argsHash(crd *stackv1.Actor) string {
	h := md5.New()
	h.Write([]byte(crd.Spec.Image))
	h.Write([]byte(crd.Spec.L1URL))
	h.Write([]byte(crd.Spec.L2URL))
	h.Write([]byte(crd.Spec.PrivateKey.String()))
	h.Write([]byte(crd.Spec.AddressManagerAddress))
	h.Write([]byte(crd.Spec.TestFilename))
	h.Write([]byte(strconv.Itoa(crd.Spec.Concurrency)))
	h.Write([]byte(strconv.Itoa(crd.Spec.RunForMS)))
	h.Write([]byte(strconv.Itoa(crd.Spec.RunCount)))
	h.Write([]byte(strconv.Itoa(crd.Spec.ThinkTimeMS)))
	return hex.EncodeToString(h.Sum(nil))
}

func (r *ActorReconciler) deployment(crd *stackv1.Actor) *appsv1.Deployment {
	replicas := int32(1)
	command := []string{
		"yarn",
		"test:actor",
		"-f",
		crd.Spec.TestFilename,
		"-c",
		strconv.Itoa(crd.Spec.Concurrency),
		"--serve",
	}
	if crd.Spec.RunForMS != 0 {
		command = append(command, "-t")
		command = append(command, strconv.Itoa(crd.Spec.RunForMS))
	} else if crd.Spec.RunCount != 0 {
		command = append(command, "-r")
		command = append(command, strconv.Itoa(crd.Spec.RunCount))
	}
	if crd.Spec.ThinkTimeMS != 0 {
		command = append(command, "--think-time")
		command = append(command, strconv.Itoa(crd.Spec.ThinkTimeMS))
	}
	env := append([]corev1.EnvVar{
		crd.Spec.PrivateKey.EnvVar("PRIVATE_KEY"),
		{
			Name:  "L1_URL",
			Value: crd.Spec.L1URL,
		},
		{
			Name:  "L2_URL",
			Value: crd.Spec.L2URL,
		},
		{
			Name:  "IS_LIVE_NETWORK",
			Value: "true",
		},
		{
			Name:  "ADDRESS_MANAGER",
			Value: crd.Spec.AddressManagerAddress,
		},
	}, crd.Spec.Env...)
	deployment := &appsv1.Deployment{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "actor", map[string]string{
			"args_hash": r.argsHash(crd),
			"actor":     crd.ObjectMeta.Name,
		}),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: r.labels(crd),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.labels(crd),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:            "actor",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							WorkingDir:      "/opt/optimism/integration-tests",
							Command:         command,
							Env:             env,
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 8545,
								},
							},
						},
					},
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, deployment, r.Scheme)
	return deployment
}

func (r *ActorReconciler) service(crd *stackv1.Actor) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "actor", r.labels(crd)),
		Spec: corev1.ServiceSpec{
			Selector: r.labels(crd),
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "metrics",
					Port: 8545,
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

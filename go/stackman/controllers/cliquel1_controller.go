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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// CliqueL1Reconciler reconciles a CliqueL1 object
type CliqueL1Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=cliquel1s/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;pods;configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CliqueL1 object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *CliqueL1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.CliqueL1{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("clique resource, not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting clique")
		return ctrl.Result{}, err
	}

	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.entrypointsCfgMap(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "clique-entrypoints"), &corev1.ConfigMap{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	nsName := ObjectNamespacedName(crd.ObjectMeta, "clique")

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.statefulSet(crd)
	}, nsName, &appsv1.StatefulSet{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, nsName, &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CliqueL1Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.CliqueL1{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *CliqueL1Reconciler) entrypointsCfgMap(crd *stackv1.CliqueL1) *corev1.ConfigMap {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique-entrypoints", map[string]string{
			"app": "clique",
		}),
		Data: map[string]string{
			"entrypoint.sh": CliqueEntrypoint,
		},
	}
	ctrl.SetControllerReference(crd, cfgMap, r.Scheme)
	return cfgMap
}

func (r *CliqueL1Reconciler) statefulSet(crd *stackv1.CliqueL1) *appsv1.StatefulSet {
	replicas := int32(1)
	labels := map[string]string{
		"app": "clique",
	}
	defaultMode := int32(0o777)
	volumes := []corev1.Volume{
		{
			Name: "entrypoints",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ObjectName(crd.ObjectMeta, "clique-entrypoints"),
					},
					DefaultMode: &defaultMode,
				},
			},
		},
	}
	var volumeClaimTemplates []corev1.PersistentVolumeClaim
	dbVolumeName := "db"
	if crd.Spec.DataPVC != nil {
		storage := resource.MustParse("128Gi")
		dbVolumeName = crd.Spec.DataPVC.Name
		if crd.Spec.DataPVC.Storage != nil {
			storage = *crd.Spec.DataPVC.Storage
		}
		volumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: crd.Spec.DataPVC.Name,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: storage,
						},
					},
				},
			},
		}
	} else {
		volumes = append(volumes, corev1.Volume{
			Name: dbVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique", labels),
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "clique",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Name:            "geth",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Command: append([]string{
								"/bin/sh",
								"/opt/entrypoints/entrypoint.sh",
							}, crd.Spec.AdditionalArgs...),
							Env: []corev1.EnvVar{
								crd.Spec.SealerPrivateKey.EnvVar("BLOCK_SIGNER_PRIVATE_KEY"),
								crd.Spec.GenesisFile.EnvVar("GENESIS_DATA"),
								{
									Name:  "BLOCK_SIGNER_ADDRESS",
									Value: crd.Spec.SealerAddress,
								},
								{
									Name:  "CHAIN_ID",
									Value: strconv.Itoa(crd.Spec.ChainID),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "entrypoints",
									MountPath: "/opt/entrypoints",
								},
								{
									Name:      dbVolumeName,
									MountPath: "/db",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8085,
								},
								{
									ContainerPort: 8086,
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
	ctrl.SetControllerReference(crd, statefulSet, r.Scheme)
	return statefulSet
}

func (r *CliqueL1Reconciler) service(crd *stackv1.CliqueL1) *corev1.Service {
	labels := map[string]string{
		"app": "clique",
	}
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "clique", labels),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "clique",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "rpc",
					Port:       8545,
					TargetPort: intstr.FromInt(8545),
				},
				{
					Name:       "ws",
					Port:       8546,
					TargetPort: intstr.FromInt(8546),
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

const CliqueEntrypoint = `
#!/bin/sh
set -exu

VERBOSITY=${VERBOSITY:-9}
GETH_DATA_DIR=/db
GETH_CHAINDATA_DIR="$GETH_DATA_DIR/geth/chaindata"
GETH_KEYSTORE_DIR="$GETH_DATA_DIR/keystore"

if [ ! -d "$GETH_KEYSTORE_DIR" ]; then
    echo "$GETH_KEYSTORE_DIR missing, running account import"
    echo -n "pwd" > "$GETH_DATA_DIR"/password
    echo -n "$BLOCK_SIGNER_PRIVATE_KEY" | sed 's/0x//' > "$GETH_DATA_DIR"/block-signer-key
    geth account import \
        --datadir="$GETH_DATA_DIR" \
        --password="$GETH_DATA_DIR"/password \
        "$GETH_DATA_DIR"/block-signer-key
else
  echo "$GETH_KEYSTORE_DIR exists."
fi

if [ ! -d "$GETH_CHAINDATA_DIR" ]; then
    echo "$GETH_CHAINDATA_DIR missing, running init"
    echo "Creating genesis file."
    echo -n "$GENESIS_DATA" > "$GETH_DATA_DIR/genesis.json"
    geth --verbosity="$VERBOSITY" init \
		--datadir="$GETH_DATA_DIR" \
		"$GETH_DATA_DIR/genesis.json"
else
  echo "$GETH_CHAINDATA_DIR exists."
fi

geth \
	--datadir="$GETH_DATA_DIR" \
	--verbosity="$VERBOSITY" \
	--http \
	--http.corsdomain="*" \
	--http.vhosts="*" \
	--http.addr=0.0.0.0 \
	--http.port=8545 \
	--ws.addr=0.0.0.0 \
	--ws.port=8546 \
	--ws.origins="*" \
	--syncmode=full \
	--nodiscover \
	--maxpeers=1 \
	--networkid=$CHAIN_ID \
	--unlock=$BLOCK_SIGNER_ADDRESS \
	--mine \
	--miner.etherbase=$BLOCK_SIGNER_ADDRESS \
	--password="$GETH_DATA_DIR"/password \
	--allow-insecure-unlock \
	"$@"
`

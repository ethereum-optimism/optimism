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
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"

	stackv1 "github.com/ethereum-optimism/optimism/go/stackman/api/v1"
)

// SequencerReconciler reconciles a Sequencer object
type SequencerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=sequencers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=sequencers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stack.optimism-stacks.net,resources=sequencers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sequencer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *SequencerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lgr := log.FromContext(ctx)

	crd := &stackv1.Sequencer{}
	if err := r.Get(ctx, req.NamespacedName, crd); err != nil {
		if errors.IsNotFound(err) {
			lgr.Info("sequencer resource not found, ignoring")
			return ctrl.Result{}, nil
		}

		lgr.Error(err, "error getting sequencer")
		return ctrl.Result{}, err
	}

	created, err := GetOrCreateResource(ctx, r, func() client.Object {
		return r.entrypointsCfgMap(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "sequencer-entrypoints"), &corev1.ConfigMap{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	statefulSet := &appsv1.StatefulSet{}
	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.statefulSet(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "sequencer"), statefulSet)
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	argsHash := r.deploymentArgsHash(crd)
	if statefulSet.Labels["args_hash"] != argsHash {
		err := r.Update(ctx, r.statefulSet(crd))
		if err != nil {
			lgr.Error(err, "error updating sequencer statefulSet")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	created, err = GetOrCreateResource(ctx, r, func() client.Object {
		return r.service(crd)
	}, ObjectNamespacedName(crd.ObjectMeta, "sequencer"), &corev1.Service{})
	if err != nil {
		return ctrl.Result{}, err
	}
	if created {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SequencerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stackv1.Sequencer{}).
		Complete(r)
}

func (r *SequencerReconciler) labels() map[string]string {
	return map[string]string{
		"app": "sequencer",
	}
}

func (r *SequencerReconciler) entrypointsCfgMap(crd *stackv1.Sequencer) *corev1.ConfigMap {
	cfgMap := &corev1.ConfigMap{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "sequencer-entrypoints", r.labels()),
		Data: map[string]string{
			"entrypoint.sh": SequencerEntrypoint,
		},
	}
	ctrl.SetControllerReference(crd, cfgMap, r.Scheme)
	return cfgMap
}

func (r *SequencerReconciler) statefulSet(crd *stackv1.Sequencer) *appsv1.StatefulSet {
	replicas := int32(1)
	defaultMode := int32(0o777)
	om := ObjectMeta(crd.ObjectMeta, "sequencer", r.labels())
	om.Labels["args_hash"] = r.deploymentArgsHash(crd)
	initContainers := []corev1.Container{
		{
			Name:            "wait-for-dtl",
			Image:           "mslipper/wait-for-it:latest",
			ImagePullPolicy: corev1.PullAlways,
			Args: []string{
				Hostify(crd.Spec.DTLURL),
				"-t",
				strconv.Itoa(crd.Spec.DTLTimeoutSeconds),
			},
		},
	}
	baseEnv := []corev1.EnvVar{
		{
			Name:  "ROLLUP_CLIENT_HTTP",
			Value: crd.Spec.DTLURL,
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
				Name:  "L2GETH_GENESIS_URL",
				Value: fmt.Sprintf("%s/state-dump.latest.json", crd.Spec.DeployerURL),
			},
			{
				Name:  "DEPLOYER_URL",
				Value: crd.Spec.DeployerURL,
			},
		}...)
	}
	volumes := []corev1.Volume{
		{
			Name: "entrypoints",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ObjectName(crd.ObjectMeta, "sequencer-entrypoints"),
					},
					DefaultMode: &defaultMode,
				},
			},
		},
	}
	var volumeClaimTemplates []corev1.PersistentVolumeClaim
	dbVolumeName := "db"
	if crd.Spec.DataPVC != nil {
		dbVolumeName = crd.Spec.DataPVC.Name
		storage := resource.MustParse("128Gi")
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
		ObjectMeta: om,
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sequencer",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.labels(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyAlways,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:            "sequencer",
							Image:           crd.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Command: append([]string{
								"/bin/sh",
								"/opt/entrypoints/entrypoint.sh",
							}, crd.Spec.AdditionalArgs...),
							Env: append(baseEnv, crd.Spec.Env...),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dbVolumeName,
									MountPath: "/db",
								},
								{
									Name:      "entrypoints",
									MountPath: "/opt/entrypoints",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8545,
								},
								{
									ContainerPort: 8546,
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

func (r *SequencerReconciler) service(crd *stackv1.Sequencer) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: ObjectMeta(crd.ObjectMeta, "sequencer", r.labels()),
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "sequencer",
			},
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "rpc",
					Port: 8545,
				},
				{
					Name: "websocket",
					Port: 8546,
				},
			},
		},
	}
	ctrl.SetControllerReference(crd, service, r.Scheme)
	return service
}

func (r *SequencerReconciler) deploymentArgsHash(crd *stackv1.Sequencer) string {
	h := md5.New()
	h.Write([]byte(crd.Spec.Image))
	h.Write([]byte(crd.Spec.L1URL))
	h.Write([]byte(strconv.Itoa(crd.Spec.L1TimeoutSeconds)))
	h.Write([]byte(crd.Spec.DeployerURL))
	h.Write([]byte(strconv.Itoa(crd.Spec.DeployerTimeoutSeconds)))
	h.Write([]byte(crd.Spec.DTLURL))
	h.Write([]byte(strconv.Itoa(crd.Spec.DTLTimeoutSeconds)))
	if crd.Spec.DataPVC != nil {
		h.Write([]byte(crd.Spec.DataPVC.Name))
		h.Write([]byte(crd.Spec.DataPVC.Storage.String()))
	}
	for _, ev := range crd.Spec.Env {
		h.Write([]byte(ev.String()))
	}
	for _, arg := range crd.Spec.AdditionalArgs {
		h.Write([]byte(arg))
	}
	return hex.EncodeToString(h.Sum(nil))
}

const SequencerEntrypoint = `
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
    echo "Retrieving genesis file $L2GETH_GENESIS_URL"
    curl --silent -o "$GETH_DATA_DIR/genesis.json" "$L2GETH_GENESIS_URL"
    geth --verbosity="$VERBOSITY" init \
		--datadir="$GETH_DATA_DIR" \
		"$GETH_DATA_DIR/genesis.json"
else
  echo "$GETH_CHAINDATA_DIR exists."
fi

geth \
  --verbosity="$VERBOSITY" \
  --datadir="$GETH_DATA_DIR" \
  --password="$GETH_DATA_DIR/password" \
  --allow-insecure-unlock \
  --unlock="$BLOCK_SIGNER_ADDRESS" \
  --mine \
  --miner.etherbase=$BLOCK_SIGNER_ADDRESS \
  "$@"
`

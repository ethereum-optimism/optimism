#!/usr/bin/env bash
#
# reth-rewind.sh — wrapper that discovers a running op-reth StatefulSet,
# generates a Job YAML for `op-wheel engine rewind-reth offline`, and applies
# it after scaling reth down.
#
# The script does NOT scale the StatefulSet back up. Verify the rewind first,
# then scale yourself:  kubectl scale sts <sts> -n <ns> --replicas=<n>
#
# Requirements: kubectl, jq.
set -euo pipefail

usage() {
  cat <<EOF
Usage: $0 -n <namespace> -t <target-block> -i <op-wheel-image> [options]

Required:
  -n NS          Kubernetes namespace containing the op-reth StatefulSet
  -t BLOCK       Target block number to rewind to
  -i IMAGE       op-wheel image that contains the 'engine rewind-reth' command
                 (must be built from a branch with the rewind-reth support —
                 the :latest tag may not have it yet)

Options:
  -s NAME        StatefulSet name (default: auto-detect via label
                 app.kubernetes.io/name=op-reth)
  -y             Skip the interactive confirmation prompt
  -k             Keep the generated Job YAML on disk (prints the path)
  -h             Show this help

Discovery is done from the live StatefulSet/pod: image tag, PVC name,
RETH_CHAIN, nodeSelector, tolerations, and (if running) current head.

Caveat — reth history pruning:
  Pruned reth nodes typically only retain ~10k blocks of account/storage
  history. 'reth stage unwind' fails with "beyond the AccountHistory limit"
  for targets older than that window. For deeper rewinds, restore from a
  snapshot (e.g. via OHM) instead.
EOF
  exit "${1:-0}"
}

NS=""
TARGET=""
OP_WHEEL_IMAGE=""
STS=""
YES=0
KEEP=0

while getopts "n:t:i:s:ykh" opt; do
  case $opt in
    n) NS="$OPTARG" ;;
    t) TARGET="$OPTARG" ;;
    i) OP_WHEEL_IMAGE="$OPTARG" ;;
    s) STS="$OPTARG" ;;
    y) YES=1 ;;
    k) KEEP=1 ;;
    h) usage 0 ;;
    *) usage 1 ;;
  esac
done

[[ -z "$NS" || -z "$TARGET" || -z "$OP_WHEEL_IMAGE" ]] && usage 1
[[ "$TARGET" =~ ^[0-9]+$ ]] || { echo "ERROR: -t must be a positive integer, got: $TARGET" >&2; exit 1; }

command -v kubectl >/dev/null || { echo "ERROR: kubectl not found on PATH" >&2; exit 1; }
command -v jq      >/dev/null || { echo "ERROR: jq not found on PATH"      >&2; exit 1; }

CTX=$(kubectl config current-context)
echo "==> kubectl context: $CTX"
echo "==> namespace:       $NS"
echo "==> target block:    $TARGET"
echo

if [[ -z "$STS" ]]; then
  STS=$(kubectl get sts -n "$NS" -l app.kubernetes.io/name=op-reth \
          -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
  [[ -z "$STS" ]] && { echo "ERROR: no op-reth StatefulSet found in $NS (try -s)" >&2; exit 1; }
fi
echo "StatefulSet:   $STS"

REPLICAS=$(kubectl get sts "$STS" -n "$NS" -o jsonpath='{.spec.replicas}')
echo "Replicas:      $REPLICAS"

RETH_CONTAINER=$(kubectl get sts "$STS" -n "$NS" \
  -o jsonpath='{.spec.template.spec.containers[0].name}')
RETH_IMAGE=$(kubectl get sts "$STS" -n "$NS" \
  -o jsonpath="{.spec.template.spec.containers[?(@.name==\"$RETH_CONTAINER\")].image}")
echo "Reth image:    $RETH_IMAGE"

PVC="datadir-${STS}-0"
kubectl get pvc "$PVC" -n "$NS" >/dev/null 2>&1 \
  || { echo "ERROR: PVC $PVC not found" >&2; exit 1; }
echo "PVC:           $PVC"

POD="${STS}-0"
POD_RUNNING=0
if kubectl get pod "$POD" -n "$NS" -o jsonpath='{.status.phase}' 2>/dev/null | grep -q Running; then
  POD_RUNNING=1
fi

RETH_CHAIN=""
if (( POD_RUNNING )); then
  RETH_CHAIN=$(kubectl exec -n "$NS" "$POD" -c "$RETH_CONTAINER" -- \
                 printenv RETH_CHAIN 2>/dev/null || true)
fi
if [[ -z "$RETH_CHAIN" ]]; then
  RETH_CHAIN=$(kubectl get sts "$STS" -n "$NS" -o json \
    | jq -r ".spec.template.spec.containers[] | select(.name==\"$RETH_CONTAINER\") | .env[]? | select(.name==\"RETH_CHAIN\") | .value // empty" \
    2>/dev/null || true)
fi
RETH_CHAIN="${RETH_CHAIN:-optimism}"
echo "RETH_CHAIN:    $RETH_CHAIN"

CURRENT_HEAD=""
if (( POD_RUNNING )); then
  HEX=$(kubectl exec -n "$NS" "$POD" -c "$RETH_CONTAINER" -- \
          curl -s -X POST http://localhost:8545 \
            -H 'content-type: application/json' \
            -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' 2>/dev/null \
        | jq -r '.result // empty' 2>/dev/null || true)
  if [[ -n "$HEX" ]]; then
    CURRENT_HEAD=$(printf "%d" "$HEX")
    echo "Current head:  $CURRENT_HEAD"
    if (( TARGET >= CURRENT_HEAD )); then
      echo "ERROR: --to ($TARGET) must be strictly less than current head ($CURRENT_HEAD)" >&2
      exit 1
    fi
  fi
fi

if (( POD_RUNNING )); then
  NODESELECTOR_JSON=$(kubectl get pod "$POD" -n "$NS" -o json | jq -c '.spec.nodeSelector // {}')
  TOLERATIONS_JSON=$(kubectl get pod "$POD" -n "$NS" -o json | jq -c '.spec.tolerations // []')
else
  NODESELECTOR_JSON=$(kubectl get sts "$STS" -n "$NS" -o json | jq -c '.spec.template.spec.nodeSelector // {}')
  TOLERATIONS_JSON=$(kubectl get sts "$STS" -n "$NS" -o json | jq -c '.spec.template.spec.tolerations // []')
fi

JOB_YAML=$(mktemp -t reth-rewind-XXXXXX).yaml
if (( ! KEEP )); then
  trap 'rm -f "$JOB_YAML"' EXIT
fi

{
  echo "apiVersion: batch/v1"
  echo "kind: Job"
  echo "metadata:"
  echo "  name: reth-rewind"
  echo "  namespace: $NS"
  echo "spec:"
  echo "  backoffLimit: 0"
  echo "  ttlSecondsAfterFinished: 86400"
  echo "  template:"
  echo "    spec:"
  echo "      restartPolicy: Never"
  echo "      terminationGracePeriodSeconds: 600"

  # YAML is a superset of JSON, so flow-style JSON is valid here.
  if [[ "$NODESELECTOR_JSON" != "{}" ]]; then
    echo "      nodeSelector: $NODESELECTOR_JSON"
  fi
  if [[ "$TOLERATIONS_JSON" != "[]" ]]; then
    echo "      tolerations: $TOLERATIONS_JSON"
  fi

  # The op-wheel image is Alpine/musl but op-reth is dynamically linked against
  # glibc — running op-reth from inside the op-wheel container fails with
  # ENOENT on its dynamic linker. op-wheel is CGO_ENABLED=0 static, so we
  # stage it into a shared volume and exec it from inside the op-reth image,
  # where the glibc runtime op-reth needs is present.
  cat <<EOF
      initContainers:
        - name: copy-wheel
          image: $OP_WHEEL_IMAGE
          command: ["cp", "/usr/local/bin/op-wheel", "/shared/op-wheel"]
          volumeMounts:
            - name: shared-bin
              mountPath: /shared
      containers:
        - name: rewind
          image: $RETH_IMAGE
          command: ["/shared/op-wheel"]
          args:
            - engine
            - rewind-reth
            - offline
            - --to
            - "$TARGET"
            - --reth-binary
            - /usr/local/bin/op-reth
            - --reth-datadir
            - /db
            - --reth-chain
            - "$RETH_CHAIN"
          resources:
            requests:
              cpu: "2"
              memory: 4Gi
            limits:
              cpu: "4"
              memory: 8Gi
          volumeMounts:
            - name: datadir
              mountPath: /db
            - name: shared-bin
              mountPath: /shared
      volumes:
        - name: datadir
          persistentVolumeClaim:
            claimName: $PVC
        - name: shared-bin
          emptyDir: {}
EOF
} > "$JOB_YAML"

echo
echo "=== Generated Job YAML ($JOB_YAML) ==="
cat "$JOB_YAML"
echo "======================================"
echo

if kubectl get job reth-rewind -n "$NS" >/dev/null 2>&1; then
  echo "NOTE: a Job named reth-rewind already exists in $NS."
  if (( YES )); then
    kubectl delete job reth-rewind -n "$NS"
  else
    read -r -p "Delete it before applying? [yes/NO] " ans
    [[ "$ans" == "yes" ]] || { echo "Aborted."; exit 1; }
    kubectl delete job reth-rewind -n "$NS"
  fi
fi

if (( ! YES )); then
  echo "This will:"
  echo "  1. scale sts/$STS in $NS from $REPLICAS to 0"
  echo "  2. apply the Job above (rewinds reth to block $TARGET)"
  echo "  3. stream logs until you Ctrl-C or the Job finishes"
  echo
  read -r -p "Proceed? [yes/NO] " ans
  [[ "$ans" == "yes" ]] || { echo "Aborted."; exit 1; }
fi

echo "==> Scaling $STS to 0 replicas"
kubectl scale sts "$STS" -n "$NS" --replicas=0
kubectl wait --for=delete "pod/$POD" -n "$NS" --timeout=300s 2>/dev/null || true

echo "==> Applying Job"
kubectl apply -f "$JOB_YAML"

echo "==> Waiting for Job pod to start (up to 5m)"
kubectl wait --for=condition=ready pod -n "$NS" -l job-name=reth-rewind --timeout=300s 2>/dev/null || true

echo "==> Streaming logs (Ctrl-C detaches; Job keeps running)"
kubectl logs -n "$NS" -l job-name=reth-rewind -c rewind -f --tail=-1 || true

echo
echo "==> Final Job status:"
kubectl get job reth-rewind -n "$NS"
echo
echo "Next steps:"
echo "  # once Job is Complete and you've verified the rewind:"
echo "  kubectl scale sts $STS -n $NS --replicas=$REPLICAS"
echo "  kubectl delete job reth-rewind -n $NS   # optional; auto-cleans in 24h"

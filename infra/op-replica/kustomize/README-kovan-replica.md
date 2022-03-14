# kovan-replica

This README covers the features used in the [kovan replica overlay](./overlays/kovan-replica-0-4-3/kustomization.yaml)

The first 2 lines are required to be a valid kustomzation target and describe how the file will be processed.

```
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
```
---

The next line is a global namespace setting. This namespace will apply to, and override, the namespace on **all** generated resources.

This makes it very safe as you know the namespace affected by a deployment! But, can't always be used if you need to deploy to multiple namespaces. In that case the namespace can be applied by a referenced kustomization.

```
namespace: kovan-replica-0-4-3
```
---

Next we list the base resources we want to include in this overlay. Since we don't need all the components, we'll select the relevant directories.

```
resources:
  - ../../bases/data-transport-layer
  - ../../bases/l2geth-replica
  - ../../bases/configmaps
  - ../../bases/servicemonitors
  - ../../bases/replica-healthcheck
```

In addition to the base resources, we also need to create some specific to this overlay.

```
 - ./l2geth-volume.yaml
```

Kustomize will add these overlay resources to the build output.

----

The next feature is a `configMapGenerator`. A generator will collect the data from the files (in this case environmental files) and create a configMap with the contents.

**Generators also apply a hash to the configMap** to track changes! You don't have to worry too much about this as the process is automatic and kustomize updates the resources that reference the configMap with the hashed name.

```
configMapGenerator:
 - name: data-transport-layer
   envs:
     - replica-data-transport-layer.env
 - name: l2geth-replica
   envs:
     - replica-l2geth.env
```

This is nice because the files can stay in their native format and allows CI to check files for validity. For example you can run `shellcheck` on scripts that get included in configMaps.

---

Next are the image replacements. You can simply identify a target image you want to replace, provide a new image name and tag. Anywhere that image is found, it's replaced with the overlay setting.

```
images:
 - name: ethereumoptimism/data-transport-layer
   newName: ethereumoptimism/data-transport-layer
   newTag: 0.4.3
 - name: ethereumoptimism/l2geth
   newName: ethereumoptimism/l2geth
   newTag: 0.4.3
```

---

Now we start patching the resources. If you just want to override something, the `patchesStrategicMerge` feature is pretty simple. In the patch file you define the path to the values you want to replace

(kustomization.yaml)
```
patchesStrategicMerge:
 - patch-l2geth-resources.yaml
```
(patch-l2geth-resources.yaml)
```
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
 name: l2geth-replica
spec:
 template:
   spec:
     containers:
       - name: l2geth-replica
         resources:
           limits:
             memory: 6Gi
           requests:
             memory: 6Gi
```

In this way we can easily update keys with brand new settings.

But, this method will not work if you want to add or replace a list item.

In these cases we define a target for the patch and a definition of the modification.

(kustomization.yaml)
```
patches:
 - path: patch-l2geth-volumes.yaml
   target:
     group: apps
     version: v1
     kind: StatefulSet
     name: l2geth-replica
```
(patch-l2geth-volumes.yaml)
```
- op: replace
 path: /spec/template/spec/volumes/2
 value:
   name: l2geth-replica-data
   persistentVolumeClaim:
     claimName: l2geth-replica-data
```

This patch acts on the StatefuleSet `l2geth-replica` and replaces the 3rd volume with a `persistentVolumeClaim`.

We define the `persistentVolumeClaim` as a local resource in `l2geth-volume.yaml`

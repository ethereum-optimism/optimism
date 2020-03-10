## Unreleased

BREAKING CHANGES:

* `consul-k8s` `v0.12.0`+ is now required. The chart is passing new flags that are only available in this version.
  To use this version if not using the chart defaults, set
  ```yaml
  global:
    imageK8S: hashicorp/consul-k8s:0.12.0
  ```

IMPROVEMENTS:

* Catalog Sync
  * New Helm values have been added to configure which Kubernetes namespaces we will sync from. The defaults are shown below:
    ```yaml
    syncCatalog:
      toConsul: true
      k8sAllowNamespaces: ["*"]
      k8sDenyNamespaces: ["kube-system", "kube-public"]
    ```
  * If running Consul Enterprise 1.7.0+, Consul namespaces are supported. New Helm values have been added to allow configuring which
    Consul namespaces Kubernetes services are synced to. See [https://www.consul.io/docs/platform/k8s/service-sync.html#consul-enterprise-namespaces](https://www.consul.io/docs/platform/k8s/service-sync.html#consul-enterprise-namespaces) for more details.

    ```yaml
    global:
      enableConsulNamespaces: true
    syncCatalog:
      consulNamespaces:
        # consulDestinationNamespace is the name of the Consul namespace to register all
        # k8s services into. If the Consul namespace does not already exist,
        # it will be created. This will be ignored if `mirroringK8S` is true.
        consulDestinationNamespace: "default"

        # mirroringK8S causes k8s services to be registered into a Consul namespace
        # of the same name as their k8s namespace, optionally prefixed if
        # `mirroringK8SPrefix` is set below. If the Consul namespace does not
        # already exist, it will be created. Turning this on overrides the
        # `consulDestinationNamespace` setting.
        # `addK8SNamespaceSuffix` may no longer be needed if enabling this option.
        mirroringK8S: false

        # If `mirroringK8S` is set to true, `mirroringK8SPrefix` allows each Consul namespace
        # to be given a prefix. For example, if `mirroringK8SPrefix` is set to "k8s-", a
        # service in the k8s `staging` namespace will be registered into the
        # `k8s-staging` Consul namespace.
        mirroringK8SPrefix: ""
    ```

* Connect Inject
  * New Helm values have been added to configure which Kubernetes namespaces we will inject pods in. The defaults are shown below:
    ```yaml
    connectInject:
      k8sAllowNamespaces: ["*"]
      k8sDenyNamespaces: []
    ```
  * If running Consul Enterprise 1.7.0+, Consul namespaces are supported. New Helm values have been added to allow configuring which Consul namespaces Kubernetes pods
    are registered into. See [https://www.consul.io/docs/platform/k8s/connect.html#consul-enterprise-namespaces](https://www.consul.io/docs/platform/k8s/connect.html#consul-enterprise-namespaces) for more details.
    ```yaml
    global:
      enableConsulNamespaces: true

    connectInject:
      consulNamespaces:
        # consulDestinationNamespace is the name of the Consul namespace to register all
        # k8s pods into. If the Consul namespace does not already exist,
        # it will be created. This will be ignored if `mirroringK8S` is true.
        consulDestinationNamespace: "default"

        # mirroringK8S causes k8s pods to be registered into a Consul namespace
        # of the same name as their k8s namespace, optionally prefixed if
        # `mirroringK8SPrefix` is set below. If the Consul namespace does not
        # already exist, it will be created. Turning this on overrides the
        # `consulDestinationNamespace` setting.
        mirroringK8S: false

        # If `mirroringK8S` is set to true, `mirroringK8SPrefix` allows each Consul namespace
        # to be given a prefix. For example, if `mirroringK8SPrefix` is set to "k8s-", a
        # pod in the k8s `staging` namespace will be registered into the
        # `k8s-staging` Consul namespace.
        mirroringK8SPrefix: ""
    ```

BUG FIXES:

* Fix template rendering bug when setting `connectInject.overrideAuthMethodName` [[GH-342](https://github.com/hashicorp/consul-helm/pull/342)]
* Set `"consul.hashicorp.com/connect-inject": "false"` annotation on enterprise license job so it is not connect injected [[GH-343](https://github.com/hashicorp/consul-helm/pull/343)]

DEPRECATIONS:

* `.syncCatalog.k8sSourceNamespace` should no longer be used. Instead, use the new `.syncCatalog.k8sAllowNamespaces` and `.syncCatalog.k8sDenyNamespaces` features. For backward compatibility, if both this and the allow/deny lists are set, the allow/deny lists will be ignored.

NOTES:

* Bootstrap ACLs: Previously, ACL policies were not updated after creation. Now, if namespaces are enabled, they are updated every time the ACL bootstrapper is run so that any namespace config changes can be adjusted. This change is only an issue if you are updating ACL policies after creation.

## 0.16.2 (Jan 15, 2020)

BUG FIXES:

  * Fix Helm Chart version.

## 0.16.1 (Jan 14, 2020)

BUG FIXES:

  * Fix a bug with the `tls-init` job, in which it could not correctly detect CA file
    if Consul domain is provided [[GH-329](https://github.com/hashicorp/consul-helm/pull/329)].

## 0.16.0 (Jan 10, 2020)

IMPROVEMENTS:

  * Optionally allow enabling TLS for Consul communication [[GH-313](https://github.com/hashicorp/consul-helm/pull/313)].
    If `global.tls.enabled` is set to `true`, the Helm chart will generate a CA and necessary certificates and
    enable TLS for servers, clients, Connect injector, Mesh gateways, catalog sync, ACL bootstrapping, and snapshot agents.

    Note that this feature is only supported if both servers and clients are running
    on Kubernetes. We will have better support for other deployment architectures,
    as well as bringing your own CA, in the future.

    Also, note that simply turning on this feature and running `helm upgrade` will result in downtime if you are using
    Consul Connect or Sync Catalog features. We will be adding instructions on how to do this upgrade without downtime soon.
    Additionally, if you do decide to proceed with an upgrade despite downtime
    and you're using Consul Connect, all application pods need to be recreated after upgrade, so that the Connect injector
    can re-inject Envoy sidecars with TLS enabled.

  * Use the latest version of consul-k8s (0.11.0).

  * Add pod name as metadata to client nodes to help users map nodes in Consul to underlying client pods
    [[GH-315](https://github.com/hashicorp/consul-helm/pull/315)].

  * Rename `enterprise-licence.yaml` template to `enterprise-license-job.yaml` [[GH-321](https://github.com/hashicorp/consul-helm/pull/321)].

BUG FIXES:

  * Fix graceful termination for servers [[GH-313](https://github.com/hashicorp/consul-helm/pull/313)].
    `terminationGracePeriod` is now set to 30 seconds for the servers. The previous setting of 10 seconds
    wasn't always enough time for a graceful leave, and in those cases, servers leave the cluster
    in a "failed" state. Additionally, clients always set `leave_on_terminate` to `true`.
    This replaces the `preStop` hook that was calling `consul leave`. Note that `leave_on_terminate` defaults
    to true for clients as of Consul `0.7`, so this change only affects earlier versions.

  * Helm test runner now respects the provided namespace [[GH-320](https://github.com/hashicorp/consul-helm/pull/320)].

  * Add pod security policies for the `enterprise-license` [[GH-325](https://github.com/hashicorp/consul-helm/pull/325)]
    and the `server-acl-init` jobs [[GH-326](https://github.com/hashicorp/consul-helm/pull/325)].

## 0.15.0 (Dec 17, 2019)

BREAKING CHANGES:

  * `connectInject.centralConfig` defaults to `true` now instead of `false`. This is to make it
     easier to configure Connect via `service-defaults` and other routing
     config [[GH-302](https://github.com/hashicorp/consul-helm/pull/302)].
     See https://www.consul.io/docs/agent/options.html#enable_central_service_config.

     If you wish to disable central config, set `connectInject.centralConfig` to
     false in your local values file. NOTE: If `connectInject.enabled` is false,
     then central config is not enabled so this change will not affect you. 
  
  * Connect Inject: If using Connect Inject, you must also upgrade your `consul-k8s` version
    to a version >= 0.10.1. A new flag is being passed in to `consul-k8s` which is not
    supported in earlier versions.

BUG FIXES:
  * Fix bug with `fullnameOverride` and add new `global.name` setting for changing
    the default prefix for resources. [[GH-286](https://github.com/hashicorp/consul-helm/issues/286)]

  * Connect Inject: Fix critical bug where Connect-registered services instances would be de-registered
    when the Consul client on the same node was restarted. This fix adds a new
    sidecar that ensures the service instance is always registered. [[GH-314](https://github.com/hashicorp/consul-helm/pull/314)]

## 0.14.0 (Dec 10, 2019)

IMPROVEMENTS:

  * Consul client DaemonSet can now use a [hostPath mount](https://kubernetes.io/docs/concepts/storage/volumes/#hostpath)
    for its data directory by setting the `client.dataDirectoryHostPath` value.
    This setting is currently necessary to ensure that when a Consul client Pod is deleted,
    e.g. during a Consul version upgrade, it does not lose its Connect service
    registrations. In the next version, we plan to have services automatically
    re-register which will remove the need for this. [[GH-298](https://github.com/hashicorp/consul-helm/pull/298)]
    (**Update:** 0.15.0 uses a version of consul-k8s that fixes this bug and so hostPath is longer necessary)
    
    **Security Warning:** If using this setting, Pod Security Policies *must* be enabled on your cluster
     and in this Helm chart (via the `global.enablePodSecurityPolicies` setting)
     to prevent other Pods from mounting the same host path and gaining
     access to all of Consul's data. Consul's data is not encrypted at rest.

  * New configuration option `client.updateStrategy` allows setting the update
    strategy for the Client DaemonSet. [[GH-298](https://github.com/hashicorp/consul-helm/pull/298)]

  * New configuration option `client.dnsPolicy` allows setting the DNS
    policy for the Client DaemonSet. [[GH-298](https://github.com/hashicorp/consul-helm/pull/298)]

## 0.13.0 (Dec 5, 2019)

BREAKING CHANGES:

  * `client.grpc` defaults to `true` now instead of `false`. This is to make it
    harder to misconfigure Connect. [[GH-282](https://github.com/hashicorp/consul-helm/pull/282)]
     
    If you do not wish to enable gRPC for clients, set `client.grpc` to
    `false` in your local values file.

  * Add `syncCatalog.addK8SNamespaceSuffix` and default it to `true`. [[GH-280](https://github.com/hashicorp/consul-helm/pull/280)]
    Note: upgrading an existing installation will result in deregistering
    of existing synced services in Consul and registering them with a new name.
    If you would like to avoid this behavior set `syncCatalog.addK8SNamespaceSuffix`
    to `false`.
    
    This changes the default service names registered from Kubernetes into Consul. Previously, we would register all Kubernetes services, regardless of namespace, as the same service in Consul. After this change, the default behaviour is to append the Kubernetes namespace to the Consul service name. For example, given a Kubernetes service `foo` in the namespace `namespace`, it would be registered in Consul as `foo-namespace`. The name can also be controlled via the `consul.hashicorp.com/service-name` annotation.

IMPROVEMENTS:

  * Use the latest version of consul (1.6.2)
  * Use the latest version of consul-k8s (0.9.5)
  * Add `connectInject.overrideAuthMethodName` to allow setting the `-acl-auth-method flag` [[GH-278](https://github.com/hashicorp/consul-helm/pull/278)]
  * Support external to k8s Consul servers [[GH-289](https://github.com/hashicorp/consul-helm/pull/289)]

BUG FIXES:

  * Do not run `server-acl-init` during server rollout [[GH-292](https://github.com/hashicorp/consul-helm/pull/292)]

## 0.12.0 (Oct 28, 2019)

IMPROVEMENTS:

  * Use the latest version of consul-k8s (0.9.4)
  * Support `bootstrapACLs` when only servers are enabled (not clients) [[GH-250](https://github.com/hashicorp/consul-helm/pull/250)]
  * Use less privileges for catalog sync when not syncing to k8s [[GH-248](https://github.com/hashicorp/consul-helm/pull/248)]
  * Enable disabling tests for users using `helm template` [[GH-249](https://github.com/hashicorp/consul-helm/pull/249)]

BUG FIXES:

  * Fix `missing required field "caBundle"` bug [[GH-213](https://github.com/hashicorp/consul-helm/issues/213)]


## 0.11.0 (Oct 15, 2019)

IMPROVEMENTS:

  * Use the latest version of Consul (1.6.1)

BUG FIXES:

  * Use the latest version of `consul-k8s` (0.9.3) which fixes issues with upgrading between Helm chart
    versions when `bootstrapACLs` is enabled [[GH-246](https://github.com/hashicorp/consul-helm/pull/246)].
  * Add `server-acl-init-cleanup` job to clean up the `server-acl-init` job
    when it completes successfully [[GH-246](https://github.com/hashicorp/consul-helm/pull/246)].
  * Add the ability to specify Consul client daemonset affinity [[GH-165](https://github.com/hashicorp/consul-helm/pull/165)]

## 0.10.0 (Oct 4, 2019)

IMPROVEMENTS:

  * Use latest version of Consul (1.6.0) and consul-k8s (0.9.2)
  * Remove random value from `helm test` to enable helmfile use [[GH-143](https://github.com/hashicorp/consul-helm/pull/143)]
  
BUG FIXES:
  
  * The latest version of `consul-k8s` fixes issues with the `server-acl-init`
    job failing repeatedly.

## 0.9.0 (Sep 6, 2019)

IMPROVEMENTS:

  * Support running the consul snapshot agent
  * Support mesh gateways
  * Allow setting annotations for the DNS service
  * Allow setting `-consul-write-interval`, `-log-level` and `-k8s-source-namespace` flags for consul-k8s sync
  * Allow setting DNS service IP
  * Fix issues where acl-init job would fail repeatedly and ACLs would not be
    bootstrapped

BUG FIXES:

  * Fix enterprise license application when ACLs are turned off
  * `rules` key must always be set (fixes https://github.com/hashicorp/consul-helm/issues/178)

## 0.8.1 (May 9, 2019)

IMPROVEMENTS:

  * Update default consul-k8s version to 0.8.1 for a central config bug fix

## 0.8.0 (May 8, 2019)

IMPROVEMENTS:

  * Support adding a prefix to Kubernetes services registered in Consul [[GH 140](https://github.com/hashicorp/consul-helm/issues/140)]
  * Support an option for automatically bootstrapping ACLs in a Consul cluster that is run fully in Kubernetes. If connectInject is enabled with this option on, this also automatically configures a new Kubernetes AuthMethod so that injected services are automatically granted ACL tokens based on their Kubernetes service account.
  * Support central service configuration including proxy defaults in Connect (available in Consul 1.5+).
  * Remove the `gossipEncryption.enabled` option and instead have the implementation based on the existence of the secretName and secretKey.

## 0.7.0 (March 21, 2019)

BREAKING CHANGES:

  * If previously setting the release name to `consul`, you must now set `fullnameOverride: consul` in your config to prevent all resources being renamed.

IMPROVEMENTS:

  * Support pod PriorityClasses for Consul servers and clients
  * Add annotation and additional spec values for the UI service
  * Add liveness and readiness checks to the catalog sync pod [[consul-k8s GH 57](https://github.com/hashicorp/consul-k8s/issues/57)]
  * Support custom annotations for Consul clients and servers
  * Support PodSecurityPolicies for Consul components
  * Add service accounts and cluster roles/role bindings for each Consul component
  * Add the namespace to the metadata volume name
  * Support tolerations on Consul client and server pods
  * Support gossip protocol encryption
  * Allows custom environment variables for Consul client and server pods
  * Support nodeSelectors for all components
  
BUG FIXES:

  * Allow setting `extraConfig` variables using Helm's `--set` flag [[GH 74](https://github.com/hashicorp/consul-helm/issues/74)]
  * Fix a formatting bug in the enterprise license command

## 0.6.0 (February 8, 2019)

IMPROVEMENTS:

  * Supports applying a Consul Enterprise License to the cluster through the Helm chart
  * Support assigning an ACL token to the catalog sync process [[GH 26](https://github.com/hashicorp/consul-k8s/issues/26)]
  * Updates default `consul` version to `1.4.2` and `consul-k8s` version to `0.5.0`
  
BUG FIXES:

  * Switch the chart labels to a non-changing value to allow helm upgrades [[GH 86](https://github.com/hashicorp/consul-helm/issues/86)]
  
## 0.5.0 (January 11, 2019)

IMPROVEMENTS:

  * Supports new NodePort syncing style that uses the node ip address
  * Adds a configurable tab to the Kubernetes -> Consul sync

## 0.4.0 (December 7, 2018)

IMPROVEMENTS:

  * RBAC support for `syncCatalog`. This will create the `ClusterRole`, `ClusterRoleBinding`
    and `ServiceAccount` that is necessary for the catalog sync. [[GH-20](https://github.com/hashicorp/consul-helm/issues/20)]
  * client: agents now have the node name set to the actual K8S node name [[GH-14](https://github.com/hashicorp/consul-helm/issues/14)]
  * RBAC support for `connectInject`. This will create a `ClusterRole`, `ClusterRoleBinding`,
    and `ServiceAccount` that is necessary for the connect injector to automatically generate
    TLS certificates to interact with the Kubernetes API.
  * Server affinity is now configurable. This makes it easier to run an entire
    Consul cluster on Minikube. [[GH-13](https://github.com/hashicorp/consul-helm/issues/13)]
  * Liveness probes are now http calls, reducing errors in the logs.
  * All namespaced resources now specify the namespace metadata, making `helm template` usage in 
    a non-default namespace easier. [[GH-66](https://github.com/hashicorp/consul-helm/issues/66)]
  * Add support for ClusterIP service syncing.

BUG FIXES:

  * Add catalog sync default behavior flag to the chart [GH-28]
  * Updated images to point to latest versions for 0.3.0.
  * Add missing continuation characters to long commands [[GH-26](https://github.com/hashicorp/consul-helm/issues/26)].
  * connectInject: set the correct namespace for the MutatingWebhookConfiguration
    so that deployments work in non-default namespaces. [[GH-38](https://github.com/hashicorp/consul-helm/issues/38)]
  * Provide a valid `maxUnavailable` value when replicas=1. [[GH-58](https://github.com/hashicorp/consul-helm/issues/58)]
  * Correctly sets server resource requirements.
  * Update the `maxUnavailable` default calculation to allow rolling updates on 3 server clusters. [[GH-71](https://github.com/hashicorp/consul-helm/issues/71)]

## 0.3.0 (October 11, 2018)

FEATURES:

  * `connectInject` can install the automatic Connect sidecar injector.

## 0.2.0 (September 26, 2018)

FEATURES:

  * `syncCatalog` can install the [service catalog sync](https://www.hashicorp.com/blog/consul-and-kubernetes-service-catalog-sync)
    functionality.

IMPROVEMENTS:

  * server: support `storageClass` [[GH-7](https://github.com/hashicorp/consul-helm/issues/7)]

## 0.1.0

Initial release

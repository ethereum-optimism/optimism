## Unreleased

Features:
* Added `volumes` and `volumeMounts` for mounting _any_ type of volume [GH-314](https://github.com/hashicorp/vault-helm/pull/314).

Improvements:
* Added `defaultMode` configurable to `extraVolumes`[GH-321](https://github.com/hashicorp/vault-helm/pull/321)
* Option to install and use PodSecurityPolicy's for vault server and injector [GH-177](https://github.com/hashicorp/vault-helm/pull/177)
* `VAULT_API_ADDR` is now configurable [GH-290](https://github.com/hashicorp/vault-helm/pull/290)
* Removed deprecated tolerate unready endpoint annotations [GH-363](https://github.com/hashicorp/vault-helm/pull/363)

Bugs:
* Fix python dependency in test image [GH-337](https://github.com/hashicorp/vault-helm/pull/337)
* Fix caBundle not being quoted causing validation issues with Helm 3 [GH-352](https://github.com/hashicorp/vault-helm/pull/352)
* Fix injector network policy being rendered when injector is not enabled [GH-358](https://github.com/hashicorp/vault-helm/pull/358)

## 0.6.0 (June 3rd, 2020)

Features:
* Added `extraInitContainers` to define init containers for the Vault cluster [GH-258](https://github.com/hashicorp/vault-helm/pull/258)
* Added `postStart` lifecycle hook allowing users to configure commands to run on the Vault pods after they're ready [GH-315](https://github.com/hashicorp/vault-helm/pull/315)
* Beta: Added OpenShift support [GH-319](https://github.com/hashicorp/vault-helm/pull/319)

Improvements:
* Server configs can now be defined in YAML.  Multi-line string configs are still compatible [GH-213](https://github.com/hashicorp/vault-helm/pull/213)
* Removed IPC_LOCK privileges since swap is disabled on containers [[GH-198](https://github.com/hashicorp/vault-helm/pull/198)]
* Use port names that map to vault.scheme [[GH-223](https://github.com/hashicorp/vault-helm/pull/223)]
* Allow both yaml and multi-line string annotations [[GH-272](https://github.com/hashicorp/vault-helm/pull/272)]
* Added configurable to set the Raft node name to hostname [[GH-269](https://github.com/hashicorp/vault-helm/pull/269)]
* Support setting priorityClassName on pods [[GH-282](https://github.com/hashicorp/vault-helm/pull/282)]
* Added support for ingress apiVersion `networking.k8s.io/v1beta1` [[GH-310](https://github.com/hashicorp/vault-helm/pull/310)]
* Added configurable to change service type for the HA active service [GH-317](https://github.com/hashicorp/vault-helm/pull/317)

Bugs:
* Fixed default ingress path [[GH-224](https://github.com/hashicorp/vault-helm/pull/224)]
* Fixed annotations for HA standby/active services [[GH-268](https://github.com/hashicorp/vault-helm/pull/268)]
* Updated some value defaults to match their use in templates [[GH-309](https://github.com/hashicorp/vault-helm/pull/309)]
* Use active service on ingress when ha [[GH-270](https://github.com/hashicorp/vault-helm/pull/270)]
* Fixed bug where pull secrets weren't being used for injector image [GH-298](https://github.com/hashicorp/vault-helm/pull/298)

## 0.5.0 (April 9th, 2020)

Features:

* Added Raft support for HA mode [[GH-228](https://github.com/hashicorp/vault-helm/pull/229)]
* Now supports Vault Enterprise [[GH-250](https://github.com/hashicorp/vault-helm/pull/250)]
* Added K8s Service Registration for HA modes [[GH-250](https://github.com/hashicorp/vault-helm/pull/250)]

* Option to set `AGENT_INJECT_VAULT_AUTH_PATH` for the injector [[GH-185](https://github.com/hashicorp/vault-helm/pull/185)]
* Added environment variables for logging and revocation on Vault Agent Injector [[GH-219](https://github.com/hashicorp/vault-helm/pull/219)]
* Option to set environment variables for the injector deployment [[GH-232](https://github.com/hashicorp/vault-helm/pull/232)]
* Added affinity, tolerations, and nodeSelector options for the injector deployment [[GH-234](https://github.com/hashicorp/vault-helm/pull/234)]
* Made all annotations multi-line strings [[GH-227](https://github.com/hashicorp/vault-helm/pull/227)]

## 0.4.0 (February 21st, 2020)

Improvements:

* Allow process namespace sharing between Vault and sidecar containers [[GH-174](https://github.com/hashicorp/vault-helm/pull/174)]
* Added configurable to change updateStrategy [[GH-172](https://github.com/hashicorp/vault-helm/pull/172)]
* Added sleep in the preStop lifecycle step [[GH-188](https://github.com/hashicorp/vault-helm/pull/188)]
* Updated chart and tests to Helm 3 [[GH-195](https://github.com/hashicorp/vault-helm/pull/195)]
* Adds Values.injector.externalVaultAddr to use the injector with an external vault [[GH-207](https://github.com/hashicorp/vault-helm/pull/207)]

Bugs:

* Fix bug where Vault lifecycle was appended after extra containers. [[GH-179](https://github.com/hashicorp/vault-helm/pull/179)]

## 0.3.3 (January 14th, 2020)

Security:

* Added `server.extraArgs` to allow loading of additional Vault configurations containing sensitive settings [GH-175](https://github.com/hashicorp/vault-helm/issues/175)

Bugs:

* Fixed injection bug where wrong environment variables were being used for manually mounted TLS files

## 0.3.2 (January 8th, 2020)

Bugs:

* Fixed injection bug where TLS Skip Verify was true by default [VK8S-35]

## 0.3.1 (January 2nd, 2020)

Bugs:

* Fixed injection bug causing kube-system pods to be rejected [VK8S-14]

## 0.3.0 (December 19th, 2019)

Features:

* Extra containers can now be added to the Vault pods
* Added configurability of pod probes
* Added Vault Agent Injector 

Improvements:

* Moved `global.image` to `server.image`
* Changed UI service template to route pods that aren't ready via `publishNotReadyAddresses: true`
* Added better HTTP/HTTPS scheme support to http probes
* Added configurable node port for Vault service
* `server.authDelegator` is now enabled by default

Bugs:

* Fixed upgrade bug by removing chart label which contained the version
* Fixed typo on `serviceAccount` (was `serviceaccount`)
* Fixed readiness/liveliness HTTP probe default to accept standbys

## 0.2.1 (November 12th, 2019)

Bugs:

* Removed `readOnlyRootFilesystem` causing issues when validating deployments

## 0.2.0 (October 29th, 2019)

Features:

* Added load balancer support
* Added ingress support
* Added configurable for service types (ClusterIP, NodePort, LoadBalancer, etc)
* Removed root requirements, now runs as Vault user

Improvements:

* Added namespace value to all rendered objects
* Made ports configurable in services
* Added the ability to add custom annotations to services
* Added docker image for running bats test in CircleCI
* Removed restrictions around `dev` mode such as annotations
* `readOnlyRootFilesystem` is now configurable
* Image Pull Policy is now configurable

Bugs:

* Fixed selector bugs related to Helm label updates (services, affinities, and pod disruption)
* Fixed bug where audit storage was not being mounted in HA mode
* Fixed bug where Vault pod wasn't receiving SIGTERM signals


## 0.1.2 (August 22nd, 2019)

Features:

* Added `extraSecretEnvironmentVars` to allow users to mount secrets as
  environment variables
* Added `tlsDisable` configurable to change HTTP protocols from HTTP/HTTPS 
  depending on the value
* Added `serviceNodePort` to configure a NodePort value when setting `serviceType` 
  to "NodePort"

Improvements:

* Changed UI port to 8200 for better HTTP protocol support
* Added `path` to `extraVolumes` to define where the volume should be 
  mounted.  Defaults to `/vault/userconfig`
* Upgraded Vault to 1.2.2

Bugs:

* Fixed bug where upgrade would fail because immutable labels were being 
  changed (Helm Version label)
* Fixed bug where UI service used wrong selector after updating helm labels
* Added `VAULT_API_ADDR` env to Vault pod to fixed bug where Vault thinks
  Consul is the active node
* Removed `step-down` preStop since it requires authentication.  Shutdown signal
  sent by Kube acts similar to `step-down`


## 0.1.1 (August 7th, 2019)

Features:

* Added `authDelegator` Cluster Role Binding to Vault service account for
  bootstrapping Kube auth method

Improvements:

* Added `server.service.clusterIP` to `values.yml` so users can toggle
  the Vault service to headless by using the value `None`.
* Upgraded Vault to 1.2.1

## 0.1.0 (August 6th, 2019)

Initial release

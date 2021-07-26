*********************************
Vault Design Q&A
*********************************

Here we will discuss and capture requirements for the Vault cluster architecture and plugin design.

Plugin Design and Implementation
#################################

Authority discussion
*****************************

I'd like to discuss the workflow for establishing an Authority. A possible workflow is:

Vault administrator creates the Authority wallet:
::

	vault write -f immutability-eth-plugin/wallets/authority

Vault administrator creates an Authority account - this will be the account that is used to identify the Authority.

::

	AUTHORITY_ACCOUNT=$(vault write -f -field=address immutability-eth-plugin/wallets/authority/accounts)

Maintainer sets authority using out-of-band contract call with $AUTHORITY_ACCOUNT

After Authority is set, then the following API can be used to submit blocks:

::

	vault write immutability-eth-plugin/wallets/authority/accounts/$AUTHORITY_ACCOUNT/submitBlock block_number=$BLOCK_NUMBER


Vault Cluster Design
#################################


Topology
*****************************

* What is your preferred deployment?

   * GKE

* Do you have any multi-region or HA/DR requirements?

    * Currently storage is held in block storage on GCP.  Can't move horizontally, single region.
    * After first launch will move to postgres and will have multi-region.
    * Design will focus on a single region with information on how to move to multi-region.
    * Multi-region will most likely be hot-warm.

* What about load balancers and whitelisting of public and/or private Endpoints?  Our recommendation is to only allow internal access to the Vault cluster.

    * No external access is fine.  VPN preferred.
    * We need to include the VPN in the design.  VPN would only allow access to a single other (prod mainnet) cluster.

* How highly available should it be?  We recommend running in 3 zones generally.

   * 2 or 3.

* How many environments will you run?

    * multiple dev, one staging, and one production

* Do you have a preferred storage backend for Vault?  We recommend GCS: https://www.vaultproject.io/docs/configuration/storage/google-cloud-storage.html

    * Preference is Consul since it is memory based.  Perhaps raft will be a good fit here since it will provide similar features with less overhead.

Security
*****************************

* How do you currently handle user access in GCP?

    * Historically very little access.  Has changed recently.  Prod access only 4 people have access.
    * Vault cluster - new google project.  Access is restricted to a few laptops that are locked away.
    * Using google apps.

* How do you provision certificates today? How will we get certificates for Vault in the pipeline?

    * Edge - cloudflare.  Origin letsencrypt.

* How will users access the Vault cluster?  Should we implement OIDC or is some other authn/z used?

    * Users will need to be able to import/export key material.
    * k8s service accounts for vault application/api access.
    * We'll drill into policy in another meeting.
    * Today RBAC in k8s, google login for IAM.

* How important is security?  For example:

    * Do you want us to CIS harden everything?

        * Yes

    * Do you want us to do penetration testing?

        * We'll talk w/ OM security on this.

* Unsealing

    * Leaning towards manual.  Will talk to Kasima to get a business decision.

* Certificates

    * LE for now.  This may change.  Means < 90 day rolling updates

* Access to Vault from apps

    * Initially GCP service accounts


Administration & Maintenance
*****************************

* What are your SLA requirements?  For example, can your applications tolerate an outage of a few seconds when deploying?

    * None today.  Currently can handle outage of a few seconds.

* How often will the cluster, vault, and plugin be updated?

    * regular sec updates - ~ 1/month

Other Details
*****************************

* How many transactions per second should Vault handle?

    * 1/second at most.  Throttled by ethereum mainnet.

* How do you handle logging/log storage?

    * Use datadog for logs and log storage.

* What about monitoring?  You mentioned you would like to explore Prometheus instead of DataDog?

    * Datadog for initial design.  Include some prometheus discussion/details.

* What is your backup strategy?

    * We'll discuss offline and will discuss with security in a subsequent meeting.

* Do you have any requirements for config-as-code?   For example: Terraform/Ansible/Chef/Salt/bash?

    * Terraform and Helm.



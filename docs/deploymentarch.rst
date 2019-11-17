*********************************
Deployment Architecture
*********************************

Immutability's recommendation for Vault deployments is based on a number of factors which we will discuss in this section of the documentation.

Regardless of a client's risk tolerance we always recommend that Vault run in a separately provisioned and cordoned off network.  Using a walled garden approach with Vault significantly increases security and makes audits simpler as you only need be concerned with Vault, allowed traffic to Vault, and any monitoring solution.  This reduces the probability that mistakes will be made when doing things like opening ports, installing libraries, etc.

Note that the diagrams throughout this document will show a single instantiation of a Vault cluster deployment.  It is expected there will be multiple instantiations for dev/stage/prod/etc.

The Network
#################################

Our general recommendation is that Vault run in its own network with the only allowed traffic being ingress over a layer-4 load balancer.  Egress is allowed for certain operations but is tightly controlled.  Depending upon the requirements, risk tolerance of the client, and other factors we may recommend using or not using a cloud provider.

Base Network
*****************************

We show below what the base network should look like.  Access is allowed from same-level environments and some other trusted network which will be used for deployments.

Note that below it is assumed that only same-level environments are connected.  For example, only prod environments should be linked from prod OmiseGO GCP projects to the Vault GCP prod project.

In the case of GCP, we recommend a private network (VPC) be used with a VPN gateway and tunnels for access from external networks.  Note the image below assumes that Vault runs in GKE however running on virtual instances is similar.

.. image:: _static/omisego-arch-network.png
  :width: 1200
  :alt: Base GCP Network

There will most likely need to be 3 VPN gateways.  2 for GCP->GCP HA and 1 for administration:

https://cloud.google.com/vpn/docs/how-to/choosing-a-vpn

https://cloud.google.com/vpn/docs/how-to/creating-ha-vpn2

Ingress/Egress on GKE
*****************************

Generally speaking no ingress should be allowed except over a private load balancer and between nodes in the vault pod and control plane (master).  Egress should be tightly locked down.  Defaults are bad.

At a high level we need to:

1.  Create a VPC and subnets for a GKE cluster with private access

2.  Lock down VPC with firewall rules

      - block egress to 0.0.0.0/0
      - allow ingress from Google health checks
      - allow egress to Google health checks, restricted APIs, and GKE private master CIDRs.

3.  Remove default route automatically created in VPC (0.0.0.0/0 with default internet gateway as next hop).

4.  Create route to reach Google restricted APIs (199.36.153.4/30) through a default internet gateway

5.  Make the Cloud DNS changes and attach the zones to the VPC:

      - Create private DNS zone googleapis.com with a CNAME record to restricted.googleapis.com for '*'.googleapis.com and A record to 199.36.153.4/30 for restricted.googleapis.com
      - Create private DNS zone gcr.io with a CNAME record to gcr.io for '*'.gcr.io and A record to 199.36.153.4/30 for a blank gcr.io DNS name

A high level overview of creating a private GKE cluster is here:

https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters

Similar changes will need to be made if running on virtual instances instead of GKE.


# Install the Consul Helm chart with value overrides
# This depends on the Consul gossip key existing in K8s secrets
# prior to attempting to install the Helm chart
resource "helm_release" "consul_base" {
  depends_on = [kubernetes_secret.consul_gossip_key]

  name  = "consul-backend"
  chart = "../helm/consul-backend"

  cleanup_on_fail = true

  set {
    name  = "global.enabled"
    value = false
  }

  set {
    name  = "global.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "global.gossipEncryption.secretName"
    value = var.consul_gossip_key_name
  }

  set {
    name  = "global.gossipEncryption.secretKey"
    value = "key"
  }

  set {
    name  = "global.bootstrapACLs"
    value = true
  }

  set {
    name  = "global.tls.enabled"
    value = true
  }

  set {
    name  = "global.tls.verify"
    value = true
  }

  set {
    name  = "consul.server.enabled"
    value = true
  }

  set {
    name  = "consul.server.replicas"
    value = var.consul_replicas
  }

  set {
    name  = "consul.server.bootstrapExpect"
    value = var.consul_bootstrap_expect
  }

  set {
    name  = "consul.server.connect"
    value = false
  }

  set {
    name  = "consul.server.affinity"
    value = <<EOF
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 1
          podAffinityTerm:
            topologyKey: kubernetes.io/hostname
            labelSelector:
              matchExpressions:
              - key: component
                operator: In
                values:
                - "{{ .Release.Name }}-{{ .Values.Component }}"
    EOF
  }

  set {
    name  = "consul.client.enabled"
    value = true
  }
}

#!/usr/bin/env sh

sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $HOME/etc/vault.unsealer/ca.pem
CERT_DIR=$(minikube ssh "route -n | grep ^0.0.0.0 | awk '{ print \$2 }'" | tr -d '\r')
scp  -i $(minikube ssh-key) $HOME/etc/vault.unsealer/ca.pem docker@$(minikube ip):/home/docker/
minikube ssh "sudo mkdir -p /etc/docker/certs.d/$CERT_DIR:5000"
minikube ssh "sudo cp /home/docker/ca.pem /etc/docker/certs.d/$CERT_DIR:5000/ca.crt"

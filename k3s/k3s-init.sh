#!/bin/bash

# Check k3s installation
k3s_path=$(which k3s)
if [ -x "$k3s_path" ]; then
    echo "k3s is already installed"
    exit 0
fi

# Install k3s
$ curl -sfL https://get.k3s.io | K3S_KUBECONFIG_MODE="644" sh -s -

K3S_KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# Check if kubectl is installed
echo "Checking if kubectl is in PATH..."
kubectl_path=$(which kubectl)
if [ -x "$kubectl_path" ]; then
    echo "kubectl is installed, merging kubeconfig files..."
    
    # Backup the existing kubeconfig file if it exists
    if [ -f ~/.kube/config ]; then
        KUBECONFIG_BACKUP=~/.kube/config.backup
        cp -p ~/.kube/config $KUBECONFIG_BACKUP
        echo "Existing kubeconfig backed up to $KUBECONFIG_BACKUP"
    fi

    # Merge kubeconfig files
    env KUBECONFIG=~/.kube/config:$K3S_KUBECONFIG kubectl config view --merge --flatten > /tmp/config
    cat /tmp/config > ~/.kube/config && rm /tmp/config
    kubectl config rename-context default k3s
    echo "kubeconfig files merged successfully"
else
    echo "kubectl is not installed, skipping kubeconfig merge, use k3s kubectl"
fi

echo "k3s installation and configuration completed successfully"

#!/bin/bash

# Check k3s installation
K3S_PATH=$(which k3s)
if [ -x "$K3S_PATH" ]; then
    echo "k3s is already installed"
    exit 0
fi

# Install k3s
curl -sfL https://get.k3s.io | K3S_KUBECONFIG_MODE="644" sh -s -

K3S_KUBECONFIG=/etc/rancher/k3s/k3s.yaml
DEFAULT_KUBECONFIG=~/.kube/config
TMP_KUBECONFIG=~/.kube/config.k3s_merge

# Check if kubectl is installed
echo "Checking if kubectl is in PATH..."
KUBECTL_PATH=$(which kubectl)
if [ -x "$KUBECTL_PATH" ]; then
    echo "kubectl is installed, merging kubeconfig files..."

    # Backup the existing kubeconfig file if it exists
    if [ -f $DEFAULT_KUBECONFIG ]; then
        KUBECONFIG_BACKUP=$DEFAULT_KUBECONFIG.backup
        cp -p $DEFAULT_KUBECONFIG $KUBECONFIG_BACKUP
        echo "Existing kubeconfig backed up to $KUBECONFIG_BACKUP"
    fi

    # Merge kubeconfig files
    env KUBECONFIG=$DEFAULT_KUBECONFIG:$K3S_KUBECONFIG kubectl config view --merge --flatten > $TMP_KUBECONFIG
    cat $TMP_KUBECONFIG > $DEFAULT_KUBECONFIG && rm $TMP_KUBECONFIG
    kubectl config rename-context default k3s
    echo "kubeconfig files merged successfully"
else
    echo "kubectl is not installed, skipping kubeconfig merge, use k3s kubectl"
fi

echo "k3s installation and configuration completed successfully"

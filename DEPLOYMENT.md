---
title: Installing Telemetry Server on Amazon EKS
---

This page covers installing Telemetry Server on an Amazon EKS cluster.

If you already have an EKS Kubernetes cluster, skip to the step about [installing an ingress.](#5-install-an-ingress) Then install the TelemetryServer Helm chart following the instructions at the end of this document.

## Creating an EKS Cluster for the Telemetry Server

In this section, you'll install an EKS cluster with an ingress by using command line tools. 
:::note Prerequisites:

- You should already have an AWS account.
- It is recommended to use an IAM user instead of the root AWS account. You will need the IAM user's access key and secret key to configure the AWS command line interface.
- The IAM user needs the minimum IAM policies described in the official [eksctl documentation.](https://eksctl.io/usage/minimum-iam-policies/)

:::

### 1. Prepare your Workstation

Install the following command line tools on your workstation:

- **The AWS CLI v2:** For help, refer to these [installation steps.](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
- **eksctl:** For help, refer to these [installation steps.](https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html)
- **kubectl:** For help, refer to these [installation steps.](https://docs.aws.amazon.com/eks/latest/userguide/install-kubectl.html)
- **helm:** For help, refer to these [installation steps.](https://helm.sh/docs/intro/install/)

### 2. Configure the AWS CLI

To configure the AWS CLI, run the following command:

```
aws configure
```

Then enter the following values:

| Value | Description |
|-------|-------------|
| AWS Access Key ID | The access key credential for the IAM user with EKS permissions. |
| AWS Secret Access Key | The secret key credential for the IAM user with EKS permissions. |
| Default region name | An [AWS region](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.RegionsAndAvailabilityZones.html#Concepts.RegionsAndAvailabilityZones.Regions) where the cluster nodes will be located. |
| Default output format | Enter `json`. |

### 3. Create the EKS Cluster

To create an EKS cluster, run the following command. Use the AWS region that applies to your use case.

```
eksctl create cluster \
  --name Telemetry-Server \
  --version <VERSION> \
  --region <REGION> \
  --nodegroup-name TelemetryServerNodes \
  --nodes 3 \
  --nodes-min 1 \
  --nodes-max 4 \
  --managed
```

The cluster will take some time to be deployed with CloudFormation.

### 4. Test the Cluster

To test the cluster, run:

```
eksctl get cluster --region <REGION>
```

The result should look like the following:

```
eksctl get cluster --region us-west-2
NAME			REGION		EKSCTL CREATED
Telemetry-Server	us-west-2	True
```

### 5. Install an Ingress

The cluster needs an Ingress so that TelemetryServer can be accessed from outside the cluster.

To make sure that you choose the correct Ingress-NGINX Helm chart, first find an `Ingress-NGINX version` that's compatible with your Kubernetes version in the [Kubernetes/ingress-nginx support table](https://github.com/kubernetes/ingress-nginx#supported-versions-table).

Then, list the Helm charts available to you by running the following command:

```
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm search repo ingress-nginx -l
```

The `helm search` command's output contains an `APP VERSION` column. The versions under this column are equivalent to the `Ingress-NGINX version` you chose earlier. Using the app version, select a chart version that bundles an app compatible with your Kubernetes install. For example, if you have Kubernetes v1.30, you can select the 4.10.1 Helm chart, since Ingress-NGINX v1.10.1 comes bundled with that chart, and v1.10.1 is compatible with Kubernetes v1.30. When in doubt, select the most recent compatible version.

Now that you know which Helm chart `version` you need, run the following command. It installs an `nginx-ingress-controller` with a Kubernetes load balancer service:

```
helm upgrade --install \
  ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --set controller.service.type=LoadBalancer \
  --set controller.ingressClassResource.name=nginx \
  --set controller.ingressClassResource.default=false \
  --version 4.10.1 \
  --create-namespace
```

### 6. Get Load Balancer IP

To get the address of the load balancer, run:

```
kubectl get service ingress-nginx-controller --namespace=ingress-nginx
```

The result should look similar to the following:

```
NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP                                                              PORT(S)                      AGE
ingress-nginx-controller   LoadBalancer   10.100.187.35   adcc226dc1f2f48b195ed9b489354f43-670106350.us-west-2.elb.amazonaws.com   80:31097/TCP,443:30670/TCP   86s
```

Save the `EXTERNAL-IP`.

### 7. Set up DNS

External traffic to the Telemetry Server will need to be directed at the load balancer you created.

Set up a DNS to point at the external IP that you saved. This DNS will be used as the Telemetry Server URL.

### 8. Install the Telemetry Server Helm Chart

Next, install the Telemetry Server Helm chart by following the instructions here;

Use that DNS name from the previous step as the Telemetry Server URL when you install TelemetryServer. It can be passed in as a Helm option. For example, if the DNS name is `telemetry.suseclouddev.com`, you could run the Helm installation command with the option `--set ingress.hosts[0].host=telemetry.suseclouddev.com`.

Until the charts are published in the chart repository, we will use the charts from the repo with this command to install:

```
helm upgrade --install \
  telemetry-server ./chart \
  --namespace telemetry \
  --set image.repository="docker.io/mbelur/ts" \
  --set image.tag="v0.2.0" \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=telemetry.suseclouddev.com \
  --set ingress.hosts[0].paths[0].path="/" \
  --set ingress.hosts[0].paths[0].pathType="ImplementationSpecific" \
  --create-namespace
```

### 9. Verification of the deployment

Make sure the telemetry-server pod is READY/RUNNING

```kubectl get pods -A
NAMESPACE       NAME                                       READY   STATUS    RESTARTS   AGE
ingress-nginx   ingress-nginx-controller-cf668668c-lbjxv   1/1     Running   0          35m
kube-system     aws-node-4d5v5                             2/2     Running   0          52m
kube-system     aws-node-7ks4s                             2/2     Running   0          52m
kube-system     aws-node-gmzcl                             2/2     Running   0          52m
kube-system     coredns-787cb67946-6lzv2                   1/1     Running   0          57m
kube-system     coredns-787cb67946-7jzmm                   1/1     Running   0          57m
kube-system     kube-proxy-9dfp8                           1/1     Running   0          52m
kube-system     kube-proxy-csrk8                           1/1     Running   0          52m
kube-system     kube-proxy-n9rm4                           1/1     Running   0          52m
telemetry       telemetry-server-646955f68-5rg6p           1/1     Running   0          5m22s

```

Functional Verification that the API server is responding:

```
curl http://telemetry.suseclouddev.com/healthz
```
returns
```
"{\"alive\": true}"
```


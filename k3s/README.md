# Local Development Environment Setup

This guide will help you set up your local development environment for the SUSE Telemetry Gateway service.

## Setting up k3s

1. Navigate to the `k3s` directory:
    ```sh
    cd k3s
    ```
2. Run the `k3s-init.sh` script to install and start k3s:
    ```sh
    ./k3s-init.sh
    ```

## Deploying PostgreSQL

1. Navigate to the `charts/postgres` directory:
    ```sh
    cd ../charts/postgres
    ```
2. Deploy PostgreSQL using Helm:
    ```sh
    helm install postgres . -n telemetry --create-namespace
    ```

## Deploying Telemetry Server

1. Navigate to the `charts/telemetry-server` directory:
    ```sh
    cd ../telemetry-server
    ```
2. Deploy Telemetry Server using Helm:
    ```sh
    helm install telemetry-server . -n telemetry
    ```

## Debugging

### Checking Pods

1. Get the list of pods:
    ```sh
    kubectl get pods -n telemetry
    ```

2. Check the status of the PostgreSQL pod:
    ```sh
    kubectl get pod -l app=postgres -n telemetry
    ```

3. Check the status of the Telemetry Server pod:
    ```sh
    kubectl get pod -l app=telemetry-server -n telemetry
    ```

### Checking Services

1. Get the list of services:
    ```sh
    kubectl get svc -n telemetry
    ```

### Logs and Descriptions

1. Check the logs of the PostgreSQL pod:
    ```sh
    kubectl logs -l app=postgres -n telemetry
    ```

2. Check the logs of the Telemetry Server pod:
    ```sh
    kubectl logs -l app=telemetry-server -n telemetry
    ```

3. Describe the PostgreSQL pod for more details:
    ```sh
    kubectl describe pod -l app=postgres -n telemetry
    ```

4. Describe the Telemetry Server pod for more details:
    ```sh
    kubectl describe pod -l app=telemetry-server -n telemetry
    ```

## Cleaning Up

To clean up your local development environment:

1. Delete the Helm releases:
    ```sh
    helm uninstall telemetry-server -n telemetry
    helm uninstall postgres -n telemetry
    ```

2. Delete the k3s cluster:
    ```sh
    /usr/local/bin/k3s-uninstall.sh
    ```

3. Remove the local mounted volumes:
    ```sh
    sudo rm -rf /mnt/data
    ```


## Stopping k3s

If you wish to stop k3s instead of completely uninstalling it run:

  ```sh
  /usr/local/bin/k3s-killall.sh
  ```
## Restarting k3s

To restart k3s, run the following commands:

1. Start k3s service:
    ```sh
    sudo systemctl start k3s
    ```

2. Start k3s-agent service:
    ```sh
    sudo systemctl start k3s-agent
    ```

## Testing

Once the deployments are up and running, you can test the Telemetry Server by port-forwarding the telemetry-server pod.

1. **Port Forward the Telemetry Server Pod:**

    First, identify the Telemetry Server pod name:
    ```sh
    kubectl get pods -n telemetry -l app=telemetry-server
    ```

    Use the pod name to set up port forwarding:
    ```sh
    kubectl port-forward -n telemetry <telemetry-server-pod-name> 8080:8080
    ```

    This command forwards port 8080 on your local machine to port 8080 on the Telemetry Server pod.

2. Follow the steps on telemetry-server README.md as usual.

By following these steps, you can confirm that the Telemetry Server is operational and capable of handling requests.

## Notes

- This is a local k3s installation for test purposes. Changing permissions for k3s kubeconfig files is not recommended for production environments.
- Update the Helm values as necessary in the `values.yaml` files for both PostgreSQL and Telemetry Server before deploying.

By following these steps, you should have a local k3s cluster running with PostgreSQL and Telemetry Server deployed.

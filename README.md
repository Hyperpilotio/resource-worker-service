# Resource Worker Service

## Building:

### Installing Dependencies:
`make init`

### Building:
`make build`

### Local testing:
`make run`

`make test` (from a separate terminal)

### Building Docker Image:
`make docker-build`

## Deploying:

### 1. Create an empty Kubernetes cluster:

- Deploying with GCP:
    1. `gcloud container clusters create resource-worker-service`
    2. `gcloud container clusters get-credentials resource-worker-service`

- Deploying onto AWS with deployer
    1. Deploy an empty K8S cluster: `./deploy-k8s.sh [username]`
    2. Retrieve K8S credentials (or through the UI): `curl localhost:7777/v1/deployments/[DEPLOYMENT_ID]/kubeconfig > ~/.kube/config`

### 2. Deploy stats collector
- prometheus
    1. `kubectl create -f prometheus-k8s/`
- statsd
    1. `kubectl create -f statsd-k8s/`
    2. Modify `deploy/statefulset.yaml` so that `STATS_PUBLISHER` environment variable is set to "statsd", see the comment section in it

### 3. Deploy resource worker service
- `kubectl create -f deploy/`

### 4. Access the Service Through K8S Proxy:
- `kubectl proxy`
- `curl -XPOST -H "Content-Type: application/json" -d '{"requests": [{"cpu": {"cycles": 300}}]}' http://localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`
- Or with httpie: `echo '{"requests": [{"cpu": {"cycles": 300}}]}' | http POST localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`

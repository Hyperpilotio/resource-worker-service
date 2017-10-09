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

### Deploying with GCP:
- `gcloud container clusters create resource-worker-service`
- `gcloud container clusters get-credentials resource-worker-service`
- `kubectl create -f deploy/`

## Deploying onto AWS with deployer
- Deploy an empty K8S cluster: `./deploy-k8s.sh [username]`
- Retrieve K8S credentials (or through the UI): `curl localhost:7777/v1/deployments/[DEPLOYMENT_ID]/kubeconfig > ~/.kube/config`
- `kubectl create -f deploy/`

### Accessing the Service Through K8S Proxy:
- `kubectl proxy`
- `curl -XPOST -H "Content-Type: application/json" -d '{"requests": [{"cpu": {"cycles": 300}}]}' http://localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`
- Or with httpie: `echo '{"requests": [{"cpu": {"cycles": 300}}]}' | http POST localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`

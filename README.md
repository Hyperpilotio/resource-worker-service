# Resource Worker Service

## Building:

### Installing Dependencies:
`make init`

### Building:
`make build`

### Building Docker Image:
`make docker-build`

## Deploying onto Kubernetes:
`kubectl apply -f deploy/`

### Accessing the Service Through K8S Proxy:
- `kubectl proxy`
- `curl -XPOST -H "Content-Type: application/json" -d '{"requests": [{"cpu": {"cycles": 300}}]}' http://localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`
- Or with httpie: `echo '{"requests": [{"cpu": {"cycles": 300}}]}' | http POST localhost:8001/api/v1/namespaces/default/services/resource-worker-service/proxy/run`

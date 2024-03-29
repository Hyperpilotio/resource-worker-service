GLIDE=$(which glide)
GO_EXECUTABLE ?= go

ORGANIZATION=hyperpilot
IMAGE=resource-worker-service
TAG=latest

glide-check:
	@if [ -z $(GLIDE) ]; then \
		echo "glide doesn't exist."; \
		curl https://glide.sh/get | sh; \
	else \
		echo "glide installed"; \
	fi

init: glide-check
	glide install

build: init
	CGO_ENABLED=0 go build -a -installsuffix cgo

run:
	./resource-worker-service -logtostderr=true -v=2

test:
	dd if=/dev/zero of=testfile bs=100M count=1
	curl -XPOST -H "Content-Type: application/json" localhost:7998/run -d @resource-requests.json
	curl -XPOST -H "Content-Type: application/json" localhost:7998/run -d @multi-resource-requests.json

build-linux: init
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -installsuffix cgo

docker-build:
	docker build --no-cache -t $(ORGANIZATION)/$(IMAGE):$(TAG) .

docker-push:
	docker push ${ORGANIZATION}/${IMAGE}:${TAG}

deploy-k8sconntrack:
	@echo "Deploying k8sconntrack ..."
	# Make sure you set up the path of kubeconfig in the following command
	# ex: kubectl create secret generic vmt-config --from-file ~/.kube/config
	kubectl create secret generic vmt-config --from-file 
	kubectl create -f ./k8sconntrack


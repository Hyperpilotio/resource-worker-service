GLIDE=$(which glide)
GO_EXECUTABLE ?= go

ORGANIZATION=hyperpilot
IMAGE=resource-worker-service
TAG=latest

glide-check:
	@if [ -z $GLIDE ]; then \
		echo "glide doesn't exist."; \
		curl https://glide.sh/get | sh; \
	else \
		echo "glide installed"; \
	fi

init: glide-check
	glide install

build: init
	$(GO_EXECUTABLE) build .

run:
	./resource-worker-service -logtostderr=true -v=2

test:
	dd if=/dev/zero of=testfile bs=100M count=1
	curl -XPOST -H "Content-Type: application/json" localhost:7998/run -d @resource-requests.json

build-linux: init
	GOOS=linux GOARCH=amd64 $(GO_EXECUTABLE) build .

docker-build: build-linux
	docker build -t $(ORGANIZATION)/$(IMAGE):$(TAG) .

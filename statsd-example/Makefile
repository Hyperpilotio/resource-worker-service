OUTPUT_DIR=./_output
BINARY=${OUTPUT_DIR}/server
ORGANIZATION=hyperpilot
IMAGE=statsd-example
TAG=test

build: clean
	env GOOS=linux go build -o ${BINARY} .

docker:
	docker build -t ${ORGANIZATION}/${IMAGE}:${TAG} .

test:
	${ENVS} go test ./...

.PHONY: clean
clean:
	@if [ -f ${OUTPUT_DIR} ]; then rm -rf ${OUTPUT_DIR};fi

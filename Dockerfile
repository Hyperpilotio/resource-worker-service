FROM alpine:3.4

WORKDIR /home

COPY resource-worker-service /home/resource-worker-service

CMD ["/home/resource-worker-service"]

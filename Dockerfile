FROM alpine:3.4

WORKDIR /home

RUN dd if=/dev/zero of=50MBfile bs=50M count=1

COPY resource-worker-service /home/resource-worker-service

CMD ["/home/resource-worker-service"]

FROM alpine:3.4

WORKDIR /home

RUN dd if=/dev/zero of=testfile bs=100M count=1

COPY entrypoint.sh /home/entrypoint.sh
COPY resource-worker-service /home/resource-worker-service

CMD ["/home/entrypoint.sh"]

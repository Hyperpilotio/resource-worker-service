#!/bin/sh

echo 3 > $HOST_PROC/sys/vm/drop_caches

./resource-worker-service

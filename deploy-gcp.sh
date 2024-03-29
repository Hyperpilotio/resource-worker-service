#!/usr/bin/env bash

if [ "$#" -ne 1 ]
then
    echo "Usage: deploy-gcp.sh <userId>"
    exit 1
fi

DEPLOYER_URL="localhost"

curl -XPOST $DEPLOYER_URL:7777/v1/users/$1/deployments --data-binary @deploy-gcp.json

echo "Please check progress of your deployment at http://$DEPLOYER_URL:7777/ui"

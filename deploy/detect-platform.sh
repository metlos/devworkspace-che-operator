#!/bin/sh

K8S_CLI=$1

ROUTES_PRESENT=$(${K8S_CLI} api-resources --api-group='route.openshift.io'  2>&1 | grep -o routes)
if [ "${ROUTES_PRESENT}" == "routes" ]; then \
    echo openshift
else
    echo kubernetes
fi


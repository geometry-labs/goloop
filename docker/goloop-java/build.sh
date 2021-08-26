#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOBUILD_TAGS=${GOBUILD_TAGS}
if [ ! -z "${GOBUILD_TAGS}" ] && [ -z "${GOBUILD_TAGS##*rocksdb*}" ]; then
  IMAGE_SUFFIX_DB_TYPE=-rocksdb
fi
export IMAGE_BASE=${IMAGE_BASE:-goloop/base-java${IMAGE_SUFFIX_DB_TYPE}:latest}

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOLOOP_JAVA=${IMAGE_GOLOOP_JAVA:-goloop-java:latest}

./update.sh "${IMAGE_GOLOOP_JAVA}" ../..

cd $PRE_PWD

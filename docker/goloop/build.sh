#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export GOBUILD_TAGS=${GOBUILD_TAGS}
if [ ! -z "${GOBUILD_TAGS}" ] && [ -z "${GOBUILD_TAGS##*rocksdb*}" ]; then
  IMAGE_SUFFIX_DB_TYPE=-rocksdb
fi
export IMAGE_BASE=${IMAGE_BASE:-goloop/base-all${IMAGE_SUFFIX_DB_TYPE}:latest}

export GOLOOP_VERSION=${GOLOOP_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOLOOP=${IMAGE_GOLOOP:-goloop:latest}

./update.sh "${IMAGE_GOLOOP}" ../..

cd $PRE_PWD

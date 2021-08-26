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

export GOCHAIN_VERSION=${GOCHAIN_VERSION:-$(git describe --always --tags --dirty)}
IMAGE_GOCHAIN=${IMAGE_GOCHAIN:-goloop/gochain:latest}

./update.sh "${IMAGE_GOCHAIN}" ../..

cd $PRE_PWD

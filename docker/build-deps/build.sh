#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

export IMAGE_GO_DEPS=${IMAGE_GO_DEPS:-goloop/go-deps:latest}
export IMAGE_PY_DEPS=${IMAGE_PY_DEPS:-goloop/py-deps:latest}
export IMAGE_JAVA_DEPS=${IMAGE_JAVA_DEPS:-goloop/java-deps:latest}
export IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS:-goloop/rocksdb-deps:latest}
IMAGE_BUILD_DEPS=${IMAGE_BUILD_DEPS:-goloop/build-deps:latest}

./update.sh "build" "${IMAGE_BUILD_DEPS}" ../..

cd $PRE_PWD

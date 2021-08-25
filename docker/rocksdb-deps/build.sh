#!/bin/sh
set -e

PRE_PWD=$(pwd)
WORKDIR=$(dirname "$(readlink -f ${0})")
cd $WORKDIR

IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS:-goloop/rocksdb-deps:latest}

./update.sh "${IMAGE_ROCKSDB_DEPS}" ../..

cd $PRE_PWD

#!/bin/sh

BASE_DIR=$(dirname $0)
. ${BASE_DIR}/../version.sh
. ${BASE_DIR}/../function.sh

LABEL="GOLOOP_BASE_SHA"

get_hash_of_dir() {
    local BASE=$1
    local SRC_DIR=$2
    local SUM=$(get_hash_of_files \
        "${SRC_DIR}/docker/base/Dockerfile")
    echo "${BASE}-${ALPINE_VERSION}-${IMAGE_PY_DEPS}-${IMAGE_ROCKSDB_DEPS}-${SUM}"
}

update_image() {
    if [ $# -lt 1 ] ; then
        echo "Usage: $0 <engine> [<image_name>] [<src_dir>] [<build_dir>]"
        echo "\t <engine>: all, py, java"
        return 1
    fi

    local ENGINE=${1}
    case $TARGET in
    all);;py);;java);;
    *)
      echo "invalid engine ${ENGINE}"
      return 1
    ;;
    esac

    local BASE=base-${ENGINE}
    if [ ! -z "${GOBUILD_TAGS}" ] && [ -z "${GOBUILD_TAGS##*rocksdb*}" ]; then
      BASE=${BASE}-rocksdb
    fi

    local TARGET_IMAGE=${2:-goloop/${BASE}:latest}
    local TARGET_REPO=${TARGET_IMAGE%%:*}
    local SRC_DIR=${3}
    if [ -z "${SRC_DIR}" ] ; then
        SRC_DIR="."
    fi
    local BUILD_DIR=${4}

    local HASH_OF_DIR=$(get_hash_of_dir ${BASE} ${SRC_DIR})
    local HASH_OF_IMAGE=$(get_label_of_image ${LABEL} ${TARGET_IMAGE})

    if [ "${HASH_OF_DIR}" != "${HASH_OF_IMAGE}" ] ; then
        local IMAGE_ID=$(get_id_with_hash ${TARGET_REPO} ${LABEL} ${HASH_OF_DIR})
        if [ "${IMAGE_ID}" != "" ]; then
            echo "Tagging image ${IMAGE_ID} as ${TARGET_IMAGE}"
            docker tag ${IMAGE_ID} ${TARGET_IMAGE}
            return $?
        fi

        # Prepare build directory if it's set
        if [ "${BUILD_DIR}" != "" ] ; then
            rm -rf ${BUILD_DIR}
            mkdir -p ${BUILD_DIR}
            cp ${BASE_DIR}/* ${BUILD_DIR}
        else
            BUILD_DIR=${BASE_DIR}
        fi

        CDIR=$(pwd)
        cd ${BUILD_DIR}

        echo "Building image ${TARGET_IMAGE}"
        echo "ALPINE_VERSION=${ALPINE_VERSION} IMAGE_PY_DEPS=${IMAGE_PY_DEPS}, IMAGE_ROCKSDB_DEPS=${IMAGE_ROCKSDB_DEPS}, BASE=${BASE}"
        docker build \
            --build-arg ${LABEL}=${HASH_OF_DIR} \
            --build-arg ALPINE_VERSION="${ALPINE_VERSION}" \
            --build-arg IMAGE_PY_DEPS="${IMAGE_PY_DEPS}" \
            --build-arg IMAGE_ROCKSDB_DEPS="${IMAGE_ROCKSDB_DEPS}" \
            --build-arg BASE="${BASE}" \
            --tag ${TARGET_IMAGE} .

        local result=$?
        cd ${CDIR}
        return $result
    else
        echo "Reuse image ${TARGET_IMAGE} for ${HASH_OF_DIR}"
        return 0
    fi
    return 0
}

update_image "$@"

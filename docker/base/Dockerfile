ARG ALPINE_VERSION
ARG IMAGE_PY_DEPS
ARG IMAGE_ROCKSDB_DEPS
ARG BASE
ARG BASE_SHA
FROM alpine:${ALPINE_VERSION} as base-java
RUN apk add --no-cache openjdk11-jre-headless

FROM ${IMAGE_ROCKSDB_DEPS} as rocksdb

FROM base-java as base-java-rocksdb
RUN apk add --update --no-cache zlib bzip2 snappy lz4-dev zstd-dev libtbb gflags
COPY --from=rocksdb /work/rocksdb/lib /usr/lib/

ARG IMAGE_PY_DEPS
FROM ${IMAGE_PY_DEPS} as base-py

FROM base-py as base-py-rocksdb
RUN apk add --update --no-cache zlib bzip2 snappy lz4-dev zstd-dev libtbb gflags
COPY --from=rocksdb /work/rocksdb/lib /usr/lib/

FROM base-py as base-all
RUN apk add --no-cache openjdk11-jre-headless

FROM base-all as base-all-rocksdb
RUN apk add --update --no-cache zlib bzip2 snappy lz4-dev zstd-dev libtbb gflags
COPY --from=rocksdb /work/rocksdb/lib /usr/lib/

ARG BASE
FROM ${BASE}
LABEL MAINTAINER="t_arch@iconloop.com"

ARG GOLOOP_BASE_SHA
LABEL GOLOOP_BASE_SHA="$GOLOOP_BASE_SHA"
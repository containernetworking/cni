#!/usr/bin/env bash
set -xe

SRC_DIR="${SRC_DIR:-$PWD}"

FEDORA_INSTALL="dnf install -y golang tar xz bzip2 gzip sudo iproute"
FEDORA_IMAGE="docker://fedora:23"

TAG=$(git describe --exact-match --abbrev=0) || TAG=$(git describe)
RELEASE_DIR=release-${TAG}
OUTPUT_DIR=bin

rm -Rf ${SRC_DIR}/${RELEASE_DIR}
mkdir -p ${SRC_DIR}/${RELEASE_DIR}

sudo -E rkt run \
  --volume rslvconf,kind=host,source=/etc/resolv.conf \
  --mount volume=rslvconf,target=/etc/resolv.conf \
  --volume src-dir,kind=host,source=$SRC_DIR \
  --mount volume=src-dir,target=/opt/src \
  --interactive \
  --insecure-options=image \
  ${FEDORA_IMAGE} \
  --exec /bin/bash \
  -- -xe -c "\
    ${FEDORA_INSTALL}; cd /opt/src; umask 0022; ./build; ./test || true; \
    for format in txz tbz2 tgz; do \
      FILENAME=CNI-${TAG}.\$format; \
      FILEPATH=${RELEASE_DIR}/\$FILENAME; \
      tar -C ${OUTPUT_DIR} --owner=0 --group=0 -caf \$FILEPATH .; \
      pushd ${RELEASE_DIR}; md5sum \$FILENAME > \$FILENAME.md5; popd; \
    done; \
    chown -R ${UID} ${OUTPUT_DIR} ${RELEASE_DIR}; \
    :"

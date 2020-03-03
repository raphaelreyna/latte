#!/bin/bash

# ----------------------------------------------------------------------
# A bash script to parametrically build and tag Docker images for Latte
# Written by: Raphael Reyna
# Last updated on: 2020/03/02
# ----------------------------------------------------------------------

TMP_DOCKERFILE="Dockerfile.tmp"
IMAGE_TAG=""
PACKAGES=""
TAGS=""

while getopts 'd:t:p:' flag; do
    case "${flag}" in
        d) TAGS="${TAGS},${OPTARG}" ;;
        p) PACKAGES="${PACKAGES} ${OPTARG}" ;;
        t) IMAGE_TAG="${OPTARG}" ;;
    esac
done

# Add '-tags' flag to TAGS if user provided any
[[ "${TAGS}" != "" ]] && TAGS="-tags ${TAGS:1}"

# Make sure we have at least a minimal tex installation
[[ "${PACKAGES}" == "" ]] && PACKAGES="texlive-base"

# Make sure we have an image build tag
[[ "${IMAGE_TAG}" == "" ]] && echo "no image tag passed" ; exit 1

# Create Dockerfile from template
sed "s/{BUILD_TAGS}/${TAGS}/g;s/{TEXLIVE_PACKAGES}/${PACKAGES}/g" \
    ./build/Dockerfile.template > "${TMP_DOCKERFILE}"

# Build the image and clean up
docker build --tag "${IMAGE_TAG}" -f "${TMP_DOCKERFILE}" .. && rm "${TMP_DOCKERFILE}"

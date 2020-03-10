#!/bin/bash

# ----------------------------------------------------------------------
# A bash script to parametrically build and tag Docker images for Latte.
# Written by: Raphael Reyna
# ----------------------------------------------------------------------

DOCKERFILE="\
# Build Stage
FROM golang:1.13 AS build-stage
ADD ./ /latte
RUN cd /latte && env GOOS=linux GOARCH=amd64 go build {BUILD_TAGS} ./cmd/latte

# Final Stage
FROM ubuntu:19.10
MAINTAINER Raphael Reyna <raphaelreyna@protonmail.com>
RUN apt update -q \\
  && env DEBIAN_FRONTEND=noninteractive \\
  apt install -qy {TEXLIVE_PACKAGES} \\
  && rm -rf /var/lib/apt/lists/*
COPY --from=build-stage /latte/latte /bin/latte
CMD [\"latte\"]\
"

IMAGE_TAG=""
PACKAGES=""
TAGS=""
SKIP=""
NO_CONFIRM=""
ME=`basename "$0"`
DESCRIPTOR=""
USERNAME="raphaelreyna"
HOSTNAME=""

function usage {
    echo "\
Usage: ${ME} [-h] [-s] [-b build_tag] [-p latex_package] [-t image_tag] \
[-d descriptor] [-H host_name] [-u user_name]

Description: ${ME} parametrically builds and tags Docker images for Latte.
             The tag used for the image follows the template bellow:
                 host_name/user_name/image_name:image_tag-descriptor

Flags:
  -b Build tags to be passed to the Go compiler.

  -d Descriptor to be used when tagging the built image.

  -h Show this help text.

  -p LaTeX package to install, must be available in default Ubuntu repos.
     (default: texlive-base)

  -s Skip the image building stage. The generated Dockerfile will be sent to std out.

  -t Tag the Docker image.
     The image will be tagged with the current git tag if this flag is omitted.
     If no git tag is found, we default to using 'latest' as the image tag.

  -u Username to be used when tagging the built image.
     (default: raphaelreyna)

  -H Hostname to be used when tagging the built image.

  -y Do not ask to confirm image tab before building image.\
"
    exit 0
}

function generate_dockerfile {
    TMP_DOCKERFILE="Dockerfile.tmp"
    [[ "${TAGS}" != "" ]] && TAGS="-tags ${TAGS:1}"
    # Make sure we have at least a minimal tex installation
    [[ "${PACKAGES}" == "" ]] && PACKAGES="texlive-base"
    echo "${DOCKERFILE}" | \
        sed "s/{BUILD_TAGS}/${TAGS}/g;s/{TEXLIVE_PACKAGES}/${PACKAGES}/g" > "${TMP_DOCKERFILE}"
}

function build_image {
    # Make sure we have an image build tag
    if [ "${IMAGE_TAG}" == "" ]; then
        GIT_TAG=`git describe --tags --abbrev=0 2> /dev/null`
        if [ "${GIT_TAG}" = "" ]; then
            IMAGE_TAG="latest"
        else
            IMAGE_TAG="${GIT_TAG}"
        fi
    fi
    [[ "${HOSTNAME}" != "" ]] && HOSTNAME="${HOSTNAME}/"
    [[ "${DESCRIPTOR}" != "" ]] && DESCRIPTOR="-${DESCRIPTOR}"
    IMAGE_NAME="${HOSTNAME}${USERNAME}/latte:${IMAGE_TAG}${DESCRIPTOR}"
    if [ "${NO_CONFIRM}" != "T" ]; then
        PROMPT="${ME}: tag image as ${IMAGE_NAME} (y/N)? "
        read -p "${PROMPT}" userconfirmation
        if [ "$userconfirmation" != "y" ]; then
            clean_up
            exit 0
        fi
    fi
    docker build --tag "${IMAGE_NAME}" -f "${TMP_DOCKERFILE}" ..
}

function clean_up {
    rm "${TMP_DOCKERFILE}"
}

while getopts 'b:d:t:p:syu:H:' flag; do
    case "${flag}" in
        b) TAGS="${TAGS},${OPTARG}" ;;
        d) DESCRIPTOR="${OPTARG}" ;;
        H) HOSTNAME="${OPTARG}" ;;
        p) PACKAGES="${PACKAGES} ${OPTARG}" ;;
        t) IMAGE_TAG="${OPTARG}" ;;
        s) SKIP="T" ;;
        u) USERNAME="${OPTARG}" ;;
        y) NO_CONFIRM="T" ;;
        \?) usage ;;
    esac
done

generate_dockerfile
if [ "${SKIP}" == "T" ]; then
    cat "${TMP_DOCKERFILE}"
    clean_up
    exit 0
fi
build_image
clean_up

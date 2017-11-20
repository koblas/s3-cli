#!/bin/bash

TAG="v${1}"

if [ "${1}X" = "X" ]; then
  echo "missing tag name"
  exit 2
fi

if [ "${2}" = "--clean" ]; then
    git push --delete origin "${TAG}" 
    git tag --delete "${TAG}"
fi

## HERE HAPPENS THE MAGIC
git tag -a "${TAG}" && \
git push origin "${TAG}" && \
goreleaser --rm-dist
#!/usr/bin/env bash
#
# Update vendored dedendencies.
#
set -e

PROJ="github.com/lucab/dkrender"

if ! [[ "$PWD" = "$GOPATH/src/$PROJ" ]]; then
  echo "must be run from \$GOPATH/src/$PROJ"
  exit 1
fi

if [ ! $(command -v glide) ]; then
	echo "glide: command not found"
	exit 1
fi

glide update --strip-vendor

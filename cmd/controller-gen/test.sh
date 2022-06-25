#!/usr/bin/env bash

set -x

CURRENT=$PWD

links=$(find "$(dirname "$0")/../config/core/300-resources" -type l)
for link in $links; do
  cp "$link" "$link.bkp"
done

pushd $HOME/work/serving-schema

manifests=$PWD/config/core/300-resources
output_dir="$CURRENT"
paths="$PWD/pkg/apis/..."
config="$PWD/hack/schemapatch-config.yaml"

popd

# go run main.go \
dlv debug main.go -- \
  schemapatch:manifests=$manifests,generateEmbeddedObjectMeta=true \
  output:dir=$output_dir \
  paths=$paths \
  typeOverrides=$config


echo $?

for link in $links; do
  cat "$link.bkp" > "$link"
  rm "$link.bkp"
done

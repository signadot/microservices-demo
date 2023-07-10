#!/bin/bash
set -e

pushd .
cd "$(dirname "$0")"
helm install mariadb bitnami/mariadb --version=11.5.0 -f ./values.yaml
popd
#!/bin/bash
# exit when any command fails
set -e

echo "Sandbox id: ${SIGNADOT_SANDBOX_ID}"
echo "Resource name: ${SIGNADOT_RESOURCE_NAME}"

# Undeploy the temporary DB for this Sandbox.
export NAMESPACE=signadot
RELEASE_NAME="signadot-${SIGNADOT_RESOURCE_NAME,,}-${SIGNADOT_SANDBOX_ID}"
echo "Deleting Helm release: ${RELEASE_NAME}"
helm -n ${NAMESPACE} uninstall "${RELEASE_NAME}" --wait --timeout 5m0s
#!/usr/bin/env bash

set -e

VERSION=$(${HELM_PLUGIN_DIR}/yq r ${HELM_PLUGIN_DIR}/plugin.yaml version)

test -f ${HELM_PLUGIN_DIR}/bin/linux/rockvalues || {
    curl https://github.com/rockops/rockvalues/releases/download/$VERSION/rockvalues \
        -o ${HELM_PLUGIN_DIR}/bin/linux/rockvalues || {
        echo "Error: rockvalues binary not found in ${HELM_PLUGIN_DIR}/bin/linux/"
        exit 1
    }
    chmod +x ${HELM_PLUGIN_DIR}/bin/linux/rockvalues
}

cp ${HELM_PLUGIN_DIR}/bin/linux/rockvalues ${HELM_PLUGIN_DIR}/rockvalues

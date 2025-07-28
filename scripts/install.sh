#!/usr/bin/env bash

set -e

mkdir -p ${HELM_PLUGIN_DIR}/bin/linux
cp ${HELM_PLUGIN_DIR}/bin/linux/values-downloader ${HELM_PLUGIN_DIR}/values-downloader

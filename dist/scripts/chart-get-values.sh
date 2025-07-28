#!/usr/bin/env bash

# helm downloader
# Called with args: certFile keyFile caFile full-URL

# Supports the following syntax :
# chart://path/to/file.yaml => gets the file path/to/file.yaml in the chart being installed
# chart://path/to/file.yaml@repo/chartname[:version] => gets the file path/to/file.yaml from the chart chartname
#    "chart" is a chart pulled from the helm repo "repo"
#    "version" is the version to pull (optional)

error() {
    echo >&2
    echo "** ERROR ** : " $* >&2
    echo >&2
}

debug() {
    test "$HELM_DEBUG" = "true" && echo "local-plugin [debug] $*" >&2 || true
}

info() {
    echo "local-plugin [info ] $*" >&2
}

usage() {
    error "Usage: helm values chart://valuefile.yaml[@chart] [chart]"
    exit 1
}


debug "Helm get values tester"
debug "$*"

while [ -n "$1" ]; do
    if [ "$1" = "-f" ]; then
        shift
        echo "# Source: $1"
        echo "$1" | grep "^chart://" > /dev/null
        if [ $? = 0 ]; then
            $HELM_PLUGIN_DIR/rockvalues certFile keyFile caFile "$1" || exit 1
        else
            cat $1 || echo "# File does not exist"
        fi
        echo
        echo "---"
    else
        debug "Param $1 ignored (does not represent a file to load)"
    fi
    shift
done 

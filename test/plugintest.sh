#!/usr/bin/env bash


oneTimeSetUp() {

    compress() {
        local CHART
        local OLDPWD

        cd "$1" || exit 1
        echo "Compressing $PWD"
        for CHART in $(ls); do
            OLDPWD=$PWD
            test -d $CHART/charts && compress $CHART/charts
            cd "$OLDPWD" || exit 1
            tar czvf ${CHART}-1.0.0.tgz $CHART
            rm -rf $CHART
        done
    }

    TOP=$(readlink -f $(dirname $0))
    YQ=$TOP/../yq

    rm -rf $TOP/appgz
    cp -r $TOP/app $TOP/appgz
    pushd $TOP/appgz/charts || exit 1

    compress .

    popd

    helm repo add ingress-nginx-valuetest https://kubernetes.github.io/ingress-nginx
    helm repo update
} 


oneTimeTearDown() {
    test -f /tmp/test.yaml && rm /tmp/test.yaml || true
    rm -rf $TOP/appgz || true

    if helm repo ls | grep -q ingress-nginx-valuetest; then
        helm repo remove ingress-nginx-valuetest || true
    else
        true
    fi
} 


setUp() {
    echo ""
    echo "========================================"
    rm -f /tmp/test.yaml
}

tearDown() {

    test -f /tmp/test.yaml && {
        echo ">>>>> RESULT <<<<<"
        cat /tmp/test.yaml
        grep ' *{}' /tmp/test.yaml > /dev/null
        assertEquals "Rendered values must not contain {}" "1" "$?"
        rm -f /tmp/test.yaml
        echo ">>>>>>>>><<<<<<<<<"
    } || true
}

check() {
    LOCATION=$1
    EXPECT=$2

    ACTUAL=$($YQ read /tmp/test.yaml "$LOCATION")

    assertEquals "Value $LOCATION has a bas value" "$EXPECT" "$ACTUAL"
}

#===========================================


basic() {
    helm values -f chart://extra.yaml test $1 > /tmp/test.yaml
    
    check extra value
    check global.glob1 a
}

testBasicgz() {
    basic appgz
}

testBasic() {
    basic app
}

#===========================================

fileInSubchart() {
    helm values -f chart://extra2.yaml test $1 > /tmp/test.yaml
    
    # Main section in top chart
    check extra value

    # Global section
    check global.glob1 a
    check global.glob2 b

    # Values for subchart
    check subchart1.extra2 value2

    # There should not be a global section in subchart
    check subchart1.global ""
}

testFileInSubchart() {
    fileInSubchart app
}

testFileInSubchartgz() {
    fileInSubchart appgz
}


#===========================================


# File only in subcharts
# Global sections should be merged
fileInSubchartOnly() {
    helm values -f chart://extra3.yaml test $1 > /tmp/test.yaml
    
    # Global section
    check global.glob3 c
    check global.glob4 d

    # Values for subchart
    check subchart1.extra3 value4
    check subchart2.extra3 value3

    # There should not be a global section in subchart
    check subchart1.global ""
    check subchart2.global ""

}

testFileInSubchartOnly() {
    fileInSubchartOnly app
}

testFileInSubchartOnlygz() {
    fileInSubchartOnly appgz
}


#===========================================

# Entry of global section in subchart and main chart 
# The entry in the main chart overrides the subchart
over() {
    helm values -f chart://extraover.yaml test $1 > /tmp/test.yaml

    # main chart
    check over ok

    # Global section
    check global.over ok
    check global.notover ok

    # Values for subchart
    check subchart1.over ok

    # There should not be a global section in subchart
    check subchart1.global ""
}

testOver() {
    over app
}

testOvergz() {
    over appgz
}

#===========================================

# Global section alone in subchart
globalalone() {
    helm values -f chart://glob.yaml test $1 > /tmp/test.yaml
}

testGlobalalone() {
    globalalone app
}

testGlobalalonegz() {
    globalalone appgz
}

#===========================================

# Section defined in chart and subchart
sectionOver() {
    helm values -f chart://sectionover.yaml test app > /tmp/test.yaml

    check subchart1.over.a ok
    check subchart1.over.b ok
    check subchart1.over.c ok
    check subchart2.over.d ok

    # We should have only 1 "over" section per subchart
    NB=$(grep "  over:" /tmp/test.yaml | wc -l)
    assertEquals "2" "$NB"
}


testSectionOver() {
    sectionOver app
}

testSectionOvergz() {
    sectionOver appgz
}

#===========================================


# Get data from a remote chart
testRemote() {
    helm values -f chart://Chart.yaml@ingress-nginx-valuetest/ingress-nginx  test app > /tmp/test.yaml
    check name ingress-nginx
}


# Get data from a remote chart with a specific version
# For this test, we get the Chart.yaml file from the remote chart
# and check that the version is correct
testRemoteWithVersionOfValues() {
    helm values -f chart://Chart.yaml@ingress-nginx-valuetest/ingress-nginx:4.12.0  test app > /tmp/test.yaml
    check version 4.12.0
}


# Get values in the chart with a specific version

testRemoteWithVersion() {
    helm values -f chart://Chart.yaml test ingress-nginx-valuetest/ingress-nginx --version "4.12.0" > /tmp/test.yaml
    check version 4.12.0

    helm values -f chart://Chart.yaml test ingress-nginx-valuetest/ingress-nginx --version 4.12.0 > /tmp/test.yaml
    check version 4.12.0

    helm values -f chart://Chart.yaml test ingress-nginx-valuetest/ingress-nginx --version=4.12.0 > /tmp/test.yaml
    check version 4.12.0

    helm values -f chart://ci/controller-service-values.yaml test ingress-nginx-valuetest/ingress-nginx --version "4.12.0" > /tmp/test.yaml
    check controller.image.tag 1.0.0-dev

    helm values -f chart://ci/controller-service-values.yaml test ingress-nginx-valuetest/ingress-nginx --version 4.12.0 > /tmp/test.yaml
    check controller.image.tag 1.0.0-dev

    helm values -f chart://ci/controller-service-values.yaml test ingress-nginx-valuetest/ingress-nginx --version=4.12.0 > /tmp/test.yaml
    check controller.image.tag 1.0.0-dev
}



#===========================================

testTemplateGlobalOverride() {
    helm template app
    helm template app | grep -q value=default
    assertEquals "Prequisites not met: wrong template. Should contain \"value=default\"" $? 0 

    helm template -f chart://globroot.yaml app
    helm template -f chart://globroot.yaml app | grep -q value=ok
    assertEquals "Wrong template. Should contain \"value=ok\"" $? 0 
}


testTemplateNoGlobalOverride() {
    helm template app
    helm template app | grep -q value=default
    assertEquals "Prequisites not met: wrong template. Should contain \"value=default\"" $? 0 

    helm template -f chart://noglobroot.yaml app
    helm template -f chart://noglobroot.yaml app | grep -q value=default
    assertEquals "Wrong template. Should contain \"value=default\"" $? 0 
}

#===========================================

testRemoteWithRepo() {
    helm values -f chart://ci/controller-service-values.yaml --repo https://kubernetes.github.io/ingress-nginx --version 4.12.0 test ingress-nginx > /tmp/test.yaml
    check controller.image.tag 1.0.0-dev
}

#===========================================

fileInSubchartNew() {
    helm values -f chart://extra4.yaml test $1 > /tmp/test.yaml

    # Global section
    check global.glob4 d

    # Values for subchart
    check subchart3.extra4 value4

    # There should not be a global section in subchart
    check subchart1.global ""
}

testFileInSubchartNew() {
    fileInSubchartNew app
}

testFileInSubchartgzNew() {
    fileInSubchartNew appgz
}

#===========================================

fileInNestedSubchart() {
    helm values -f chart://extra5.yaml test $1 > /tmp/test.yaml
    
    # Global section
    check global.glob5 e
    check global.glob5bis e-bis

    # Values for subchart
    check subchart3.extra5 value5

    # Values for nested subchart
    check subchart3.subchart3_1.extra5bis value5-bis

    # There should not be a global section in nested subchart
    check subchart3.global ""
    check subchart3.subchart3_1.global ""
}

testFileInNestedSubchart() {
    fileInNestedSubchart app
}

testFileInNestedSubchartgz() {
    fileInNestedSubchart appgz
}

#===========================================

fileInNestedSubchartAndFolder() {
    helm values -f chart://values/option/dev.yaml test $1 > /tmp/test.yaml
    
    # Global section
    check global.glob2 b
    check global.glob3 c
    check global.glob4 d
    check global.glob5 e

    # Values for subchart
    check subchart2.dev false

    # Values for nested subchart
    check subchart3.subchart3_1.dev true

    # There should not be a global section in nested subchart
    check subchart3.global ""
    check subchart3.subchart3_1.global ""
}


testFileInNestedSubchartAndFolder() {
    fileInNestedSubchartAndFolder app
}

testFileInNestedSubchartAndFoldergz() {
    fileInNestedSubchartAndFolder appgz
}


#===========================================
# Test with sub.yaml at deepest level
# app
# └── charts
#     └── subchart3
#         └── charts
#             └── subchart3_1
#                 └── charts
#                     └── subchart3_1_1
#                         └── deep.yaml

nestedDeep() {
    helm values -f chart://deep.yaml test $1 > /tmp/test.yaml

    # Check that the value is set at each level
    check subchart3.subchart3_1.subchart3_1_1.value deep
}

testNestedDeep() {
    nestedDeep app
}

testNestedDeepgz() {
    nestedDeep appgz
}


#===========================================
# Test with sub.yaml at multiple levels
# app
# ├── charts
# │   └── subchart3
# │       ├── charts
# │       │   └── subchart3_1
# │       │       ├── charts
# │       │       │   └── subchart3_1_1
# │       │       │       └── sub.yaml
# │       │       └── sub.yaml
# │       └── sub.yaml
# └── sub.yaml

nestedBasic() {
    helm values -f chart://sub.yaml test $1 > /tmp/test.yaml

    # Check that the value is set at each level
    check value sub
    check subchart3.value sub
    check subchart3.subchart3_1.value sub
    check subchart3.subchart3_1.subchart3_1_1.value sub
}

testNestedBasic() {
    nestedBasic app
}

testNestedBasicgz() {
    nestedBasic appgz
}


#===========================================
# Test with sub.yaml at multiple levels
# Chek that top values override sub values
# app
# ├── charts
# │   └── subchart3
# │       ├── charts
# │       │   └── subchart3_1
# │       │       ├── charts
# │       │       │   └── subchart3_1_1
# │       │       │       └── sub.yaml (v1, v2, v3, v4)
# │       │       └── sub.yaml (v1, v2, v3)
# │       └── sub.yaml (v1,v2)
# └── sub.yaml (v1)
# 
# Deepest values are overriden by the top values

nestedOver() {
    helm values -f chart://over.yaml test $1 > /tmp/test.yaml

    # Check that the value is set at each level
    check subchart3.subchart3_1.subchart3_1_1.v1 top
    check subchart3.subchart3_1.subchart3_1_1.v2 depth1
    check subchart3.subchart3_1.subchart3_1_1.v3 depth2
    check subchart3.subchart3_1.subchart3_1_1.v4 depth3

    # Check the global values
    check global.gv1 top
    check global.gv2 depth1
    check global.gv3 depth2
    check global.gv4 depth3

    # The tags are consolidated and brought to the top
    check tags.t1 true
    check tags.t2 true
    check tags.t3 true
    check tags.t4 true
}

testNestedOver() {
    nestedOver app
}

testNestedOvergz() {
    nestedOver appgz
}


#===========================================

# Load shunit2
. shunit/shunit2

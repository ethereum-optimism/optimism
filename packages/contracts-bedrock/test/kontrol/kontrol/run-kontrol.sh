#!/bin/bash

set -euxo pipefail

FOUNDRY_PROFILE=kontrol

kontrol_build() {
    kontrol build                     \
            --verbose                 \
            --require ${lemmas}       \
            --module-import ${module} \
            ${rekompile}
}

kontrol_prove() {
    kontrol prove                              \
            --max-depth ${max_depth}           \
            --max-iterations ${max_iterations} \
            --smt-timeout ${smt_timeout}       \
            --bmc-depth ${bmc_depth}           \
            --workers ${workers}               \
            ${reinit}                          \
            ${bug_report}                      \
            ${break_on_calls}                  \
            ${auto_abstract}                   \
            ${tests}                           \
            ${use_booster}
}

###
# kontrol build options
###
# NOTE: This script should be executed from the `contracts-bedrock` directory
lemmas=test/kontrol/kontrol/pausability-lemmas.k
base_module=PAUSABILITY-LEMMAS
module=CounterTest:${base_module}

rekompile=--rekompile
rekompile=

###
# kontrol prove options
###
max_depth=10000

max_iterations=10000

smt_timeout=100000

bmc_depth=10

workers=2

reinit=--reinit
reinit=

break_on_calls=--no-break-on-calls
# break_on_calls=

auto_abstract=--auto-abstract-gas
# auto_abstract=

bug_report=--bug-report
bug_report=

use_booster=--use-booster
# use_booster=

# List of tests to symbolically execute
tests=""
tests+="--match-test CounterTest.test_SetNumber "

kontrol_build
kontrol_prove

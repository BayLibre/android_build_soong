#!/bin/bash -eu
#
# Copyright 2017 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# To track how long we took to startup. %N isn't supported on Darwin, but
# that's detected in the Go code
export TRACE_BEGIN_SOONG=$(date +%s%N)

# Function to find top of the source tree (if $TOP isn't set) by walking up the
# tree.
function gettop
{
    local TOPFILE=build/soong/root.bp
    if [ -z "${TOP-}" -a -f "${TOP-}/${TOPFILE}" ] ; then
        # The following circumlocution ensures we remove symlinks from TOP.
        (cd $TOP; PWD= /bin/pwd)
    else
        if [ -f $TOPFILE ] ; then
            # The following circumlocution (repeated below as well) ensures
            # that we record the true directory name and not one that is
            # faked up with symlink names.
            PWD= /bin/pwd
        else
            local HERE=$PWD
            T=
            while [ \( ! \( -f $TOPFILE \) \) -a \( $PWD != "/" \) ]; do
                \cd ..
                T=`PWD= /bin/pwd -P`
            done
            \cd $HERE
            if [ -f "$T/$TOPFILE" ]; then
                echo $T
            fi
        fi
    fi
}

# Bootstrap microfactory from source if necessary and use it to build the
# soong_ui binary, then run soong_ui.
function run_go
{
    # Increment when microfactory changes enough that it cannot rebuild itself.
    # Hopefully we never need to use this.
    local mf_version=1

    local mf_src=${TOP}/build/soong/cmd/microfactory/microfactory.go
    local soong_ui_src=${TOP}/build/soong/soong_ui.fact

    local out_dir=${OUT_DIR:-${TOP}/out}
    local mf_bin=${out_dir}/microfactory_$(uname)
    local mf_version_file=${out_dir}/.microfactory_$(uname)_version
    local soong_ui_bin=${out_dir}/soong_ui
    local from_src=1

    if [ -f "${mf_bin}" -a -f "${mf_version_file}" ]; then
        if [ "${mf_version}" -eq $(cat "${mf_version_file}") ]; then
            from_src=0
        fi
    fi

    if [ $from_src -eq 1 ]; then
        "${GOROOT}/bin/go" run "${mf_src}" -s "${mf_src}" -b "${mf_bin}" -o "${soong_ui_bin}" "${soong_ui_src}"
        echo "${mf_version}" >"${mf_version_file}"
    else
        "${mf_bin}" -s "${mf_src}" -b "${mf_bin}" -o "${soong_ui_bin}" "${soong_ui_src}"
    fi

    exec "${out_dir}/soong_ui" "$@"
}

export TOP=$(gettop)
case $(uname) in
    Linux)
        export GOROOT=${TOP}/prebuilts/go/linux-x86/
        ;;
    Darwin)
        export GOROOT=${TOP}/prebuilts/go/darwin-x86/
        ;;
    *) echo "unknown OS:" $(uname) >&2 && exit 1;;
esac

run_go "$@"

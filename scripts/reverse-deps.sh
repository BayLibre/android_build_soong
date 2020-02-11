#!/bin/bash

readonly me=$(basename "${0}")

readonly usage="usage: ${me} {options} target [target...]"

function die() { echo -e "${*}" >&2; exit 2; }

if [ "${ANDROID_BUILD_TOP}" == "" ]; then
    die "${me}: Run lunch or set ANDROID_BUILD_TOP=..."
fi

if [ "${TARGET_PRODUCT}" == "" ]; then
    die "${me}: Run lunch or set TARGET_PRODUCT=..."
fi

if [ ! -f "${ANDROID_BUILD_TOP}/out/combined-${TARGET_PRODUCT}.ninja" ]; then
    die "${me}: Run `m nothing` to build the dependency graph"
fi

# parse the command-line

declare -a targets # one or more targets to evaluate

quiet=false      # whether to suppress progress

sep=" "          # output separator between depth and target

use_stdin=false  # whether to read targets from stdin i.e. target -

while [ $# -gt 0 ]; do
    case "${1:-}" in
      -)
        use_stdin=true
      ;;
      -*)
        flag=$(expr "${1}" : '^-*\(.*\)$')
        case "${flag:-}" in
          q) ;&
          quiet)
            quiet=true;;
          noq) ;&
          noquiet)
            quiet=false;;
          csv)
            sep=",";;
          sep)
            sep="${2?"${usage}"}"; shift;;
          sep=*)
            sep=$(expr "${flag}" : '^sep=\(.*\)$';;
          *)
            die "Unknown flag ${1}"
          ;;
        esac
      ;;
      *)
        targets+=("${1:-}")
      ;;
    esac
    shift
done

if [ ! -v targets[0] ] && ! ${use_stdin}; then
    die "${usage}\n\nNo target specified."
fi

# showProgress when stderr is a tty
if [ -t 2 ] && ! ${quiet}; then
    showProgress=true
else
    showProgress=false
fi

# interactive when both stderr and stdin are tty
if ${showProgress} && [ -t 0 ]; then
    interactive=true
else
    interactive=false
fi


# Reads one input target per line from stdin; outputs (isnotice target) tuples.
#
# output target is a ninja target that the input target depends on
# isnotice in {0,1} with 1 for output targets believed to be license or notice
#
# only argument is the dependency depth indicator
function getDeps() {
    (
      tr '\n' '\0' | \
      xargs -0 \
          "${ANDROID_BUILD_TOP}/prebuilts/build-tools/linux-x86/bin/ninja" \
          -f "${ANDROID_BUILD_TOP}/out/combined-${TARGET_PRODUCT}.ninja" \
          -t query
    ) | awk -v depth="${1}" '
      BEGIN {
        inoutput = 0
      }
      $0 ~ /^\S\S*:$/ {
        inoutput = 0
      }
      inoutput != 0 {
        print gensub(/^\s*/, "", "g")" "depth
      }
      $1 == "outputs:" {
        inoutput = 1
      }
    '
}

readonly tmpFiles=$(mktemp -d "${TMPDIR}.tdeps.XXXXXXXXX")

# The deps files contain unique (isnotice target) tuples where
# isnotice in {0,1} with 1 when ninja target `target` is a license or notice.
readonly oldDeps="${tmpFiles}/old"
readonly newDeps="${tmpFiles}/new"
readonly allDeps="${tmpFiles}/all"

if ${use_stdin}; then # start deps by reading 1 target per line from stdin
  awk '
    NF > 0 {
      print gensub(/\s*$/, "", "g", gensub(/^\s*/, "", "g"))" "0
    }
  ' >"${newDeps}"
else # start with no deps by clearing file
  : >"${newDeps}"
fi

# extend deps by appending targets from command-line
for idx in "${!targets[*]}"; do
    echo "${targets[${idx}]} 0" >>"${newDeps}"
done

# remove duplicates and start with new, old and all the same
sort -u <"${newDeps}" >"${allDeps}"
cp "${allDeps}" "${newDeps}"
cp "${allDeps}" "${oldDeps}"

# report depth of dependenciens when showProgress
depth=0

while true; do
    if ${showProgress}; then
        echo "depth ${depth} has "$(cat "${newDeps}" | wc -l)" targets" >&2
    fi
    depth=$(expr ${depth} + 1)
    ( # recalculate dependencies by combining unique inputs of new deps w. old
        cut -d\  -f1 "${newDeps}" | getDeps "${depth}"
        cat "${oldDeps}"
    ) | sort -n | awk '
      BEGIN {
        prev = ""
      }
      {
        depth = $NF
        $NF = ""
        gsub(/\s*$/, "")
        if ($0 != prev) {
          print gensub(/\s*$/, "", "g")" "depth
        }
        prev = $0
      }
    ' >"${allDeps}"
    # recalculate new dependencies as net additions to old dependencies
    if diff "${oldDeps}" "${allDeps}" --old-line-format='' \
      --new-line-format='%L' --unchanged-line-format='' > "${newDeps}"
    then # stop when none found
        break
    fi
    # recalculate old dependencies for next iteration
    cp "${allDeps}" "${oldDeps}"
done

# found all deps -- clean up last iteration of old and new
rm "${oldDeps}"
rm "${newDeps}"

if ${showProgress}; then
    echo $(cat "${allDeps}" | wc -l)" targets" >&2
fi

awk -v sep="${sep}" '{
  depth = $NF
  $NF = ""
  gsub(/\s*$/, "")
  print depth sep $0
}' "${allDeps}" | sort -n

if ${interactive}; then
    echo -n "`date '+%F %-k:%M:%S'` Delete ${tmpFiles}? [n] " >&2
    read answer
    case "${answer}" in [yY]*) rm -fr "${tmpFiles}";; esac
else
    rm -fr "${tmpFiles}"
fi

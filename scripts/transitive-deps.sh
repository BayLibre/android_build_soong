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

include_order_deps=false    # whether to trace through || "order dependencies"
include_implicit_deps=true  # whether to trace through | "implicit deps"
include_deps=true           # whether to trace through regular explicit deps
quiet=false                 # whether to suppress progress

projects_out=''             # where to output the list of projects
directories_out=''          # where to output the list of directories
targets_out=''              # where to output the list of targets/source files
notices_out=''              # where to output the list of license/notice files

sep=" "                     # separator between md5sum and notice filename

nofollow=''                 # regularexp must fully match targets to skip

# files with the following extensions include dependencies as content
container_types='apex|apk|zip|jar|tar|tgz'

use_stdin=false             # whether to read targets from stdin i.e. target -

while [ $# -gt 0 ]; do
    case "${1:-}" in
      -)
        use_stdin=true
      ;;
      -*)
        flag=$(expr "${1}" : '^-*\(.*\)$')
        case "${flag:-}" in
          order_deps)
            include_order_deps=true;;
          noorder_deps)
            include_order_deps=false;;
          implicit)
            include_implicit_deps=true;;
          noimplicit)
            include_implicit_deps=false;;
          direct)
            include_deps=true;;
          nodirect)
            include_deps=false;;
          csv)
            sep=",";;
          sep)
            sep="${2?"${usage}"}"; shift;;
          sep=)
            sep=$(expr "${flag}" : '^sep=\(.*\)$');;
          q) ;&
          quiet)
            quiet=true;;
          noq) ;&
          noquiet)
            quiet=false;;
          nofollow)
            case "${nofollow}" in
              '')
                nofollow="${2?"${usage}"}";;
              *)
                nofollow="${nofollow}|${2?"${usage}"}";;
            esac
            shift
          ;;
          nofollow=*)
            case "${nofollow}" in
              '')
                nofollow=$(expr "${flag}" : '^nofollow=\(.*\)$');;
              *)
                nofollow="${nofollow}|"$(expr "${flag}" : '^nofollow=\(.*\)$');;
            esac
          ;;
          container)
            container_types="${container_types}|${2?"${usage}"}";;
          container=*)
            container_types="${container_types}|"$(expr "${flag}" : '^container=\(.*\)$');;
          p) ;&
          projects)
            projects_out="${2?"${usage}"}"; shift;;
          p=*) ;&
          projects=*)
            projects_out=$(expr "${flag}" : '^.*=\(.*\)$');;
          d) ;&
          directores)
            directories_out="${2?"${usage}"}"; shift;;
          d=*) ;&
          directories=*)
            directories_out=$(expr "${flag}" : '^.*=\(.*\)$');;
          t) ;&
          targets)
            targets_out="${2?"${usage}"}"; shift;;
          t=*) ;&
          targets=)
            targets_out=$(expr "${flag}" : '^.*=\(.*\)$');;
          n) ;&
          notices)
            notices_out="${2?"${usage}"}"; shift;;
          n=*) ;&
          notices=)
            notices_out=$(expr "${flag}" : '^.*=\(.*\)$');;
          *)
            die "Unknown flag ${1}";;
        esac
      ;;
      *)
        targets+=("${1:-}")
      ;;
    esac
    shift
done

# fail fast if command-line arguments are invalid

if [ ! -v targets[0] ] && ! ${use_stdin}; then
    die "${usage}\n\nNo target specified."
fi

if [ -z "${projects_out}" ] \
  && [ -z "${directories_out}" ] \
  && [ -z "${targets_out}" ] \
  && [ -z "${notices_out}" ]
then
    targets_out='/dev/stdout'
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
function getDeps() {
    (
      tr '\n' '\0' | xargs -0 -r \
          "${ANDROID_BUILD_TOP}/prebuilts/build-tools/linux-x86/bin/ninja" \
          -f "${ANDROID_BUILD_TOP}/out/combined-${TARGET_PRODUCT}.ninja" \
          -t query
    ) | awk -v include_order="${include_order_deps}" \
        -v include_implicit="${include_implicit_deps}" \
        -v include_direct="${include_deps}" \
        -v containers="${container_types}" \
    '
      BEGIN {
        ininput = 0
        isnotice = 0
        currFileName = ""
        currExt = ""
      }
      $1 == "outputs:" {
        ininput = 0
      }
      ininput == 0 && $0 ~ /^\S\S*:$/ {
        isnotice = ($0 ~ /.*NOTICE.*[.]txt:$/)
        currFileName = gensub(/^.*[/]([^/]*)[:]$/, "\\1", "g")
        currExt = gensub(/^.*[.]([^./]*)[:]$/, "\\1", "g")
      }
      ininput != 0 && $1 !~ /^[|][|]?/ {
        if (include_direct == "true") {
          fileName = gensub(/^.*[/]([^/]*)$/, "\\1", "g")
          print ( \
              (isnotice && $0 !~ /^\s*build[/]soong[/]scripts[/]/) \
              || $0 ~ /NOTICE|LICEN[CS]E/ \
              || $0 ~ /(notice|licen[cs]e)[.]txt/ \
          )" "(fileName == currFileName||currExt ~ "^(" containers ")$")" "gensub(/^\s*/, "", "g")
        }
      }
      ininput != 0 && $1 == "|" {
        if (include_implicit == "true") {
          fileName = gensub(/^.*[/]([^/]*)$/, "\\1", "g")
          $1 = ""
          print ( \
              (isnotice && $0 !~ /^\s*build[/]soong[/]scripts[/]/) \
              || $0 ~ /NOTICE|LICEN[CS]E/ \
              || $0 ~ /(notice|licen[cs]e)[.]txt/ \
          )" "(fileName == currFileName||currExt ~ "^(" containers ")$")" "gensub(/^\s*/, "", "g")
        }
      }
      ininput != 0 && $1 == "||" {
        if (include_order == "true") {
          fileName = gensub(/^.*[/]([^/]*)$/, "\\1", "g")
          $1 = ""
          print ( \
              (isnotice && $0 !~ /^\s*build[/]soong[/]scripts[/]/) \
              || $0 ~ /NOTICE|LICEN[CS]E/ \
              || $0 ~ /(notice|licen[cs]e)[.]txt/ \
          )" "(fileName == currFileName||currExt ~ "^(" containers ")$")" "gensub(/^\s*/, "", "g")
        }
      }
      $1 == "input:" {
        ininput = 1
      }
    '
}

# Reads one input directory per line from stdin; outputs unique git projects.
function getProjects() {
    while read d; do
        while [ "${d}" != '.' ] && [ "${d}" != '/' ]; do
            if [ -d "${d}/.git/" ]; then
                echo "${d}"
                break
            fi
            d=$(dirname "${d}")
        done
    done | sort -u
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
      print ( \
          $0 ~ /NOTICE|LICEN[CS]E/ \
          || $0 ~ /(notice|licen[cs]e)[.]txt/ \
      )" "gensub(/\s*$/, "", "g", gensub(/^\s*/, "", "g"))
    }
  ' >"${newDeps}"
else # start with no deps by clearing file
  : >"${newDeps}"
fi

# extend deps by appending targets from command-line
for idx in "${!targets[*]}"; do
    case "${targets[${idx}]}" in
      *NOTICE*) ;&
      *LICEN[CS]E*) ;&
      *notice.txt) ;&
      *licen[cs]e.txt)
        echo "1 1 ${targets[${idx}]}" >>"${newDeps}";;
      *)
        echo "0 1 ${targets[${idx}]}" >>"${newDeps}";;
    esac
done

# remove duplicates and start with new, old and all the same
sort -u <"${newDeps}" >"${allDeps}"
cp "${allDeps}" "${newDeps}"
cp "${allDeps}" "${oldDeps}"

# report depth of dependenciens when showProgress
depth=0

# 1st iteration always unfiltered
filter='cat'
while true; do
    if ${showProgress}; then
        echo "depth ${depth} has "$(cat "${newDeps}" | wc -l)" targets" >&2
        depth=$(expr ${depth} + 1)
    fi
    ( # recalculate dependencies by combining unique inputs of new deps w. old
        sh -c "${filter}" <"${newDeps}" | cut -d\  -f3- | getDeps
        cat "${oldDeps}"
    ) | sort -u >"${allDeps}"
    # recalculate new dependencies as net additions to old dependencies
    if diff "${oldDeps}" "${allDeps}" --old-line-format='' --new-line-format='%L' \
      --unchanged-line-format='' > "${newDeps}"
    then # stop when none found
        break
    fi
    # apply filters on subsequent iterations
    case "${nofollow}" in
      '')
        filter='cat';;
      *)
        filter="egrep -v '^[01] 0 (${nofollow})$'"
      ;;
    esac
    # recalculate old dependencies for next iteration
    cp "${allDeps}" "${oldDeps}"
done

# found all deps -- clean up last iteration of old and new
rm "${oldDeps}"
rm "${newDeps}"

if ${showProgress}; then
    echo $(cat "${allDeps}" | wc -l)" targets" >&2
fi

if [ -n "${targets_out}" ]; then
    cut -d\  -f3- "${allDeps}" | sort -u >"${targets_out}"
fi

if [ -n "${directories_out}" ] \
  || [ -n "${projects_out}" ] \
  || [ -n "${notices_out}" ]
then
    readonly allDirs="${tmpFiles}/dirs"
    (
        cut -d\  -f3- "${allDeps}" | tr '\n' '\0' | xargs -0 dirname
    ) | sort -u >"${allDirs}"
    if ${showProgress}; then
        echo $(cat "${allDirs}" | wc -l)" directories" >&2
    fi

    case "${directories_out}" in
      '')        : do nothing;;
      *)
        cat "${allDirs}" >"${directories_out}"
      ;;
    esac
fi

if [ -n "${projects_out}" ] \
  || [ -n "${notices_out}" ]
then
    readonly allProj="${tmpFiles}/projects"
    egrep -v '^out[/]' "${allDirs}" | getProjects >"${allProj}"
    if ${showProgress}; then
        echo $(cat "${allProj}" | wc -l)" projects" >&2
    fi

    case "${projects_out}" in
      '')        : do nothing;;
      *)
        cat "${allProj}" >"${projects_out}"
      ;;
    esac
fi

case "${notices_out}" in
  '')        : do nothing;;
  *)
    readonly allNotice="${tmpFiles}/notices"
    egrep '^1' "${allDeps}" | cut -d\  -f3- | egrep -v '^out/' >"${allNotice}"
    cat "${allProj}" | while read proj; do
        for f in LICENSE LICENCE NOTICE license.txt notice.txt; do
            if [ -f "${proj}/${f}" ]; then
                echo "${proj}/${f}"
            fi
        done
    done >> "${allNotice}"
    if ${showProgres}; then
      echo $(cat "${allNotice}" | sort -u | wc -l)" notice targets" >&2
    fi
    readonly hashedNotice="${tmpFiles}/hashednotices"
    ( # md5sum outputs checksum space indicator(space or *) filename newline
        sort -u "${allNotice}" | tr '\n' '\0' | xargs -0 -r md5sum 2>/dev/null
      # use sed to replace space and indicator with separator
    ) >"${hashedNotice}"
    if ${showProgress}; then
        echo $(cut -d\  -f2- "${hashedNotice}" | sort -u | wc -l)" notice files" >&2
        echo $(cut -d\  -f1 "${hashedNotice}" | sort -u | wc -l)" distinct notices" >&2
    fi
    sed 's/^\([^ ]*\) [* ]/\1'"${sep}"'/g' "${hashedNotice}" | sort >"${notices_out}"
  ;;
esac

if ${interactive}; then
    echo -n "`date '+%F %-k:%M:%S'` Delete ${tmpFiles} ? [n] " >&2
    read answer
    case "${answer}" in [yY]*) rm -fr "${tmpFiles}";; esac
else
    rm -fr "${tmpFiles}"
fi

#!/bin/bash

set -eu

# This file generates the project.xml and lint.xml files used to drive the Android Lint CLI tool.
# It takes lists of sources in files to avoid command line length limits.

function die() {
  echo "$@" >&2
  exit 1
}

function usage() {
  violation="$1"
  die "$violation

  Usage: $0 [options]

  This file generates project.xml and lint.xml files used to drive the Android Lint CLI tool.

  Options:
    --project_out FILE       file to which the project.xml contents will be written.
    --config_out FILE        file to which the lint.xml contents will be written.
    --name NAME              name of the module.
    --srcs FILE              file containing whitespace separated list of source files.
    --generated_srcs FILE    file containing whitespace separated list of generated source files.
    --resources FILE         file containing resources.
    --classpath FILE         file containing classes from dependencies.
    --extra_checks_jar FILE  file containing extra lint checks.
    --manifest FILE          file containing the module's manifest.
    --merged_manifest FILE   file containing merged manifest for the module and its dependencies.
    --library                mark the module as a library
    --test                   mark the module as a test
    --cache_dir DIR          directory to use for cached files
    --error_check            treat a lint issue as an error
    --warning_check          treat a lint issue as a warning
    --disable_check          disable a lint issue
  "
}

arg=""
require_arg() {
  # TODO: handle --arg=value too?
  if [ -n "$2" ] && [ "${2:0:1}" != "-" ]; then
    arg=$2
  else
    usage "Error: Argument for $1 is missing"
  fi
}

PROJECT_OUT=""
CONFIG_OUT=""
NAME=""
SRCS=()
GENERATED_SRCS=()
RESOURCES=()
CLASSPATH=()
EXTRA_CHECKS_JARS=()
MANIFEST=""
MERGED_MANIFEST=""
LIBRARY=""
TEST=""
CACHE_DIR=""

ERROR_CHECKS=()
WARNING_CHECKS=()
DISABLE_CHECKS=()

while (( "$#" )); do
  case "$1" in
    --project_out)
      require_arg "$1" "$2"
      PROJECT_OUT=$arg
      shift 2
      ;;
    --config_out)
      require_arg "$1" "$2"
      CONFIG_OUT=$arg
      shift 2
      ;;
    --name)
      require_arg "$1" "$2"
      NAME=$arg
      shift 2
      ;;
    --srcs)
      require_arg "$1" "$2"
      SRCS+=("$arg")
      shift 2
      ;;
    --generated_srcs)
      require_arg "$1" "$2"
      GENERATED_SRCS+=("$arg")
      shift 2
      ;;
    --resources)
      require_arg "$1" "$2"
      RESOURCES+=("$arg")
      shift 2
      ;;
    --classpath)
      require_arg "$1" "$2"
      CLASSPATH+=("$arg")
      shift 2
      ;;
    --extra_checks_jars)
      require_arg "$1" "$2"
      EXTRA_CHECKS_JARS+=("$arg")
      shift 2
      ;;
    --manifest)
      require_arg "$1" "$2"
      MANIFEST=$arg
      shift 2
      ;;
    --merged_manifest)
      require_arg "$1" "$2"
      MERGED_MANIFEST=$arg
      shift 2
      ;;
    --library)
      LIBRARY="library='true' "
      shift 1
      ;;
    --test)
      TEST="test='true' "
      shift 1
      ;;
    --cache_dir)
      require_arg "$1" "$2"
      CACHE_DIR=$arg
      shift 2
      ;;
    --error_checks)
      require_arg "$1" "$2"
      ERROR_CHECKS+=("$arg")
      shift 2
      ;;
    --warning_checks)
      require_arg "$1" "$2"
      WARNING_CHECKS+=("$arg")
      shift 2
      ;;
    --disable_checks)
      require_arg "$1" "$2"
      DISABLE_CHECKS+=("$arg")
      shift 2
      ;;
    -*) # unsupported flags
      usage "Error: Unsupported flag $1"
      ;;
    *) # unsupported positional arguments
      usage "Error: Unsupported flag $1"
      ;;
  esac
done

if [ -n "${PROJECT_OUT}" ]; then
  if [ -z "${NAME}" ]; then
    usage "--name is required if --project_out is specified"
  fi

  rm -f "${PROJECT_OUT}"
  (
    echo "<?xml version='1.0' encoding='utf-8'?>"
    echo "<project>"
    echo "  <module name='${NAME}' android='true' ${LIBRARY}desugar='full' >"
    if [ -n "${MANIFEST}" ]; then
      echo "    <manifest file='${MANIFEST}' ${TEST}/>"
    fi
    if [ -n "${MERGED_MANIFEST}" ]; then
      echo "    <merged-manifest file='${MERGED_MANIFEST}' ${TEST}/>"
    fi
    for src_file in "${SRCS[@]}"; do
      for src in $(<${src_file}); do
        echo "    <src file='${src}' ${TEST}/>"
      done
    done
    for src_file in "${GENERATED_SRCS[@]}"; do
      for src in $(<"${src_file}"); do
        echo "    <src file='${src}' generated='true' ${TEST}/>"
      done
    done
    for res in "${RESOURCES[@]}"; do
      echo "    <resource file='${res}' ${TEST}/>"
    done
    for classpath in "${CLASSPATH[@]}"; do
      echo "    <classpath jar='${classpath}' />"
    done
    for extra in "${EXTRA_CHECKS_JARS[@]}"; do
      echo "    <lint-checks jar='${extra}' />"
    done
    if [ -n "${CACHE_DIR}" ]; then
      echo "    <cache dir='${CACHE_DIR}'/>"
    fi
    echo "  </module>"
    echo "</project>"
  ) > "${PROJECT_OUT}"
fi

if [ -n "${CONFIG_OUT}" ]; then
  rm -f "${CONFIG_OUT}"
  (
    echo "<?xml version='1.0' encoding='utf-8'?>"
    echo "<lint>"
    for check in "${ERROR_CHECKS[@]}"; do
      echo "  <issue id='${check}' severity='error' />"
    done
    for check in "${WARNING_CHECKS[@]}"; do
      echo "  <issue id='${check}' severity='warning' />"
    done
    for check in "${DISABLE_CHECKS[@]}"; do
      echo "  <issue id='${check}' severity='ignore' />"
    done
    echo "</lint>"
  ) > "${CONFIG_OUT}"
fi

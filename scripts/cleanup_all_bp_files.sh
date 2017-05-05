#!/bin/bash

#terminate on error
set -e

#set some paths
cd "$(dirname $0)"
cd ../../..
ANDROID_PATH="${PWD}"
CONVERTER_PATH="${ANDROID_PATH}/out/soong/host/linux-x86/bin/bpfmt"

function buildConverter() {
  cd "$ANDROID_PATH"
  make -j blueprint_tools
}

function convertPath() {
  inputFilePath="$1"
  inputDirectory="$(dirname $inputFilePath)"
  tempPath="${inputDirectory}/Android.bp.reformatted"
  outputPath="${inputDirectory}/Android.bp"
  echo "Reformatting ${inputFilePath}"
  if "${CONVERTER_PATH}" "${inputFilePath}" > "${tempPath}"; then
    mv "${tempPath}" "${outputPath}"
    return 0
  else
    return 1
  fi
}

function convertAll() {
  echo "Reformatting every Android.bp under ${ANDROID_PATH}"
  failures=""

  buildConverter
  for filePath in `find "${ANDROID_PATH}" -name Android.bp`; do
    if convertPath "${filePath}"; then
      echo "Converted ${filePath}"
    else
      failures="${failures}
${filePath}"
    fi
  done

  if [ "" == "${failures}" ]; then
    echo
    echo "All conversions succeeded"
  else
    echo "********************************************************************************"
    echo "*                                  FAILURES:                                   *"
    echo "********************************************************************************"
    echo "${failures}"
    echo "********************************************************************************"
    echo "*                              FAILURE DETAILS:                                *"
    echo "********************************************************************************"
    echo

    #Retry each failure so their errors are easily findable at the bottom of the list. Really we should instead save the error text and just repeat it here, but this is a throwaway script and it's not much slower to rerun a few failed conversions

    for filePath in ${failures}; do
      echo
      echo "********************************************************************************"
      convertPath "${filePath}" || true
    done
    echo end of failures
  fi
}

convertAll

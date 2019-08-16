#!/bin/bash -uv
#
# Build kzip files (source files for the indexing pipeline) for the given configuration,
# merge them and place the resulting all.kzip into $DIST_DIR.
# It is assumed that the current directory is the top of the source tree.
# The following enviromnet variables affect the result:
#   TARGET_PRODUCT        target device name, e.g., `aosp_blueline`
#   TARGET_BUILD_VARIANT  variant, e.g., `userdebug`
#   OUT_DIR               where the build is happening (./out if not specified)
#   DIST_DIR              where the resulting all.kzip will be placed
#   XREF_CORPUS           source code repository URI, e.g.,
#                        `android.googlesource.com/platform/superproject`

# The extraction might fail for some source files, so run with -k
build/soong/soong_ui.bash --build-mode --all-modules --dir=$PWD -k merge_zips xref_cxx xref_java

OUT_DIR=${OUT_DIR:-out}
DIST_DIR=${DIST_DIR:-$OUT_DIR/dist_dir/}

declare -r allkzip=all.kzip
declare -r zip_files_tmp=$(mktemp)
declare -r kzip_count=$( \
	find "${OUT_DIR}" -name "*.kzip" ! -name "${allkzip}" \
	| tee "${zip_files_tmp}" \
	| wc -l)

echo "Creating merged zips file"
printf "%-14s : %s\n"                     \
	"OUT_DIR"      "${OUT_DIR}"       \
	"DIST_DIR"     "${DIST_DIR}"      \
	"Zip files #"  "${kzip_count}"    \
	"Tmp zip file" "${zip_files_tmp}" \
	"Output zip"   "${DIST_DIR}/${allkzip}"

# We build with -k, so check that we have generated at least 100K files
# (the actual number is 180K+)
if [ "${kzip_count}" -lt "100000" ]; then
	echo "Too few kzip files were generated: ${kzip_count}."
	echo "At least 100000 kzip files are required."
	exit 1
fi

mkdir -p "${DIST_DIR}"
rm -f "${DIST_DIR}/${allkzip}"

# Pack
# TODO(asmundak): this should be done by soong.
"${OUT_DIR}/soong/host/linux-x86/bin/merge_zips" "${DIST_DIR}/${allkzip}" @${zip_files_tmp}

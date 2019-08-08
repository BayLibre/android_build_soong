# /bin/bash -uv
# Build kzip files (source files for the indexing pipeline) for the given configuration,
# merge them and place the resulting all.kzip into $DIST_DIR.
# The default configuration is 'userdebug' on 'aosp_blueline'.

# NOTE(asmundak): Soong generates additional Ninja rules when told to generate kzip files
# (i.e., if $XREF_CORPUS is set). It cannot share output directory with the build that does
# not renerate kzip files, even if the rest of the configuration is the same.
build/soong/soong_ui.bash --build-mode --all-modules --dir=$PWD -k \
  TARGET_PRODUCT="${TARGET_PRODUCT:-aosp_blueline}" \
  TARGET_BUILD_VARIANT="${TARGET_BUILD_VARIANT:-userdebug}" \
  "${OUT_DIR:+OUT_DIR=${OUT_DIR}}" \
  DIST_DIR="${DIST_DIR}" \
  XREF_CORPUS="${XREF_CORPUS:-android.googlesource.com/platform/superproject}" \
  xref_cxx xref_java merge_zips
declare -r kzip_count=$(find $OUT_DIR -name '*.kzip' | wc -l)
(($kzip_count>100000)) || { printf "Too few kzip files were generated: %d\n" $kzip_count; exit 1; }

# Pack
declare -r allkzip=all.kzip
"$OUT_DIR/soong/host/linux-x86/bin/merge_zips" "$DIST_DIR/$allkzip" @<(find $OUT_DIR -name '*.kzip')


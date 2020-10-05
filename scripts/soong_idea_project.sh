#!/usr/bin/env bash

# Create IntelliJ Idea project for Soong development.
# The project consists of three modules:
#   blueprint (source files in build/blueprint)
#   soong (source files in build/soong)
#   protobuf (source files in external/golang-protobuf)
# It works without BindFS and keeps IDEA project files away from
# Android Git repositories.
# Usage:
#   soong_idea_project.sh <ideadir>
# The current directory when running the script should be the
# top of the Android source tree.
# <ideadir> should not exist before the script is run. The script
# creates it and writes project files for Idea there.

function die() {
  printf "$@"
  exit 2
}

function dir_should_exist() {
  for i in "$@"; do
    [[ -d "$i" ]] || die "${PWD} is not an Android source tree, $i directory is missing\n"
  done
}

function write_iml() {
cat >"${project_dir}/$1.iml" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<module type="WEB_MODULE" version="4">
  <component name="Go" enabled="true" />
  <component name="NewModuleRootManager" inherit-compiler-output="true">
    <exclude-output />
    <content url="file://$3/$2" />
    <orderEntry type="sourceFolder" forTests="false" />
  </component>
</module>
EOF
}

set -eu
(($#==1)) || die "Usage: $0 <new project directory>\n"
declare -r project_dir="$1"
[[ ! -e "${project_dir}" ]] || die "%s already exists\n" "${project_dir}"
dir_should_exist build/soong build/blueprint external/golang-protobuf art/build
mkdir -p "${project_dir}"

# Replace home dir path with '$USER_HOME$' in the source tree path.
declare -r root_path="${PWD/#$HOME/\$USER_HOME\$}"

# For each module, write <modulename>.iml file.
write_iml blueprint build/blueprint "${root_path}"
write_iml soong build/soong "${root_path}"
write_iml protobuf external/golang-protobuf "${root_path}"
write_iml art art/build "${root_path}"
write_iml llvm external/llvm/soong "${root_path}"
write_iml clang external/clang/soong "${root_path}"
write_iml clangprebuilts prebuilts/clang/host/linux-x86/soong "${root_path}"
write_iml aidl system/tools/aidl/build "${root_path}"
write_iml hidl system/tools/hidl/build "${root_path}"

# Files in .idea/ subdirectory of the project directory: misc.xml, modules.xml and vcs.xml
mkdir -p "${project_dir}/.idea"
cat >"${project_dir}/.idea/misc.xml" << "EOF"
<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectRootManager">
    <output url="file://$PROJECT_DIR$/out" />
  </component>
</project>
EOF
cat >"${project_dir}/.idea/modules.xml" << "EOF"
<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectModuleManager">
    <modules>
      <module fileurl="file://$PROJECT_DIR$/blueprint.iml" filepath="$PROJECT_DIR$/blueprint.iml" />
      <module fileurl="file://$PROJECT_DIR$/protobuf.iml" filepath="$PROJECT_DIR$/protobuf.iml" />
      <module fileurl="file://$PROJECT_DIR$/soong.iml" filepath="$PROJECT_DIR$/soong.iml" />
      <module fileurl="file://$PROJECT_DIR$/art.iml" filepath="$PROJECT_DIR$/art.iml" />
      <module fileurl="file://$PROJECT_DIR$/llvm.iml" filepath="$PROJECT_DIR$/llvm.iml" />
      <module fileurl="file://$PROJECT_DIR$/clang.iml" filepath="$PROJECT_DIR$/clang.iml" />
      <module fileurl="file://$PROJECT_DIR$/clangprebuilts.iml" filepath="$PROJECT_DIR$/clangprebuilts.iml" />
      <module fileurl="file://$PROJECT_DIR$/aidl.iml" filepath="$PROJECT_DIR$/aidl.iml" />
      <module fileurl="file://$PROJECT_DIR$/hidl.iml" filepath="$PROJECT_DIR$/hidl.iml" />
    </modules>
  </component>
</project>
EOF
cat >"${project_dir}/.idea/vcs.xml" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="VcsDirectoryMappings">
    <mapping directory="${root_path}/build/blueprint" vcs="Git" />
    <mapping directory="${root_path}/build/soong" vcs="Git" />
    <mapping directory="${root_path}/external/golang-protobuf" vcs="Git" />
    <mapping directory="${root_path}/art" vcs="Git" />
    <mapping directory="${root_path}/system/tools/aidl" vcs="Git" />
    <mapping directory="${root_path}/system/tools/hidl" vcs="Git" />
    <mapping directory="${root_path}/external/llvm" vcs="Git" />
    <mapping directory="${root_path}/external/clang" vcs="Git" />
    <mapping directory="${root_path}/prebuilts/clang/host/linux-x86" vcs="Git" />
  </component>
</project>
EOF

# Most of the workspace settings Intellij IDEA will take care of,
# but GOROOT and the usage of Vgo need to be specified explicitly.
cat >"${project_dir}/.idea/workspace.xml" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="GOROOT" path="${root_path}/prebuilts/go/linux-x86" />
  <component name="VgoProject">
    <integration-enabled>true</integration-enabled>
    <proxy>direct</proxy>
  </component>
</project>
EOF

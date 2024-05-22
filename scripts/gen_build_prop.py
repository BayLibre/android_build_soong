#!/usr/bin/env python3
#
# Copyright (C) 2024 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
"""A tool for generating {partition}/build.prop"""

import argparse
import contextlib
import json
import subprocess
import sys

def parse_args():
  """Parse commandline arguments."""
  parser = argparse.ArgumentParser()
  parser.add_argument("--build-fingerprint-file", required=True, type=argparse.FileType("r"))
  parser.add_argument("--build-hostname-file", required=True, type=argparse.FileType("r"))
  parser.add_argument("--build-number-file", required=True, type=argparse.FileType("r"))
  parser.add_argument("--build-thumbprint-file", type=argparse.FileType("r"))
  parser.add_argument("--date-file", required=True, type=argparse.FileType("r"))
  parser.add_argument("--platform-preview-sdk-fingerprint-file", required=True, type=argparse.FileType("r"))
  parser.add_argument("--prop-files", action="append", type=argparse.FileType("r"), default=[])
  parser.add_argument("--build-info", required=True, type=argparse.FileType("r"))
  parser.add_argument("--partition", required=True)
  parser.add_argument("--build-broken-dup-sysprop", action="store_true", default=False)

  parser.add_argument("--out", required=True, type=argparse.FileType("w"))

  args = parser.parse_args()

  # post process parse_args requiring manual handling
  args.info = json.load(args.build_info)
  info = args.info

  info['BuildFingerprint'] = args.build_fingerprint_file.read().strip()
  info['BuildHostname'] = args.build_hostname_file.read().strip()
  info['BuildNumber'] = args.build_number_file.read().strip()
  info['BuildVersionTags'] = info['BuildKeys']
  if info['BuildType'] == "debug":
    info['BuildVersionTags'] = "debug," + info['BuildVersionTags']

  raw_date = args.date_file.read().strip()
  info['Date'] = subprocess.check_output(["date", "-d", f"@{raw_date}"], text=True).strip()
  info['DateUtc'] = subprocess.check_output(["date", "-d", f"@{raw_date}", "+%s"], text=True).strip()

  # build_desc is human readable strings that describe this build. This has the same info as the
  # build fingerprint.
  # e.g. "aosp_cf_x86_64_phone-userdebug VanillaIceCream MAIN eng.20240319.143939 test-keys"
  info['BuildDesc'] = f"{info['Product']}-{info['BuildVariant']} {info['PlatformVersion']} " \
                       f"{info['BuildId']} {info['BuildNumber']} {info['BuildVersionTags']}"

  info['PlatformPreviewSdkFingerprint'] = args.platform_preview_sdk_fingerprint_file.read().strip()

  if args.build_thumbprint_file:
    info['BuildThumbprint'] = args.build_thumbprint_file.read().strip()

  append_additional_system_props(args)
  append_additional_vendor_props(args)
  append_additional_product_props(args)

  return args

def generate_common_build_props(args):
  print("####################################")
  print("# from generate_common_build_props")
  print("# These properties identify this partition image.")
  print("####################################")

  info = args.info
  partition = args.partition

  if partition == "system":
    print(f"ro.product.{partition}.brand={info['SystemBrand']}")
    print(f"ro.product.{partition}.device={info['SystemDevice']}")
    print(f"ro.product.{partition}.manufacturer={info['SystemManufacturer']}")
    print(f"ro.product.{partition}.model={info['SystemModel']}")
    print(f"ro.product.{partition}.name={info['SystemName']}")
  else:
    print(f"ro.product.{partition}.brand={info['Brand']}")
    print(f"ro.product.{partition}.device={info['Device']}")
    print(f"ro.product.{partition}.manufacturer={info['Manufacturer']}")
    print(f"ro.product.{partition}.model={info['Model']}")
    print(f"ro.product.{partition}.name={info['Product']}")

  if partition != "system":
    if info['ModelForAttestation']:
        print(f"ro.product.model_for_attestation={info['ModelForAttestation']}")
    if info['BrandForAttestation']:
        print(f"ro.product.brand_for_attestation={info['BrandForAttestation']}")
    if info['NameForAttestation']:
        print(f"ro.product.name_for_attestation={info['NameForAttestation']}")
    if info['DeviceForAttestation']:
        print(f"ro.product.device_for_attestation={info['DeviceForAttestation']}")
    if info['ManufacturerForAttestation']:
        print(f"ro.product.manufacturer_for_attestation={info['ManufacturerForAttestation']}")

  if info['ZygoteForce64']:
    if partition == "vendor":
      print(f"ro.{partition}.product.cpu.abilist={info['CpuAbiList64']}")
      print(f"ro.{partition}.product.cpu.abilist32=")
      print(f"ro.{partition}.product.cpu.abilist64={info['CpuAbiList64']}")
  else:
    if partition == "system" or partition == "vendor" or partition == "odm":
      print(f"ro.{partition}.product.cpu.abilist={info['CpuAbiList']}")
      print(f"ro.{partition}.product.cpu.abilist32={info['CpuAbiList32']}")
      print(f"ro.{partition}.product.cpu.abilist64={info['CpuAbiList64']}")

  print(f"ro.{partition}.build.date={info['Date']}")
  print(f"ro.{partition}.build.date.utc={info['DateUtc']}")
  # Allow optional assignments for ARC forward-declarations (b/249168657)
  # TODO: Remove any tag-related inconsistencies once the goals from
  # go/arc-android-sigprop-changes have been achieved.
  print(f"ro.{partition}.build.fingerprint?={info['BuildFingerprint']}")
  print(f"ro.{partition}.build.id?={info['BuildId']}")
  print(f"ro.{partition}.build.tags?={info['BuildVersionTags']}")
  print(f"ro.{partition}.build.type={info['BuildVariant']}")
  print(f"ro.{partition}.build.version.incremental={info['BuildNumber']}")
  print(f"ro.{partition}.build.version.release={info['PlatformVersionLastStable']}")
  print(f"ro.{partition}.build.version.release_or_codename={info['PlatformVersion']}")
  print(f"ro.{partition}.build.version.sdk={info['PlatformSdkVersion']}")

def generate_build_info(args):
  print()
  print("####################################")
  print("# from gen_build_prop.py:generate_build_info")
  print("####################################")
  print("# begin build properties")

  info = args.info

  # The ro.build.id will be set dynamically by init, by appending the unique vbmeta digest.
  if info['UseVbmetaDigestInFingerprint']:
    print(f"ro.build.legacy.id={info['BuildId']}")
  else:
    print(f"ro.build.id?={info['BuildId']}")

  # ro.build.display.id is shown under Settings -> About Phone
  if info['BuildVariant'] == "user":
    # User builds should show:
    # release build number or branch.buld_number non-release builds

    # Dev. branches should have DISPLAY_BUILD_NUMBER set
    if info['DisplayBuildNumber']:
      print(f"ro.build.display.id?={info['BuildId']} {info['BuildNumber']} {info['BuildKeys']}")
    else:
      print(f"ro.build.display.id?={info['BuildId']} {info['BuildKeys']}")
  else:
    # Non-user builds should show detailed build information (See build desc above)
    print(f"ro.build.display.id?={info['BuildDesc']}")
  print(f"ro.build.version.incremental={info['BuildNumber']}")
  print(f"ro.build.version.sdk={info['PlatformSdkVersion']}")
  print(f"ro.build.version.preview_sdk={info['PlatformPreviewSdkVersion']}")
  print(f"ro.build.version.preview_sdk_fingerprint={info['PlatformPreviewSdkFingerprint']}")
  print(f"ro.build.version.codename={info['PlatformVersionCodename']}")
  print(f"ro.build.version.all_codenames={','.join(info['PlatformVersionAllCodenames'])}")
  print(f"ro.build.version.known_codenames={info['PlatformVersionKnownCodenames']}")
  print(f"ro.build.version.release={info['PlatformVersionLastStable']}")
  print(f"ro.build.version.release_or_codename={info['PlatformVersion']}")
  print(f"ro.build.version.release_or_preview_display={info['PlatformDisplayVersion']}")
  print(f"ro.build.version.security_patch={info['PlatformSecurityPatch']}")
  print(f"ro.build.version.base_os={info['PlatformBaseOs']}")
  print(f"ro.build.version.min_supported_target_sdk={info['PlatformMinSupportedTargetSdkVersion']}")
  print(f"ro.build.date={info['Date']}")
  print(f"ro.build.date.utc={info['DateUtc']}")
  print(f"ro.build.type={info['BuildVariant']}")
  print(f"ro.build.user={info['BuildUsername']}")
  print(f"ro.build.host={info['BuildHostname']}")
  # TODO: Remove any tag-related optional property declarations once the goals
  # from go/arc-android-sigprop-changes have been achieved.
  print(f"ro.build.tags?={info['BuildVersionTags']}")
  # ro.build.flavor are used only by the test harness to distinguish builds.
  # Only add _asan for a sanitized build if it isn't already a part of the
  # flavor (via a dedicated lunch config for example).
  print(f"ro.build.flavor={info['BuildFlavor']}")

  # These values are deprecated, use "ro.product.cpu.abilist"
  # instead (see below).
  print(f"# ro.product.cpu.abi and ro.product.cpu.abi2 are obsolete,")
  print(f"# use ro.product.cpu.abilist instead.")
  print(f"ro.product.cpu.abi={info['CpuAbis'][0]}")
  if len(info['CpuAbis']) > 1:
    print(f"ro.product.cpu.abi2={info['CpuAbis'][1]}")

  if info['DefaultLocale']:
    print(f"ro.product.locale={info['DefaultLocale']}")
  print(f"ro.wifi.channels={' '.join(info['DefaultWifiChannels'])}")

  print(f"# ro.build.product is obsolete; use ro.product.device")
  print(f"ro.build.product={info['Device']}")

  print(f"# Do not try to parse description or thumbprint")
  print(f"ro.build.description?={info['BuildDesc']}")
  if 'build_thumbprint' in info:
    print(f"ro.build.thumbprint={info['BuildThumbprint']}")

  print(f"# end build properties")

def write_properties_from_file(file):
  print()
  print("####################################")
  print(f"# from {file.name}")
  print("####################################")
  print(file.read(), end='')

def write_properties_from_variable(name, props, build_broken_dup_sysprop):
  print()
  print("####################################")
  print(f"# from variable {name}")
  print("####################################")

  # Implement the legacy behavior when BUILD_BROKEN_DUP_SYSPROP is on.
  # Optional assignments are all converted to normal assignments and
  # when their duplicates the first one wins.
  if build_broken_dup_sysprop:
    processed_props = []
    seen_props = set()
    for line in props:
      line = line.replace("?=", "=")
      key, value = line.split("=", 1)
      if key in seen_props:
        continue
      seen_props.add(key)
      processed_props.append(line)
    props = processed_props

  for line in props:
    print(line)

def append_additional_system_props(args):
  props = []

  info = args.info

  # Add the product-defined properties to the build properties.
  if info['PropertySplitEnabled'] or info['VendorImageFileSystemType']:
    props += info['PropVariables']["PRODUCT_PROPERTY_OVERRIDES"]

  props.append(f"ro.treble.enabled={'true' if info['FullTreble'] else 'false'}")
  # Set ro.llndk.api_level to show the maximum vendor API level that the LLNDK
  # in the system partition supports.
  if info['BoardApiLevel']:
    props.append(f"ro.llndk.api_level={info['BoardApiLevel']}")

  # Sets ro.actionable_compatible_property.enabled to know on runtime whether
  # the allowed list of actionable compatible properties is enabled or not.
  props.append("ro.actionable_compatible_property.enabled=true")

  # Enable core platform API violation warnings on userdebug and eng builds.
  if info['BuildVariant'] != "user":
    props.append("persist.debug.dalvik.vm.core_platform_api_policy=just-warn")

  # Define ro.sanitize.<name> properties for all global sanitizers.
  for sanitize_target in info['SanitizeTargets']:
    props.append(f"ro.sanitize.{sanitize_target}=true")

  # Sets the default value of ro.postinstall.fstab.prefix to /system.
  # Device board config should override the value to /product when needed by:
  #
  #     PRODUCT_PRODUCT_PROPERTIES += ro.postinstall.fstab.prefix=/product
  #
  # It then uses ${ro.postinstall.fstab.prefix}/etc/fstab.postinstall to
  # mount system_other partition.
  props.append("ro.postinstall.fstab.prefix=/system")

  enable_target_debugging = True
  if info['BuildVariant'] == "user" or info['BuildVariant'] == "userdebug":
    # Target is secure in user builds.
    props.append("ro.secure=1")
    props.append("security.perf_harden=1")

    if info['BuildVariant'] == "user":
      # Disable debugging in plain user builds.
      props.append("ro.adb.secure=1")
      enable_target_debugging = False

    # Disallow mock locations by default for user builds
    props.append("ro.allow.mock.location=0")
  else:
    # Turn on checkjni for non-user builds.
    props.append("ro.kernel.android.checkjni=1")
    # Set device insecure for non-user builds.
    props.append("ro.secure=0")
    # Allow mock locations by default for non user builds
    props.append("ro.allow.mock.location=1")

  if enable_target_debugging:
    # Target is more debuggable and adbd is on by default
    props.append("ro.debuggable=1")
    # Enable Dalvik lock contention logging.
    props.append("dalvik.vm.lockprof.threshold=500")
  else:
    # Target is less debuggable and adbd is off by default
    props.append("ro.debuggable=0")

  if info['BuildVariant'] == "eng":
    if "ro.setupwizard.mode=ENABLED" in props:
      # Don't require the setup wizard on eng builds
      props = list(filter(lambda x: not x.startswith("ro.setupwizard.mode="), props))
      props.append("ro.setupwizard.mode=OPTIONAL")

    if not info['SdkBuild']:
      # To speedup startup of non-preopted builds, don't verify or compile the boot image.
      props.append("dalvik.vm.image-dex2oat-filter=extract")
    # b/323566535
    props.append("init.svc_debug.no_fatal.zygote=true")

  if info['SdkBuild']:
    props.append("xmpp.auto-presence=true")
    props.append("ro.config.nocheckin=yes")

  props.append("net.bt.name=Android")

  # This property is set by flashing debug boot image, so default to false.
  props.append("ro.force.debuggable=0")

  info['PropVariables']["ADDITIONAL_SYSTEM_PROPERTIES"] = props

def append_additional_vendor_props(args):
  props = []

  info = args.info

  # Add cpu properties for bionic and ART.
  props.append(f"ro.bionic.arch={info['Arch']}")
  props.append(f"ro.bionic.cpu_variant={info['ArchVariantRuntime']}")
  props.append(f"ro.bionic.2nd_arch={info['SecondaryArch']}")
  props.append(f"ro.bionic.2nd_cpu_variant={info['SecondaryArchVariantRuntime']}")

  props.append(f"persist.sys.dalvik.vm.lib.2=libart.so")
  props.append(f"dalvik.vm.isa.{info['Arch']}.variant={info['Dex2oatTargetCpuVariantRuntime']}")
  if info['Dex2oatTargetInstructionSetFeatures']:
    props.append(f"dalvik.vm.isa.{info['Arch']}.features={info['Dex2oatTargetInstructionSetFeatures']}")

  if info['SecondaryArch']:
    props.append(f"dalvik.vm.isa.{info['SecondaryArch']}.variant={info['SecondaryDex2oatCpuVariantRuntime']}")
    if info['SecondaryDex2oatInstructionSetFeatures']:
      props.append(f"dalvik.vm.isa.{info['SecondaryArch']}.features={info['SecondaryDex2oatInstructionSetFeatures']}")

  # Although these variables are prefixed with TARGET_RECOVERY_, they are also needed under charger
  # mode (via libminui).
  if info['RecoveryDefaultRotation']:
    props.append(f"ro.minui.default_rotation={info['RecoveryDefaultRotation']}")

  if info['RecoveryOverscanPercent']:
    props.append(f"ro.minui.overscan_percent={info['RecoveryOverscanPercent']}")

  if info['RecoveryPixelFormat']:
    props.append(f"ro.minui.pixel_format={info['RecoveryPixelFormat']}")

  if 'UseDynamicPartitions' in info:
    props.append(f"ro.boot.dynamic_partitions={'true' if info['UseDynamicPartitions'] else 'false'}")

  if 'RetrofitDynamicPartitions' in info:
    props.append(f"ro.boot.dynamic_partitions_retrofit={'true' if info['RetrofitDynamicPartitions'] else 'false'}")

  if info['ProductShippingApiLevel']:
    props.append(f"ro.product.first_api_level={info['ProductShippingApiLevel']}")

  if info['ProductShippingVendorApiLevel']:
    props.append(f"ro.vendor.api_level={info['ProductShippingVendorApiLevel']}")

  if info['BuildVariant'] != "user" and info['SetDebugfsRestrictions']:
    props.append(f"ro.product.debugfs_restrictions.enabled=true")

  # Vendors with GRF must define BOARD_SHIPPING_API_LEVEL for the vendor API level.
  # This must not be defined for the non-GRF devices.
  # The values of the GRF properties will be verified by post_process_props.py
  if info['BoardShippingApiLevel']:
    props.append(f"ro.board.first_api_level={info['ProductShippingApiLevel']}")

  # Build system set BOARD_API_LEVEL to show the api level of the vendor API surface.
  # This must not be altered outside of build system.
  if info['BoardApiLevel']:
    props.append(f"ro.board.api_level={info['BoardApiLevel']}")

  # RELEASE_BOARD_API_LEVEL_FROZEN is true when the vendor API surface is frozen.
  if info['BoardApiLevelFrozen']:
    props.append(f"ro.board.api_frozen=true")

  # Set build prop. This prop is read by ota_from_target_files when generating OTA,
  # to decide if VABC should be disabled.
  if info['DontUseVabcOta']:
    props.append(f"ro.vendor.build.dont_use_vabc=true")

  # Set the flag in vendor. So VTS would know if the new fingerprint format is in use when
  # the system images are replaced by GSI.
  if info['UseVbmetaDigestInFingerprint']:
    props.append(f"ro.vendor.build.fingerprint_has_digest=1")

  props.append(f"ro.vendor.build.security_patch={info['VendorSecurityPatch']}")
  props.append(f"ro.product.board={info['BootloaderBoardName']}")
  props.append(f"ro.board.platform={info['BoardPlatform']}")
  props.append(f"ro.hwui.use_vulkan={'true' if info['UsesVulkan'] else 'false'}")

  if info['ScreenDensity']:
    props.append(f"ro.sf.lcd_density={info['ScreenDensity']}")

  if 'AbOtaUpdater' in info:
    props.append(f"ro.build.ab_update={'true' if info['AbOtaUpdater'] else 'false'}")
    if info['AbOtaUpdater']:
      props.append(f"ro.vendor.build.ab_ota_partitions={info['AbOtaPartitions']}")

  info['PropVariables']["ADDITIONAL_VENDOR_PROPERTIES"] = props

def append_additional_product_props(args):
  props = []

  info = args.info

  # Add the system server compiler filter if they are specified for the product.
  if info['SystemServerCompilerFilter']:
    props.append(f"dalvik.vm.systemservercompilerfilter={info['SystemServerCompilerFilter']}")

  # Add the 16K developer args if it is defined for the product.
  props.append(f"ro.product.build.16k_page.enabled={'true' if info['Product16kDeveloperOption'] else 'false'}")

  props.append(f"ro.build.characteristics={info['AaptCharacteristics']}")

  if 'AbOtaUpdater' in info and info['AbOtaUpdater']:
    props.append(f"ro.product.ab_ota_partitions={info['AbOtaPartitions']}")

  # Set this property for VTS to skip large page size tests on unsupported devices.
  props.append(f"ro.product.cpu.pagesize.max={info['MaxPageSizeSupported']}")

  if info['NoBionicPageSizeMacro']:
    props.append(f"ro.product.build.no_bionic_page_size_macro=true")

  # If the value is "default", it will be mangled by post_process_props.py.
  props.append(f"ro.dalvik.vm.enable_uffd_gc={info['EnableUffdGc']}")

  info['PropVariables']["ADDITIONAL_PRODUCT_PROPERTIES"] = props

def build_system_prop(args):
  info = args.info

  # Order matters here. When there are duplicates, the last one wins.
  # TODO(b/117892318): don't allow duplicates so that the ordering doesn't matter
  variables = [
    "ADDITIONAL_SYSTEM_PROPERTIES",
    "PRODUCT_SYSTEM_PROPERTIES",
    # TODO(b/117892318): deprecate this
    "PRODUCT_SYSTEM_DEFAULT_PROPERTIES",
  ]

  if not info['PropertySplitEnabled']:
    variables += [
      "ADDITIONAL_VENDOR_PROPERTIES",
      "PRODUCT_VENDOR_PROPERTIES",
    ]

  build_prop(args, gen_build_info=True, gen_common_build_props=True, variables=variables)

'''
def build_vendor_prop(args):
  info = args.info

  # Order matters here. When there are duplicates, the last one wins.
  # TODO(b/117892318): don't allow duplicates so that the ordering doesn't matter
  variables = []
  if info['PropertySplitEnabled']:
    variables += [
      "ADDITIONAL_VENDOR_PROPERTIES",
      "PRODUCT_VENDOR_PROPERTIES",
      # TODO(b/117892318): deprecate this
      "PRODUCT_DEFAULT_PROPERTY_OVERRIDES",
      "PRODUCT_PROPERTY_OVERRIDES",
    ]

  build_prop(args, gen_build_info=False, gen_common_build_props=True, variables=variables)

def build_product_prop(args):
  info = args.info

  # Order matters here. When there are duplicates, the last one wins.
  # TODO(b/117892318): don't allow duplicates so that the ordering doesn't matter
  variables = [
    "ADDITIONAL_PRODUCT_PROPERTIES",
    "PRODUCT_PRODUCT_PROPERTIES",
  ]
  build_prop(args, gen_build_info=False, gen_common_build_props=True, variables=variables)
'''

def build_prop(args, gen_build_info, gen_common_build_props, variables):
  info = args.info

  if gen_common_build_props:
    generate_common_build_props(args)

  if gen_build_info:
    generate_build_info(args)

  for prop_file in args.prop_files:
    write_properties_from_file(prop_file)

  for variable in variables:
    if variable in info['PropVariables']:
      write_properties_from_variable(variable, info['PropVariables'][variable], args.build_broken_dup_sysprop)

def main():
  args = parse_args()

  with contextlib.redirect_stdout(args.out):
    if args.partition == "system":
      build_system_prop(args)
      '''
    elif args.partition == "vendor":
      build_vendor_prop(args)
    elif args.partition == "product":
      build_product_prop(args)
      '''
    else:
      sys.exit(f"not supported partition {args.partition}")

if __name__ == "__main__":
  main()

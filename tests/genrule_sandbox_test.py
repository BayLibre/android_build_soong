#!/usr/bin/env python3

# Copyright (C) 2023 The Android Open Source Project
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

import argparse
import collections
import json
import os
import subprocess
import sys
import tempfile

def get_top() -> str:
  path = '.'
  while not os.path.isfile(os.path.join(path, 'build/soong/tests/genrule_sandbox_test.py')):
    if os.path.abspath(path) == '/':
      sys.exit('Could not find android source tree root.')
    path = os.path.join(path, '..')
  return os.path.abspath(path)

def _build_with_soong(targets, target_product, *, keep_going = False, extra_env={}):
  env = {
      **os.environ,
      "TARGET_PRODUCT": target_product,
      "TARGET_BUILD_VARIANT": "userdebug",
  }
  env.update(extra_env)
  args = [
      "build/soong/soong_ui.bash",
      "--make-mode",
      "--skip-soong-tests",
  ]
  if keep_going:
    args.append("-k")
  args.extend(targets)
  try:
    subprocess.check_output(
        args,
        env=env,
    )
  except subprocess.CalledProcessError as e:
    print(e)
    print(e.stdout)
    print(e.stderr)
    exit(1)


def _find_outputs_for_modules(modules, out_dir, target_product):
  module_path = os.path.join(out_dir, "soong", "module-actions.json")

  if not os.path.exists(module_path):
    _build_with_soong(["json-module-graph"], target_product)

  with open(module_path) as f:
    action_graph = json.load(f)

  module_to_outs = collections.defaultdict(set)
  for mod in action_graph:
    name = mod["Name"]
    if name in modules:
      for act in mod["Module"]["Actions"]:
        if "}generate" in act["Desc"]:
          module_to_outs[name].update(act["Outputs"])
  return module_to_outs

def _find_outputs_for_all_genrules(out_dir, target_product):
  query_output = subprocess.check_output([
    'prebuilts/build-tools/linux-x86/bin/ninja',
    '-f',
    os.path.join(out_dir, f'combined-{target_product}.ninja'),
    '-t',
    'query',
    'all_genrules'
  ], text=True)

  # the output starts with:
  #
  # all_genrules:
  #   input: phony
  #
  # and ends with:
  #
  # outputs:
  #
  # so cut off those lines
  return [x.strip() for x in query_output.splitlines()[2:-1]]

def _compare_outputs(module_to_outs, tempdir) -> dict[str, list[str]]:
  different_modules = collections.defaultdict(list)
  for module, outs in module_to_outs.items():
    for out in outs:
      try:
        subprocess.check_output(["diff", os.path.join(tempdir, out), out], text=True)
      except subprocess.CalledProcessError as e:
        different_modules[module].append(e.stdout)

  return different_modules

def _compare_outputs_without_module_information(all_outs, tempdir) -> dict[str, str]:
  file_to_diff = {}
  num_no_diffs = 0
  for out in all_outs:
    try:
      subprocess.check_output(["diff", os.path.join(tempdir, out), out], text=True)
      num_no_diffs += 1
    except subprocess.CalledProcessError as e:
      file_to_diff[out] = e.stdout
  print(f"{num_no_diffs} files has no diffs")
  return file_to_diff

def main():
  parser = argparse.ArgumentParser()
  parser.add_argument(
      "--target_product",
      "-t",
      default="aosp_cf_arm64_phone",
      help="optional, target product, always runs as eng",
  )
  parser.add_argument(
      "modules",
      nargs="+",
      help="modules to compare builds with genrule sandboxing enabled/not",
  )
  parser.add_argument(
      "--show-diff",
      "-d",
      action="store_true",
      help="whether to display differing files",
  )
  parser.add_argument(
      "--output-paths-only",
      "-o",
      action="store_true",
      help="Whether to only return the output paths per module",
  )
  args = parser.parse_args()
  os.chdir(get_top())

  modules = set(args.modules)
  all_genrules = False
  if "all" in modules:
    all_genrules = True
    if len(modules) > 1:
      sys.exit("Cannot name other modules along with 'all'")

  out_dir = os.environ.get("OUT_DIR", "out")

  if all_genrules:
    print("finding output files for all genrules...")
    _build_with_soong(['nothing'], args.target_product)
    all_outs = _find_outputs_for_all_genrules(out_dir, args.target_product)
    if not all_outs:
      sys.exit("No outputs found")

    if args.output_paths_only:
      for o in all_outs.items():
        print(o)
      sys.exit(0)
  else:
    print("finding output files for the modules...")
    module_to_outs = _find_outputs_for_modules(set(args.modules), out_dir, args.target_product)
    if not module_to_outs:
      sys.exit("No outputs found")

    if args.output_paths_only:
      for m, o in module_to_outs.items():
        print(f"{m} outputs: {o}")
      sys.exit(0)

    all_outs = list(set.union(*module_to_outs.values()))

  # Using all_outs when building all genrules makes the command line too long
  to_build = ["all_genrules"] if all_genrules else all_outs

  print("building without sandboxing...")
  _build_with_soong(to_build, args.target_product)

  with tempfile.TemporaryDirectory() as tempdir:
    for f in all_outs:
      subprocess.check_call(["cp", "--parents", f, tempdir])

    print("building with sandboxing...")
    _build_with_soong(
        to_build,
        args.target_product,
        # We've verified these build without sandboxing already, so do the sandboxing build
        # with keep_going = True so that we can find all the genrules that fail to build with
        # sandboxing.
        keep_going = True,
        extra_env={"GENRULE_SANDBOXING": "true", "OUT_DIR": "out_sandboxed"},
    )

    if all_genrules:
      diffs = _compare_outputs_without_module_information(all_outs, tempdir)
    else:
      diffs = _compare_outputs(module_to_outs, tempdir)
    if len(diffs) == 0:
      print("All modules are correct")
    elif args.show_diff:
      if all_genrules:
        for file, diff in diffs.items():
          print(f"{file}:")
          print(diff)
      else:
        for m, d in diffs.items():
          print(f"Module {m} has diffs {d}")
    else:
      print(f"Modules/files with diffs:")
      for m in diffs.keys():
        print(m)

if __name__ == "__main__":
  main()

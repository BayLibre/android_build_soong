#!/usr/bin/env python
#
# Copyright (C) 2022 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Unit tests for analyzing bootclasspath_fragment modules."""
import contextlib
import io
import os.path
import shutil
import tempfile
import unittest
import unittest.mock

import analyze_bcpf as ab


class FakeBuildOperation(ab.BuildOperation):

    def __init__(self, lines, return_code):
        ab.BuildOperation.__init__(self, None)
        self._lines = lines
        self.returncode = return_code

    def lines(self):
        return iter(self._lines)

    def wait(self, *args, **kwargs):
        return


class TestAnalyzeBcpf(unittest.TestCase):

    def setUp(self):
        # Create a temporary directory
        self.test_dir = tempfile.mkdtemp()

    def tearDown(self):
        # Remove the directory after the test
        shutil.rmtree(self.test_dir)

    @staticmethod
    def write_abs_file(abs_path, contents):
        os.makedirs(os.path.dirname(abs_path), exist_ok=True)
        with open(abs_path, "w", encoding="utf8") as f:
            print(contents.removeprefix("\n"), file=f, end="")

    def populate_fs(self, fs):
        for path, contents in fs.items():
            abs_path = os.path.join(self.test_dir, path)
            self.write_abs_file(abs_path, contents)

    def create_analyzer_for_test(self,
                                 fs=None,
                                 bcpf="bcpf",
                                 apex="apex",
                                 sdk="sdk"):
        if fs:
            self.populate_fs(fs)

        top_dir = self.test_dir
        out_dir = os.path.join(self.test_dir, "out")
        product_out_dir = "out/product"

        bcpf_dir = f"{bcpf}-dir"
        modules = {bcpf: {"path": [bcpf_dir]}}
        module_info = ab.ModuleInfo(modules)

        analyzer = ab.BcpfAnalyzer(
            top_dir=top_dir,
            out_dir=out_dir,
            product_out_dir=product_out_dir,
            bcpf=bcpf,
            apex=apex,
            sdk=sdk,
            module_info=module_info,
        )
        analyzer.load_all_flags()
        return analyzer

    def test_reformat_report_text(self):
        lines = """
99. An item in a numbered list
that traverses multiple lines.

   An indented example
   that should not be reformatted.
"""
        reformatted = ab.BcpfAnalyzer.reformat_report_test(lines)
        self.assertEqual(
            """
99. An item in a numbered list that traverses multiple lines.

   An indented example
   that should not be reformatted.
""", reformatted)

    def test_build_stub_flags(self):
        lines = """
ERROR: Hidden API flags are inconsistent:
< out/soong/.intermediates/bcpf-dir/bcpf-dir/filtered-flags.csv
> out/soong/hiddenapi/hiddenapi-flags.csv

< Landroid/compat/Compatibility$1;-><init>()V,blocked
> Landroid/compat/Compatibility$1;-><init>()V,max-target-q
16:37:32 ninja failed with: exit status 1
""".strip().splitlines()
        operation = FakeBuildOperation(lines=lines, return_code=1)

        fs = {
            "frameworks/base/boot/hiddenapi/hiddenapi-max-target-q.txt":
                """
Landroid/compat/Compatibility$1;-><init>()V
""",
            "out/soong/.intermediates/bcpf-dir/bcpf/all-flags.csv":
                """
Landroid/compat/Compatibility$1;-><init>()V,blocked
""",
        }

        analyzer = self.create_analyzer_for_test(fs)

        # Override the build_file_read_output() method to just return a fake
        # build operation.
        analyzer.build_file_read_output = unittest.mock.Mock(
            return_value=operation)

        # Override the run_command() method to do nothing.
        analyzer.run_command = unittest.mock.Mock()

        with io.StringIO() as buf, contextlib.redirect_stdout(buf):
            diffs, property_snippet = analyzer.build_monolithic_flags()
            expected_diffs = {
                "Landroid/compat/Compatibility$1;-><init>()V":
                    (["blocked"], ["max-target-q"])
            }
            self.assertEqual(expected_diffs, diffs, msg="flag differences")

            expected_property_snippet = """
        max_target_q: ["hiddenapi/hiddenapi-max-target-q.txt"],
""".strip("\n")
            self.assertEqual(
                expected_property_snippet,
                property_snippet,
                msg="hiddenapi snippet")

    def test_compute_hiddenapi_package_properties(self):
        fs = {
            "out/soong/.intermediates/bcpf-dir/bcpf/all-flags.csv":
                """
La/b/C;->m()V
La/b/c/D;->m()V
La/b/c/E;->m()V
Lb/c/D;->m()V
Lb/c/E;->m()V
Lb/c/d/E;->m()V
""",
            "out/soong/hiddenapi/hiddenapi-flags.csv":
                """
La/b/C;->m()V
La/b/D;->m()V
La/b/E;->m()V
La/b/c/D;->m()V
La/b/c/E;->m()V
La/b/c/d/E;->m()V
Lb/c/D;->m()V
Lb/c/E;->m()V
Lb/c/d/E;->m()V
"""
        }
        analyzer = self.create_analyzer_for_test(fs)
        analyzer.load_all_flags()

        split_packages, single_packages, package_prefixes = \
            analyzer.compute_hiddenapi_package_properties()
        self.assertEqual(["a.b"], split_packages)
        self.assertEqual(["a.b.c"], single_packages)
        self.assertEqual(["b"], package_prefixes)


if __name__ == "__main__":
    unittest.main(verbosity=3)

# Copyright 2024 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from importlib import resources
import pathlib
import shutil
import subprocess
import tempfile
import unittest


class EnableFeatureBasedOnRolloutPercentageTest(unittest.TestCase):

  def setUp(self):
    super().setUp()
    self.working_dir = tempfile.TemporaryDirectory()
    self.script_path = pathlib.Path(self.working_dir.name).joinpath(
        "enable_feature_based_on_rollout_percentage"
    )
    with resources.as_file(
        resources.files("testdata").joinpath(
            "enable_feature_based_on_rollout_percentage"
        )
    ) as p:
      shutil.copy(p, self.script_path)
    self.script_path.chmod(0o755)

  def tearDown(self):
    self.working_dir.cleanup()
    super().tearDown()

  def test_user_unset(self):
    process = self._run_script(f"""
      unset USER
      {self.script_path} "test_feature" 100
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "error")
    self.assertEqual(process.returncode, 1)

  def test_nonnumeric_rollout_percentage(self):
    process = self._run_script(f"""
      {self.script_path} "test_feature" xyz
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "error")
    self.assertEqual(process.returncode, 1)

  def test_rollout_percentage_out_of_range(self):
    process = self._run_script(f"""
      {self.script_path} "test_feature" 101
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "error")
    self.assertEqual(process.returncode, 1)

    process = self._run_script(f"""
      {self.script_path} "test_feature" -1
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "error")
    self.assertEqual(process.returncode, 1)

  def test_consistently_enabled_feature_for_test_user(self):
    process = self._run_script(f"""
      USER="test_user"
      {self.script_path} "test_feature" 59
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "true")
    self.assertEqual(process.returncode, 0)

    process = self._run_script(f"""
      USER="test_user"
      {self.script_path} "test_feature" 59
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "true")
    self.assertEqual(process.returncode, 0)

  def test_consistently_disable_feature_for_test_user(self):
    process = self._run_script(f"""
      USER="test_user"
      {self.script_path} "test_feature" 58
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "false")
    self.assertEqual(process.returncode, 0)

    process = self._run_script(f"""
      USER="test_user"
      {self.script_path} "test_feature" 58
    """)

    self.assertEqual(process.stdout.decode().rstrip("\n"), "false")
    self.assertEqual(process.returncode, 0)

  def _run_script(self, test_script: str) -> subprocess.CompletedProcess:
    return subprocess.run(
        ["bash", "-c", test_script],
        capture_output=True,
        check=False,
        cwd=self.working_dir.name,
    )


if __name__ == "__main__":
  unittest.main()

#!/usr/bin/env python
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
"""A tool for calculating the hash of a directory based on file contents and metadata."""

import hashlib
import os
import stat
import sys

def calculate_hash(directory: str) -> str:
    """
    Calculates the hash of a directory, including file contents and metadata.

    Following informations are taken into consideration:
    * Name: The file or directory name.
    * File Type: Whether it's a regular file, directory, symbolic link, etc.
    * Size: The size of the file in bytes.
    * Permissions: The file's access permissions (read, write, execute).
    * Content Hash (for files): The SHA-1 hash of the file's content.
    """

    output = []
    for root, _, files in os.walk(directory, topdown=False):
        for file in files:
            filepath = os.path.join(root, file)
            try:
                file_stat = os.stat(filepath)
                stat_info = f"{filepath} {stat.filemode(file_stat.st_mode)} {file_stat.st_size} {file_stat.st_mode & 0o777}"

                if os.path.isfile(filepath):
                    with open(filepath, "rb") as f:
                        file_hash = hashlib.sha1(f.read()).hexdigest()
                    stat_info += f" {file_hash}"

                output.append(stat_info)

            except OSError as e:
                print(f"Error processing {filepath}: {e}", file=sys.stderr)

    return hashlib.sha1("\n".join(sorted(output)).encode()).hexdigest()

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: calculate_hash.py <directory> <output_file>")
        sys.exit(1)

    directory = sys.argv[1]
    output_file = sys.argv[2]

    try:
        hash_value = calculate_hash(directory)
        with open(output_file, "w") as f:
            f.write(hash_value)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

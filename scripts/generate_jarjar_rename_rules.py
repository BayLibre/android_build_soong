#!/usr/bin/env python3
#
# Copyright (C) 2025 The Android Open Source Project
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
"""A tool to generate the jarjar rename rules."""

import argparse

def generate_jarjar_rules(classnames, prefix, output_path, existing_rules_paths):
    """
    Generates jarjar rename rules, merging existing and new rules with sorting.
    Raises an error if conflicting rules are found.

    Args:
      classnames: A list of class names to rename.
      prefix: The prefix to add to the renamed class names.
      output_path: The path to write the output file.
      existing_rules_paths: A list of paths to files containing existing jarjar rules.
    """

    existing_rules = {}  # Use a dictionary to store rules and detect conflicts
    for path in existing_rules_paths:
        with open(path, 'r') as f:
            for line in f:
                rule = line.strip()
                if len(rule.split()) != 3:
                    raise ValueError(f"Unknown jarjar rule format {rule} found in {path}")
                original_name = rule.split()[1]  # Extract original name from rule
                if original_name in existing_rules and existing_rules[original_name] != rule:
                    raise ValueError(f"Conflicting rule found for {original_name}: {existing_rules[original_name]} vs {rule}")
                existing_rules[original_name] = rule

    all_rules = existing_rules.copy()  # Start with existing rules
    for classname in classnames:
        new_rule = f"rule {classname} {prefix}.{classname}"
        if classname in existing_rules and existing_rules[classname] != new_rule:
            raise ValueError(f"Conflicting rule found for {classname}: {existing_rules[classname]} vs {new_rule}")
        all_rules[classname] = new_rule  # Add new rule

    with open(output_path, 'w') as f:
        for rule in sorted(all_rules.values()):  # Sort and write the rules
            f.write(f"{rule}\n")

def main():
    parser = argparse.ArgumentParser(description="Generate jarjar rename rules.")
    parser.add_argument(
        "classnames",
        nargs="*",
        help="Class names to rename.",
    )
    parser.add_argument(
        "-f", "--classnames-files", 
        nargs="*", 
        help="Paths to text files containing lists of class names to rename.",
    )
    parser.add_argument(
        "-p", "--prefix", required=True, help="Prefix to add to renamed class names"
    )
    parser.add_argument(
        "-o", "--output", required=True, help="Path to output file"
    )
    parser.add_argument(
        "-e", "--existing-rules", 
        nargs="*", 
        default=[], 
        help="Paths to files containing existing jarjar rules."
    )
    args = parser.parse_args()

    all_classnames = args.classnames 
    if args.classnames_files:
        for filepath in args.classnames_files:
            with open(filepath, 'r') as f:
                all_classnames.extend(line.strip() for line in f)

    generate_jarjar_rules(all_classnames, args.prefix, args.output, args.existing_rules)

if __name__ == "__main__":
    main()
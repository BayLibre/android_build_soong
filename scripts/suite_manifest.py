#!/usr/bin/env python3

import argparse
import json
import os
import sys
import zipfile
from typing import List

# Input dir
# .
# └── out/
#     ├── suites/
#     │   └── my-team-suite/
#     │       ├── my-team-suite.json
#     │       ├── config/
#     │       │       └── my-team-suite.xml
#     │       ├── host/
#     │       │   └── testcases
#     │       │       └── MyTeamTest1
#     │       │           ├── MyTeamTest1.apk -> _path_to_MyTeamTest1.apk_
#     │       │           └── MyTeamTest1.config -> _path_to_MyTeamTest1.config_
#     │       └── target
#     │           └── testcases
#     │               └── MyTeamTest2
#     │                   ├── MyTeamTest2.apk -> _path_to_MyTeamTest2.apk_
#     │                   └── MyTeamTest2.config -> _path_to_MyTeamTest12.config_

# Output file
#    my-team-suite.json
# {
#   "name": "my-team-suite",
#   "config": "config/my-team-suite.xml",
#   "files": [
#       "config/my-team-suite.xml",
#       "host/testcases/MyTeamTest1/MyTeamTest1.config",
#       "host/testcases/MyTeamTest1/MyTeamTest1.apk",
#       "target/testcases/MyTeamTest2/MyTeamTest2.config",
#       "target/testcases/MyTeamTest2/MyTeamTest2.apk"
#   ]
# }

def main():
    parser = argparse.ArgumentParser(
        description='Generates a list of all files under a suite, ignorning the manifest file and zip file'
    )
    parser.add_argument('--suite_dir', help='Root of test suite to walk')
    parser.add_argument('--output_json', help='Where to write json file')
    parser.add_argument('--output_list', help='Where to write tesxt file of file list')
    args = parser.parse_args()
    all_files = []
    for root, dirs, files in os.walk(args.suite_dir, topdown=True, followlinks=True):
          for name in files:
              # TODO(ron): strip out suite-config and zip if they are there.
              all_files.append(os.path.join(root, name).removeprefix(args.suite_dir + "/"))

    output = {}
    all_files = sorted(all_files)
    output['files'] = all_files
    with open(args.output_json, 'w') as f:
        json.dump(output, f, sort_keys=True, indent=4)

    with open(args.output_list, 'w') as f:
        f.write("\n".join([f"{args.suite_dir}/{x}" for x in all_files]))

if __name__ == "__main__":
    main()

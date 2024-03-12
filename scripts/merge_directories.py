#!/usr/bin/env python3
import argparse
import os
import shutil
import sys

def main():
    parser = argparse.ArgumentParser(
        description="Given a list of directories, this script will copy the contents of all of them into the first directory, erroring out if any duplicate files are found."
    )
    parser.add_argument("--ignore-duplicates", action="store_true",
                        help="Don't error out on duplicate files, just skip them. The file from the earliest directory listed on the command line will be the winner.")
    parser.add_argument("--file-list", help="Path to a text file containing paths relative to in_dir. There must only be 1 in_dir in this case. Only these paths will be copied out of in_dir.")
    parser.add_argument("out_dir")
    parser.add_argument("in_dirs", nargs='+')
    args = parser.parse_args()

    if not os.path.isdir(args.out_dir):
        sys.exit("error: {args.out_dir} must be a directory")

    file_list = None
    if args.file_list:
        if len(args.in_dirs) > 1:
            sys.exit("Must only use 1 input directory when using --file-list")
        with open(file_list_file, "r") as f:
            file_list = f.read().strip().splitlines()

    file_to_origional_dir = {}
    for root, dirs, files in os.walk(args.out_dir):
        for d in dirs:
            p = os.path.normpath(os.path.relpath(os.path.join(root, d), args.out_dir))
            file_to_origional_dir[p] = args.out_dir
        for f in files:
            p = os.path.relpath(os.path.join(root, f), args.out_dir)
            file_to_origional_dir[p] = args.out_dir

    for in_dir in args.in_dirs:
        if not os.path.isdir(in_dir):
            sys.exit("error: {in_dir} must be a directory")
        for root, dirs, files in os.walk(in_dir):
            rel_root = os.path.relpath(root, in_dir)
            dst_root = os.path.join(args.out_dir, rel_root)
            os.makedirs(dst_root, exist_ok=True)
            for f in files:
                src = os.path.join(root, f)
                dst = os.path.join(dst_root, f)
                p = os.path.normpath(os.path.join(rel_root, f))
                if file_list is not None and p not in file_list:
                    continue
                if os.path.lexists(dst):
                    if args.ignore_duplicates:
                        continue
                    sys.exit(f"error: {p} exists in both {file_to_origional_dir.get(p, '???')} and {in_dir}")
                file_to_origional_dir[p] = in_dir

                shutil.copy2(src, dst, follow_symlinks=False)

if __name__ == "__main__":
    main()

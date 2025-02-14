
import os
import sys
import runpy
import tempfile
import zipfile
import shutil
from pathlib import Path


sys.argv[0] = __loader__.archive

# Set sys.executable to None. The real executable is available as
# sys.argv[0], and too many things assume sys.executable is a regular Python
# binary, which isn't available. By setting it to None we get clear errors
# when people try to use it.
sys.executable = None

# Extract the shared libraries from the zip file into a temporary directory.
# This works around the limitations of dynamic linker.  Some Python libraries
# reference the so files relatively and so we can't only extract the so files
# to the sys.path in that case, so we extract the entire zip file to a tempdir
# and then add that to sys.path.
tempdir = None
with zipfile.ZipFile(__loader__.archive) as z:
  for member in z.infolist():
    if member.filename.endswith('.so'):
      # if we have any so extract entire zip file to tempdir
      # and then add tempdir to sys.path
      tempdir = tempfile.mkdtemp()
      z.extractall(tempdir)
      sys.path.insert(0, tempdir)
      break

try:
  runpy._run_module_as_main("ENTRY_POINT", alter_argv=False)
finally:
  if tempdir is not None:
    shutil.rmtree(tempdir)

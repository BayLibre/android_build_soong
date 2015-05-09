# The default check uses size and modification time, causing false misses
# since the mtime depends when the repo was checked out
export CCACHE_COMPILERCHECK=content

# See man page, optimizations to get more cache hits
# implies that __DATE__ and __TIME__ are not critical for functionality.
# Ignore include file modification time since it will depend on when
# the repo was checked out
export CCACHE_SLOPPINESS=time_macros,include_file_mtime,file_macro

# Turn all preprocessor absolute paths into relative paths.
# Fixes absolute paths in preprocessed source due to use of -g.
# We don't really use system headers much so the rootdir is
# fine; ensures these paths are relative for all Android trees
# on a workstation.
export CCACHE_BASEDIR=/

# Workaround for ccache with clang.
# See http://petereisentraut.blogspot.com/2011/09/ccache-and-clang-part-2.html
export CCACHE_CPP2=true

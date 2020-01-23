#!/bin/bash -e

cd "$(dirname "${BASH_SOURCE[0]}")"

[ -z "$CLANG" ] && CLANG="../../../prebuilts/clang/host/linux-x86/clang-r370808/bin/clang++"
CPPFLAGS="-O2 -g -S $TEST_CPPFLAGS"

echo "-fexceptions"
time $CLANG $CPPFLAGS -fexceptions test.cpp -o test.exceptions.S

echo

echo "-fno-exceptions"
time $CLANG $CPPFLAGS -fno-exceptions test.cpp -o test.no-exceptions.S

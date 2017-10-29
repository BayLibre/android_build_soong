// Copyright 2018 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package paths

// Tools that are allowed to be in the $PATH. If true, the tool is supported
// and there will be no logs about uses. If false, the tool is unsupported, and
// will not be present in $PATH.
//
// Currently, if a tool is not in this list, it will still be present in $PATH,
// but a message will be printed in soong.log when it is used. Eventually, I
// expect this to change so that a tool not in this list won't be available.
var AllowedPathEntries = map[string]bool{
	"awk":       true,
	"basename":  true,
	"bash":      true,
	"bzip2":     true,
	"cat":       true,
	"chmod":     true,
	"cmp":       true,
	"comm":      true,
	"cp":        true,
	"cut":       true,
	"date":      true,
	"dd":        true,
	"diff":      true,
	"dirname":   true,
	"echo":      true,
	"egrep":     true,
	"env":       true,
	"expr":      true,
	"find":      true,
	"getconf":   true,
	"getopt":    true,
	"git":       true,
	"grep":      true,
	"gzip":      true,
	"head":      true,
	"hexdump":   true,
	"hostname":  true,
	"jar":       true,
	"java":      true,
	"javap":     true,
	"ln":        true,
	"ls":        true,
	"m4":        true,
	"make":      true,
	"md5sum":    true,
	"mkdir":     true,
	"mktemp":    true,
	"mv":        true,
	"openssl":   true,
	"patch":     true,
	"perl":      true,
	"pstree":    true,
	"python":    true,
	"readlink":  true,
	"realpath":  true,
	"rm":        true,
	"rsync":     true,
	"runalarm":  true,
	"sed":       true,
	"setsid":    true,
	"sh":        true,
	"sha256sum": true,
	"sha512sum": true,
	"sort":      true,
	"stat":      true,
	"sum":       true,
	"tar":       true,
	"tail":      true,
	"touch":     true,
	"tr":        true,
	"true":      true,
	"uname":     true,
	"uniq":      true,
	"unzip":     true,
	"wc":        true,
	"which":     true,
	"whoami":    true,
	"xargs":     true,
	"xmllint":   true,
	"xz":        true,
	"zip":       true,
	"zipinfo":   true,

	// Host toolchain is removed. In-tree toolchain should be used instead.
	// GCC also can't find cc1 with this implementation.
	"ar":         false,
	"as":         false,
	"cc":         false,
	"clang":      false,
	"clang++":    false,
	"gcc":        false,
	"g++":        false,
	"ld":         false,
	"ld.bfd":     false,
	"ld.gold":    false,
	"pkg-config": false,

	// We've got prebuilts of these
	//"dtc":  false,
	//"lz4":  false,
	//"lz4c": false,
}

module android/soong

require (
  github.com/google/blueprint v0.0.0
  go.starlark.net v0.0.0-20211203141949-70c0e40ae128
  google.golang.org/protobuf v1.25.0
)

replace (
  github.com/google/blueprint v0.0.0 => ../blueprint
  github.com/google/go-cmp v0.5.5 => ../../external/go-cmp
  go.starlark.net => ../../external/starlark-go
  google.golang.org/protobuf => ../../external/golang-protobuf
)

exclude (
  // Indirect deps from golang-protobuf
  github.com/golang/protobuf v1.5.0
  // Indirect dep from go-cmp
  golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
)

go 1.15

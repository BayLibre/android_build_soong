module android/soong

replace (
	github.com/google/blueprint => ../blueprint
	github.com/google/go-cmp => ../../external/go-cmp
	github.com/golang/protobuf => ../../external/golang-protobuf
	google.golang.org/protobuf => ../../external/golang-protobuf
	prebuilts/bazel/common/proto/analysis_v2 => ../../prebuilts/bazel/common/proto/analysis_v2
	prebuilts/bazel/common/proto/build => ../../prebuilts/bazel/common/proto/build
)

go 1.19

require (
	github.com/google/blueprint v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v0.0.0-00010101000000-000000000000
	prebuilts/bazel/common/proto/analysis_v2 v0.0.0-00010101000000-000000000000
)

require prebuilts/bazel/common/proto/build v0.0.0-00010101000000-000000000000 // indirect

package java

import (
	"android/soong/common"
)

func (*JavaLibrary) AndroidMk() (ret common.AndroidMkData) {
	ret.Class = "JAVA_LIBRARIES"
	// TODO
	return
}

func (*JavaPrebuilt) AndroidMk() (ret common.AndroidMkData) {
	ret.Class = "JAVA_LIBRARIES"
	// TODO
	return
}

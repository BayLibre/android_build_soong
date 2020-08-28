# "cc_object" module copts, taken from build/soong/cc/object.go
_CC_OBJECT_COPTS = ["-fno-addrsig"]

def cc_object(copts = [], **kwargs):
    "Build macro to correspond with the cc_object Soong module."
    native.cc_library(
        copts = _CC_OBJECT_COPTS + copts,
        **kwargs
    )

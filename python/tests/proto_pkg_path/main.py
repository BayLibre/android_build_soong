import sys

# The python interpreter prepends the directory of the entrypoint script
# to sys.path. We should eventually remove this using -P or the
# PYTHON_SAFE_PATH enviornment variable in python 3.11, but until that
# happens, delete it manually to get more accurate results. Without the
# deletion we get the following error, though I haven't yet fully debugged why:
# TypeError: Parameter to MergeFrom() must be instance of same class: expected MyCommonMessage got MyCommonMessage. for field MyMessage.common
del sys.path[0]

import unittest
import subpackage.proto.test_pb2 as test_pb2
import subpackage.proto.common_pb2 as common_pb2

print(sys.path)

class TestProtoWithPkgPath(unittest.TestCase):

    def test_main(self):
        x = test_pb2.MyMessage(name="foo",
                               common = common_pb2.MyCommonMessage(common="common"))
        self.assertEqual(x.name, "foo")
        self.assertEqual(x.common.common, "common")

if __name__ == '__main__':
    unittest.main()

#!/usr/bin/env python
#
# Copyright (C) 2016 The Android Open Source Project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
"""Tests for gen_stub_libs.py."""
import unittest

import gen_stub_libs as gsl


# pylint: disable=missing-docstring

class StackTest(unittest.TestCase):
    def test_empty_init(self):
        stack = gsl.Stack()
        self.assertEqual(0, len(stack.stack))

    def test_stack(self):
        stack = gsl.Stack()

        stack.push('foo')
        self.assertEqual('foo', stack.top)
        self.assertEqual('foo', stack.pop())
        with self.assertRaises(IndexError):
            _ = stack.top
        with self.assertRaises(IndexError):
            stack.pop()

        stack.push('foo')
        stack.push('bar')
        self.assertEqual('bar', stack.top)
        self.assertEqual('bar', stack.pop())
        self.assertEqual('foo', stack.top)
        self.assertEqual('foo', stack.pop())
        with self.assertRaises(IndexError):
            _ = stack.top
        with self.assertRaises(IndexError):
            stack.pop()


class TagsTest(unittest.TestCase):
    def test_get_tags_no_tags(self):
        self.assertEqual([], gsl.get_tags(''))
        self.assertEqual([], gsl.get_tags('foo bar baz'))

    def test_get_tags(self):
        self.assertEqual(['foo', 'bar'], gsl.get_tags('# foo bar'))
        self.assertEqual(['bar', 'baz'], gsl.get_tags('foo # bar baz'))

    def test_get_tag_value(self):
        self.assertEqual('bar', gsl.get_tag_value('foo=bar'))
        self.assertEqual('bar=baz', gsl.get_tag_value('foo=bar=baz'))
        with self.assertRaises(ValueError):
            gsl.get_tag_value('foo')


class PrivateVersionTest(unittest.TestCase):
    def test_version_is_private(self):
        self.assertFalse(gsl.version_is_private('foo'))
        self.assertFalse(gsl.version_is_private('PRIVATE'))
        self.assertFalse(gsl.version_is_private('PLATFORM'))
        self.assertFalse(gsl.version_is_private('foo_private'))
        self.assertFalse(gsl.version_is_private('foo_platform'))
        self.assertFalse(gsl.version_is_private('foo_PRIVATE_'))
        self.assertFalse(gsl.version_is_private('foo_PLATFORM_'))

        self.assertTrue(gsl.version_is_private('foo_PRIVATE'))
        self.assertTrue(gsl.version_is_private('foo_PLATFORM'))


class SymbolPresenceTest(unittest.TestCase):
    def test_symbol_in_arch(self):
        self.assertTrue(gsl.symbol_in_arch([], 'arm'))
        self.assertTrue(gsl.symbol_in_arch(['arm'], 'arm'))

        self.assertFalse(gsl.symbol_in_arch(['x86'], 'arm'))

    def test_symbol_in_api(self):
        self.assertTrue(gsl.symbol_in_api([], 'arm', 9))
        self.assertTrue(gsl.symbol_in_api(['introduced=9'], 'arm', 9))
        self.assertTrue(gsl.symbol_in_api(['introduced=9'], 'arm', 14))
        self.assertTrue(gsl.symbol_in_api(['introduced-arm=9'], 'arm', 14))
        self.assertTrue(gsl.symbol_in_api(['introduced-arm=9'], 'arm', 14))
        self.assertTrue(gsl.symbol_in_api(['introduced-x86=14'], 'arm', 9))
        self.assertTrue(gsl.symbol_in_api(
            ['introduced-arm=9', 'introduced-x86=21'], 'arm', 14))
        self.assertTrue(gsl.symbol_in_api(
            ['introduced=9', 'introduced-x86=21'], 'arm', 14))
        self.assertTrue(gsl.symbol_in_api(
            ['introduced=21', 'introduced-arm=9'], 'arm', 14))

        self.assertFalse(gsl.symbol_in_api(['introduced=14'], 'arm', 9))
        self.assertFalse(gsl.symbol_in_api(['introduced-arm=14'], 'arm', 9))
        self.assertFalse(gsl.symbol_in_api(['future'], 'arm', 9))
        self.assertFalse(gsl.symbol_in_api(
            ['introduced=9', 'future'], 'arm', 14))
        self.assertFalse(gsl.symbol_in_api(
            ['introduced-arm=9', 'future'], 'arm', 14))
        self.assertFalse(gsl.symbol_in_api(
            ['introduced-arm=21', 'introduced-x86=9'], 'arm', 14))
        self.assertFalse(gsl.symbol_in_api(
            ['introduced=9', 'introduced-arm=21'], 'arm', 14))
        self.assertFalse(gsl.symbol_in_api(
            ['introduced=21', 'introduced-x86=9'], 'arm', 14))

        # Interesting edge case: this symbol should be omitted from the
        # library, but this call should still return true because none of the
        # tags indiciate that it's not present in this API level.
        self.assertTrue(gsl.symbol_in_api(['x86'], 'arm', 9))

    def test_verioned_in_api(self):
        self.assertTrue(gsl.symbol_versioned_in_api([], 9))
        self.assertTrue(gsl.symbol_versioned_in_api(['versioned=9'], 9))
        self.assertTrue(gsl.symbol_versioned_in_api(['versioned=9'], 14))

        self.assertFalse(gsl.symbol_versioned_in_api(['versioned=14'], 9))


class OmitVersionTest(unittest.TestCase):
    def test_omit_private(self):
        self.assertFalse(gsl.should_omit_version('foo', [], 'arm', 9))

        self.assertTrue(gsl.should_omit_version('foo_PRIVATE', [], 'arm', 9))
        self.assertTrue(gsl.should_omit_version('foo_PLATFORM', [], 'arm', 9))

    def test_omit_arch(self):
        self.assertFalse(gsl.should_omit_version('foo', [], 'arm', 9))
        self.assertFalse(gsl.should_omit_version('foo', ['arm'], 'arm', 9))

        self.assertTrue(gsl.should_omit_version('foo', ['x86'], 'arm', 9))

    def test_omit_api(self):
        self.assertFalse(gsl.should_omit_version('foo', [], 'arm', 9))
        self.assertFalse(
            gsl.should_omit_version('foo', ['introduced=9'], 'arm', 9))

        self.assertTrue(
            gsl.should_omit_version('foo', ['introduced=14'], 'arm', 9))


def main():
    suite = unittest.TestLoader().loadTestsFromName(__name__)
    unittest.TextTestRunner(verbosity=3).run(suite)


if __name__ == '__main__':
    main()

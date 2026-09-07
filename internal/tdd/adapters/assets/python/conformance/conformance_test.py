import unittest

import machinery_check


class ConformanceAsync(unittest.IsolatedAsyncioTestCase):
    async def test_async_witness(self):
        machinery_check.check(self, "conformance/async-witness", 2 * 7 == 14)


class ConformanceIdentity(unittest.TestCase):
    def test_class_identity(self):
        machinery_check.check(self, "conformance/class-identity", __name__ == "conformance_test")


class ConformanceMultiple(unittest.TestCase):
    def test_multiple(self):
        machinery_check.check(self, "conformance/multi-a", len("machinery") == 9)
        machinery_check.check(self, "conformance/multi-b", len("assurance") == 9)


class ConformanceWitness(unittest.TestCase):
    def test_witness_pass(self):
        machinery_check.check(self, "conformance/witness-pass", 6 * 7 == 42)

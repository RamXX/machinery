"""Machinery's byte-pinned assertion helper of the closed machinery-check/v1
transport for the python-unittest/v1 adapter (MAC-imtz). These bytes are
frozen: internal/tdd/adapters/python.go embeds them, pins their sha256,
materializes them into every prepared suite and verifies them before and
after native execution. Do not edit without a new reviewed revision; the
adapter rejects any mismatched copy.

check(case, assertion_id, condition) evaluates exactly one registered
assertion at its exact frozen call site. It emits exactly one machine
witness record

    {"schema":"machinery.tdd.witness/v1","kind":"witness",
     "assertion":<ID>,"test":<TestCase.id()>,"site":"<FILE>:<LINE>",
     "condition":<bool>,"thrown":null|"AssertionError"}

through the embedded harness channel, and on a false condition lets the
native framework fail the test by raising AssertionError itself. The
condition argument is strictly boolean (a non-boolean is a TypeError, never
an assertion outcome). It never catches, converts or disguises any product
exception raised by surrounding test code.
"""

import sys

_WITNESS_SCHEMA = "machinery.tdd.witness/v1"


def _harness_channel():
    harness = sys.modules.get("_machinery_harness")
    if harness is None or not hasattr(harness, "emit_witness"):
        raise RuntimeError(
            "machinery-check/v1 requires the embedded harness channel"
        )
    return harness


def check(case, assertion_id, condition):
    if not isinstance(assertion_id, str) or not assertion_id:
        raise TypeError("machinery-check/v1 assertion ids are nonempty strings")
    if type(condition) is not bool:
        raise TypeError("machinery-check/v1 condition must be strictly boolean")
    frame = sys._getframe(1)
    filename = frame.f_code.co_filename
    head = filename.rsplit("/", 1)[-1].rsplit("\\", 1)[-1]
    site = head + ":" + str(frame.f_lineno)
    test = case.id()
    channel = _harness_channel()
    if not condition:
        channel.emit_witness(
            {
                "schema": _WITNESS_SCHEMA,
                "kind": "witness",
                "assertion": assertion_id,
                "test": test,
                "site": site,
                "condition": False,
                "thrown": "AssertionError",
            }
        )
        raise AssertionError(
            "machinery-check/v1 assertion "
            + assertion_id
            + " evaluated false at "
            + site
        )
    channel.emit_witness(
        {
            "schema": _WITNESS_SCHEMA,
            "kind": "witness",
            "assertion": assertion_id,
            "test": test,
            "site": site,
            "condition": True,
            "thrown": None,
        }
    )

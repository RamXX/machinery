"""Machinery's byte-pinned bootstrap/harness of the python-unittest/v1
adapter (MAC-imtz). These bytes are frozen: internal/tdd/adapters/python.go
embeds them, pins their sha256, materializes them into every prepared suite
and verifies them before and after native execution.

The harness runs the frozen suite through the REAL stdlib unittest lifecycle
under CPython's isolated mode: it fixes sys.path to the suite root plus the
interpreter's own stdlib entries, disables bytecode writing, snapshots the
stdlib identity anchors around every module import and execution, loads the
declared modules with the native TestLoader, reconciles the complete
TestCase.id() inventory before running the closed suite, records every
startTest/stopTest/addSuccess/addFailure/addError/addSkip/expectedFailure/
unexpectedSuccess callback plus every machinery-check/v1 witness into one
closed JSONL report stream, and detects unawaited-coroutine residue
deterministically (RuntimeWarning-as-error plus an unraisable hook and a
final forced collection).

Closed subcommands (exact argv, absolute paths supplied by the adapter):

    bootstrap.py validate <inventory.json>
    bootstrap.py discover <inventory.json>
    bootstrap.py run <inventory.json> <report.jsonl>

Exit codes: 0 = requested work complete and green, 1 = only witness-backed
assertion failures, 2 = any error class (import/syntax/integrity/reconcile/
unsupported feature/unexpected failure/residue).
"""

import ast
import gc
import importlib
import io
import json
import os
import sys
import types
import unittest
import warnings

HARNESS_SCHEMA = "machinery.tdd.harness.python/v1"
INVENTORY_SCHEMA = "machinery.tdd.suite.python/v1"
TERMINAL_KIND = "bootstrap-end"
_MAX_INVENTORY_BYTES = 1 << 20
_MAX_MESSAGE = 500
_TESTCASE_BASES = ("TestCase", "IsolatedAsyncioTestCase")
_UNSUPPORTED_BASES = (
    "TestResult",
    "TextTestResult",
    "TestRunner",
    "TextTestRunner",
    "TestLoader",
    "TestSuite",
)
_FORBIDDEN_IMPORT_ROOTS = ("pytest", "doctest")
_EXIT_NAMES = ("exit", "_exit")


class InventoryError(Exception):
    pass


def _no_duplicate_keys(pairs):
    out = {}
    for key, value in pairs:
        if key in out:
            raise InventoryError("duplicate key " + key)
        out[key] = value
    return out


def load_inventory(path):
    try:
        with open(path, "r", encoding="utf-8") as handle:
            body = handle.read(_MAX_INVENTORY_BYTES + 1)
    except OSError as error:
        raise InventoryError("cannot read the inventory: " + str(error)) from None
    if len(body) > _MAX_INVENTORY_BYTES:
        raise InventoryError("inventory exceeds the bounded size")
    try:
        doc = json.loads(body, object_pairs_hook=_no_duplicate_keys)
    except (ValueError, InventoryError) as error:
        raise InventoryError("inventory is not closed JSON: " + str(error)) from None
    if not isinstance(doc, dict) or set(doc) != {"schema", "root", "tests"}:
        raise InventoryError("inventory is not the closed record")
    if doc["schema"] != INVENTORY_SCHEMA or not isinstance(doc["root"], str):
        raise InventoryError("inventory identity is not the closed schema")
    if not isinstance(doc["tests"], list) or not doc["tests"]:
        raise InventoryError("inventory declares no tests")
    seen_tests = set()
    seen_assertions = set()
    for entry in doc["tests"]:
        if not isinstance(entry, dict) or set(entry) != {
            "module",
            "class",
            "method",
            "file",
            "assertions",
        }:
            raise InventoryError("inventory test entry is not the closed record")
        for field in ("module", "class", "method", "file"):
            if not isinstance(entry[field], str) or not entry[field]:
                raise InventoryError("inventory identity field is not a string")
        identity = entry["module"] + "." + entry["class"] + "." + entry["method"]
        if identity in seen_tests:
            raise InventoryError("duplicate declared identity " + identity)
        seen_tests.add(identity)
        if not isinstance(entry["assertions"], list):
            raise InventoryError("inventory assertions are not a list")
        for assertion in entry["assertions"]:
            if not isinstance(assertion, dict) or set(assertion) != {"id", "line"}:
                raise InventoryError("inventory assertion entry is not the closed record")
            if not isinstance(assertion["id"], str) or not assertion["id"]:
                raise InventoryError("inventory assertion id is not a string")
            if not isinstance(assertion["line"], int) or isinstance(assertion["line"], bool):
                raise InventoryError("inventory assertion line is not an integer")
            if assertion["line"] < 1:
                raise InventoryError("inventory assertion line is not positive")
            if assertion["id"] in seen_assertions:
                raise InventoryError("duplicate assertion id " + assertion["id"])
            seen_assertions.add(assertion["id"])
    return doc


class _Validation:
    def __init__(self, code, message, file="", line=0):
        self.code = code
        self.message = message
        self.file = file
        self.line = line


def _dotted_name(node):
    parts = []
    while isinstance(node, ast.Attribute):
        parts.append(node.attr)
        node = node.value
    if isinstance(node, ast.Name):
        parts.append(node.id)
        return ".".join(reversed(parts))
    return ""


def _class_derives_testcase(cls, classes):
    resolved = set()
    pending = [cls]
    while pending:
        current = pending.pop()
        if current["name"] in resolved:
            continue
        resolved.add(current["name"])
        for base in current["bases"]:
            name = _dotted_name(base)
            tail = name.rsplit(".", 1)[-1] if name else ""
            if tail in _TESTCASE_BASES:
                return True
            if name in classes:
                pending.append(classes[name])
    return False


def _walk_nodes(root):
    stack = [root]
    while stack:
        node = stack.pop()
        yield node
        for child in ast.iter_child_nodes(node):
            stack.append(child)


def validate_suite(inventory):
    root = inventory["root"]
    classes = {}
    trees = {}
    declared = {}
    for entry in inventory["tests"]:
        declared[(entry["module"], entry["class"], entry["method"])] = entry
        path = os.path.join(root, entry["file"])
        if entry["file"] in trees:
            continue
        try:
            with open(path, "r", encoding="utf-8") as handle:
                source = handle.read()
        except OSError as error:
            return _Validation("MISSING_TEST", "suite source cannot be read: " + str(error), entry["file"], 0)
        try:
            tree = ast.parse(source, filename=entry["file"])
        except SyntaxError as error:
            return _Validation("BUILD_ERROR", "suite source does not parse: " + str(error), entry["file"], error.lineno or 0)
        trees[entry["file"]] = tree
    for name, tree in trees.items():
        for node in tree.body:
            if isinstance(node, ast.ClassDef):
                classes[node.name] = {"name": node.name, "bases": node.bases, "def": node, "file": name}
    for name, tree in trees.items():
        for node in tree.body:
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and node.name == "load_tests":
                return _Validation("UNSUPPORTED_FEATURE", "load_tests is outside the strict suite", name, node.lineno)
            if isinstance(node, (ast.Import, ast.ImportFrom)):
                roots = []
                if isinstance(node, ast.Import):
                    roots = [alias.name.split(".")[0] for alias in node.names]
                else:
                    roots = [(node.module or "").split(".")[0]]
                if any(item in _FORBIDDEN_IMPORT_ROOTS for item in roots):
                    return _Validation("UNSUPPORTED_FEATURE", "unsupported framework import", name, node.lineno)
            if isinstance(node, ast.Assign):
                for target in node.targets:
                    dotted = _dotted_name(target)
                    if dotted.rsplit(".", 1)[0:1] == ["unittest"] or dotted.startswith("unittest."):
                        return _Validation("UNSUPPORTED_FEATURE", "patching the stdlib unittest surface", name, node.lineno)
    for cls_name, cls in classes.items():
        for base in cls["bases"]:
            tail = _dotted_name(base).rsplit(".", 1)[-1]
            if tail in _UNSUPPORTED_BASES:
                return _Validation("UNSUPPORTED_FEATURE", "custom result/runner/loader class " + cls_name, cls["file"], cls["def"].lineno)
        for node in cls["def"].body:
            if isinstance(node, ast.Assign):
                for target in node.targets:
                    if isinstance(target, ast.Name) and target.id == "failureException":
                        return _Validation("UNSUPPORTED_FEATURE", "replacing failureException", cls_name, node.lineno)
        if cls["def"].decorator_list:
            return _Validation("UNSUPPORTED_FEATURE", "test class decorators (skip/xfail) are outside the strict suite", cls_name, cls["def"].lineno)
    for (module_name, cls_name, method_name), entry in declared.items():
        cls = classes.get(cls_name)
        if cls is None:
            return _Validation("MISSING_TEST", "declared class " + cls_name + " is not defined", entry["file"], 0)
        if not _class_derives_testcase({"name": cls_name, "bases": cls["bases"]}, {k: {"name": k, "bases": v["bases"]} for k, v in classes.items()}):
            return _Validation("UNSUPPORTED_FEATURE", "class " + cls_name + " is not a stdlib TestCase/IsolatedAsyncioTestCase", entry["file"], cls["def"].lineno)
        method = None
        for node in cls["def"].body:
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and node.name == method_name:
                method = node
                break
        if method is None:
            return _Validation("MISSING_TEST", "declared method " + cls_name + "." + method_name + " is not defined", entry["file"], cls["def"].lineno)
        if method.decorator_list:
            return _Validation("UNSUPPORTED_FEATURE", "test method decorators (skip/xfail) are outside the strict suite", entry["file"], method.lineno)
        method_nodes = list(_walk_nodes(method))
        for node in method_nodes:
            if isinstance(node, ast.Call):
                func = node.func
                if isinstance(func, ast.Attribute) and func.attr == "subTest":
                    return _Validation("UNSUPPORTED_FEATURE", "subTest identities are not strict test identities", entry["file"], node.lineno)
                if isinstance(func, ast.Attribute) and func.attr in _EXIT_NAMES:
                    dotted = _dotted_name(func.value)
                    if dotted in ("sys", "os"):
                        return _Validation("UNSUPPORTED_FEATURE", "early interpreter exit inside a test", entry["file"], node.lineno)
        for assertion in entry["assertions"]:
            match = None
            for node in method_nodes:
                if isinstance(node, ast.Call) and node.lineno == assertion["line"]:
                    match = node
                    break
            ok = False
            if match is not None and len(match.args) == 3 and not match.keywords:
                func = match.func
                name = _dotted_name(func)
                if name in ("machinery_check.check", "machinery_check.machinery_check.check") or (isinstance(func, ast.Name) and func.id == "check"):
                    if isinstance(match.args[1], ast.Constant) and isinstance(match.args[1].value, str):
                        ok = match.args[1].value == assertion["id"]
            if not ok:
                return _Validation(
                    "ASSERTION_MISMATCH",
                    "registered assertion " + assertion["id"] + " has no typed helper call at line " + str(assertion["line"]),
                    entry["file"],
                    assertion["line"],
                )
    registered_sites = set()
    for (module_name, cls_name, method_name), entry in declared.items():
        cls = classes[cls_name]
        for node in cls["def"].body:
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and node.name == method_name:
                for inner in _walk_nodes(node):
                    if isinstance(inner, ast.Call) and len(inner.args) == 3 and not inner.keywords:
                        func = inner.func
                        name = _dotted_name(func)
                        if name.endswith("machinery_check.check") or (isinstance(func, ast.Name) and func.id == "check"):
                            if isinstance(inner.args[1], ast.Constant) and isinstance(inner.args[1].value, str):
                                registered_sites.add((entry["file"], inner.lineno, inner.args[1].value))
    for name, tree in trees.items():
        for node in _walk_nodes(tree):
            if isinstance(node, ast.Call) and len(node.args) == 3 and not node.keywords:
                func = node.func
                dotted = _dotted_name(func)
                if dotted.endswith("machinery_check.check") or (isinstance(func, ast.Name) and func.id == "check"):
                    if isinstance(node.args[1], ast.Constant) and isinstance(node.args[1].value, str):
                        site = (name, node.lineno, node.args[1].value)
                        if site not in registered_sites:
                            return _Validation(
                                "UNSUPPORTED_FEATURE",
                                "helper call outside the registered inventory (setUp/tearDown or unregistered): " + node.args[1].value,
                                name,
                                node.lineno,
                            )
    return None


class _Channel(types.ModuleType):
    def __init__(self, stream):
        super().__init__("_machinery_harness")
        self._stream = stream

    def emit_witness(self, record):
        self._stream.write(json.dumps(record, sort_keys=True) + "\n")
        self._stream.flush()


class HarnessResult(unittest.TestResult):
    def __init__(self, emit):
        super().__init__()
        self._emit = emit
        self.terminated = {}

    def _record_terminal(self, test, outcome, exception=None):
        identity = test.id()
        if identity in self.terminated:
            self._emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "DUPLICATE_TERMINAL", "message": identity})
            return
        self.terminated[identity] = outcome
        record = {"schema": HARNESS_SCHEMA, "kind": "test-end", "test": identity, "outcome": outcome}
        if exception is not None:
            record["exception"] = exception
        self._emit(record)

    def startTest(self, test):
        super().startTest(test)
        self._emit({"schema": HARNESS_SCHEMA, "kind": "test-start", "test": test.id()})

    def addSuccess(self, test):
        super().addSuccess(test)
        self._record_terminal(test, "pass")

    def addFailure(self, test, err):
        super().addFailure(test, err)
        self._record_terminal(test, "assertion-fail", err[0].__name__)

    def addError(self, test, err):
        super().addError(test, err)
        self._record_terminal(test, "error", err[0].__name__)

    def addSkip(self, test, reason):
        super().addSkip(test, reason)
        self._record_terminal(test, "skipped")

    def addExpectedFailure(self, test, err):
        super().addExpectedFailure(test, err)
        self._record_terminal(test, "expected-failure", err[0].__name__)

    def addUnexpectedSuccess(self, test):
        super().addUnexpectedSuccess(test)
        self._record_terminal(test, "unexpected-success")

    def addSubTest(self, test, subtest, err):
        super().addSubTest(test, subtest, err)
        self._emit({"schema": HARNESS_SCHEMA, "kind": "unsupported-feature", "feature": "subTest", "test": test.id()})


class _Integrity:
    def __init__(self):
        self.anchors = {
            "case.run": unittest.TestCase.run,
            "case.call": unittest.TestCase.__call__,
            "async_case.run": unittest.IsolatedAsyncioTestCase.run,
        }

    def verify(self):
        return (
            self.anchors["case.run"] == unittest.TestCase.run
            and self.anchors["case.call"] == unittest.TestCase.__call__
            and self.anchors["async_case.run"] == unittest.IsolatedAsyncioTestCase.run
        )


def _prepare_interpreter(root):
    sys.dont_write_bytecode = True
    entries = [item for item in sys.path if item]
    sys.path[:] = [root] + entries


def _load_modules(module_names):
    loader = unittest.TestLoader()
    modules = []
    for name in module_names:
        module = importlib.import_module(name)
        modules.append(module)
    return loader, modules


def _suite_tests(suite):
    ordered = []
    stack = [suite]
    while stack:
        current = stack.pop(0)
        if isinstance(current, unittest.TestSuite):
            stack = list(current) + stack
        else:
            ordered.append(current)
    return ordered


def _check_case_integrity(case, integrity):
    case_type = type(case)
    if case_type.failureException is not AssertionError:
        return "custom failureException on " + case_type.__name__
    if case_type.run not in (integrity.anchors["case.run"], integrity.anchors["async_case.run"]):
        return "replaced run lifecycle on " + case_type.__name__
    if case_type.__call__ is not integrity.anchors["case.call"]:
        return "replaced __call__ lifecycle on " + case_type.__name__
    return ""


def _emit_verdict(stream, payload):
    stream.write(json.dumps(payload, sort_keys=True) + "\n")
    stream.flush()


def command_validate(inventory):
    failure = validate_suite(inventory)
    if failure is not None:
        _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "validate", "status": "error", "code": failure.code, "message": failure.message[:_MAX_MESSAGE], "file": failure.file, "line": failure.line})
        return 2
    _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "validate", "status": "ok"})
    return 0


def command_discover(inventory):
    _prepare_interpreter(inventory["root"])
    integrity = _Integrity()
    module_names = sorted({entry["module"] for entry in inventory["tests"]})
    try:
        loader, modules = _load_modules(module_names)
    except BaseException as error:  # noqa: BLE001 - closed fail-fast harness boundary
        _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "discover", "status": "error", "code": "BUILD_ERROR", "message": ("import failed: " + type(error).__name__ + ": " + str(error))[:_MAX_MESSAGE]})
        return 2
    if not integrity.verify():
        _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "discover", "status": "error", "code": "UNSUPPORTED_FEATURE", "message": "stdlib unittest surface was patched during import"})
        return 2
    tests = []
    for module in modules:
        suite = loader.loadTestsFromModule(module)
        for case in _suite_tests(suite):
            violation = _check_case_integrity(case, integrity)
            if violation:
                _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "discover", "status": "error", "code": "UNSUPPORTED_FEATURE", "message": violation + " (" + case.id() + ")"})
                return 2
            tests.append(case.id())
    _emit_verdict(sys.stdout, {"schema": HARNESS_SCHEMA, "kind": "discover", "status": "ok", "tests": tests, "modules": module_names})
    return 0


def command_run(inventory, report_path):
    _prepare_interpreter(inventory["root"])
    warnings.simplefilter("error", RuntimeWarning)
    residue = []
    previous_hook = sys.unraisablehook

    def recording_hook(unraisable):
        residue.append(type(unraisable.exc_value).__name__ + ": " + str(unraisable.exc_value)[:_MAX_MESSAGE])
        previous_hook(unraisable)

    sys.unraisablehook = recording_hook
    integrity = _Integrity()
    declared = {entry["module"] + "." + entry["class"] + "." + entry["method"]: entry for entry in inventory["tests"]}
    module_names = sorted({entry["module"] for entry in inventory["tests"]})
    with open(report_path, "w", encoding="utf-8") as handle:
        def emit(record):
            handle.write(json.dumps(record, sort_keys=True) + "\n")
            handle.flush()

        sys.modules["_machinery_harness"] = _Channel(handle)
        real_stdout = sys.stdout
        sys.stdout = io.StringIO()
        exit_code = 2
        try:
            try:
                loader, modules = _load_modules(module_names)
            except BaseException as error:  # noqa: BLE001 - closed fail-fast harness boundary
                emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "BUILD_ERROR", "message": ("import failed: " + type(error).__name__ + ": " + str(error))[:_MAX_MESSAGE]})
                return 2
            if not integrity.verify():
                emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "UNSUPPORTED_FEATURE", "message": "stdlib unittest surface was patched during import"})
                return 2
            suite = unittest.TestSuite()
            order = []
            for module in modules:
                suite.addTests(loader.loadTestsFromModule(module))
            for case in _suite_tests(suite):
                identity = case.id()
                if identity not in declared:
                    emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "DUPLICATE_TEST", "message": "undeclared native identity executed: " + identity})
                    return 2
                violation = _check_case_integrity(case, integrity)
                if violation:
                    emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "UNSUPPORTED_FEATURE", "message": violation + " (" + identity + ")"})
                    return 2
                order.append(identity)
            for identity in declared:
                if identity not in order:
                    emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "MISSING_TEST", "message": "declared identity never discovered: " + identity})
                    return 2
            emit({"schema": HARNESS_SCHEMA, "kind": "suite-start", "tests": order})
            result = HarnessResult(emit)
            suite.run(result)
            gc.collect()
            if residue:
                emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "UNEXPECTED_FAILURE", "message": ("async residue: " + residue[0])[:_MAX_MESSAGE]})
            elif not integrity.verify():
                emit({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "UNSUPPORTED_FEATURE", "message": "stdlib unittest surface was patched during execution"})
            else:
                counts = {
                    "tests_run": result.testsRun,
                    "failures": len(result.failures),
                    "errors": len(result.errors),
                    "skipped": len(result.skipped),
                    "expected_failures": len(result.expectedFailures),
                    "unexpected_successes": len(result.unexpectedSuccesses),
                    "success": result.wasSuccessful(),
                }
                emit(dict({"schema": HARNESS_SCHEMA, "kind": "suite-end"}, **counts))
                clean = (
                    counts["tests_run"] == len(order)
                    and counts["skipped"] == 0
                    and counts["expected_failures"] == 0
                    and counts["unexpected_successes"] == 0
                    and counts["errors"] == 0
                )
                if clean and counts["failures"] == 0:
                    exit_code = 0
                elif clean:
                    exit_code = 1
        finally:
            sys.stdout = real_stdout
            sys.unraisablehook = previous_hook
            emit({"schema": HARNESS_SCHEMA, "kind": TERMINAL_KIND})
            handle.flush()
    return exit_code


def main(argv):
    if len(argv) == 3 and argv[1] == "validate":
        try:
            inventory = load_inventory(argv[2])
        except InventoryError as error:
            print(json.dumps({"schema": HARNESS_SCHEMA, "kind": "validate", "status": "error", "code": "INVALID_SCHEMA", "message": str(error)[:_MAX_MESSAGE]}, sort_keys=True))
            return 2
        return command_validate(inventory)
    if len(argv) == 3 and argv[1] == "discover":
        try:
            inventory = load_inventory(argv[2])
        except InventoryError as error:
            print(json.dumps({"schema": HARNESS_SCHEMA, "kind": "discover", "status": "error", "code": "INVALID_SCHEMA", "message": str(error)[:_MAX_MESSAGE]}, sort_keys=True))
            return 2
        return command_discover(inventory)
    if len(argv) == 4 and argv[1] == "run":
        try:
            inventory = load_inventory(argv[2])
        except InventoryError as error:
            with open(argv[3], "w", encoding="utf-8") as handle:
                handle.write(json.dumps({"schema": HARNESS_SCHEMA, "kind": "suite-error", "code": "INVALID_SCHEMA", "message": str(error)[:_MAX_MESSAGE]}, sort_keys=True) + "\n")
                handle.write(json.dumps({"schema": HARNESS_SCHEMA, "kind": TERMINAL_KIND}, sort_keys=True) + "\n")
            return 2
        return command_run(inventory, argv[3])
    print(json.dumps({"schema": HARNESS_SCHEMA, "kind": "usage", "status": "error", "code": "INVALID_SCHEMA", "message": "closed argv is validate|discover <inventory> | run <inventory> <report>"}, sort_keys=True))
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))

import json
import os
import sys

import pytest

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOOLS = os.path.join(REPO, "skills", "machinery", "tools")
GO_CRM = os.path.join(REPO, "examples", "go-crm")
FULFILLMENT = os.path.join(REPO, "examples", "fulfillment")

sys.path.insert(0, TOOLS)


@pytest.fixture
def repo():
    return REPO


@pytest.fixture
def go_crm_design():
    return os.path.join(GO_CRM, "design")


@pytest.fixture
def fulfillment_design():
    return os.path.join(FULFILLMENT, "design")


def minimal_machine(**overrides):
    """A small valid machine: two domain states, one persist overlay."""
    m = {
        "id": "widget",
        "initial": "Draft",
        "context": {"widgetId": None},
        "states": {
            "Draft": {
                "on": {
                    "publish": [
                        {"target": "persisting", "guard": "guardCanPublish",
                         "actions": "setPending"},
                        {"actions": "recordDenied"},
                    ]
                }
            },
            "Published": {"type": "final"},
            "persisting": {
                "invoke": {
                    "src": "saveWidget",
                    "onDone": {"target": "Published", "actions": "commit"},
                    "onError": {"target": "Draft", "actions": "recordError"},
                },
                "after": {"persistTimeout": {"target": "Draft", "actions": "recordTimeout"}},
            },
        },
    }
    m.update(overrides)
    return m


@pytest.fixture
def machine():
    return minimal_machine()


@pytest.fixture
def write_machine(tmp_path):
    def _write(m, name="Widget"):
        p = tmp_path / f"{name}.machine.json"
        p.write_text(json.dumps(m), encoding="utf-8")
        return str(p)

    return _write

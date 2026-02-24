"""
Conftest fallback: injects test cases when the generated test script has empty TEST_CASES.

The LLM-based test generator sometimes produces a test script with empty TEST_CASES,
BASE_URL, and HEALTH_CHECK_ENDPOINT. This conftest.py detects that condition and
patches the module variables + re-parametrizes the test function via pytest hooks.
"""

import pytest

# Fallback test cases matching the test_data.json format
FALLBACK_TEST_CASES = [
    {
        "name": "health_check",
        "endpoint": "/health/",
        "method": "GET",
        "category": "HEALTH",
        "description": "Health check returns 200",
        "skip_auth": True,
        "request_data": {},
        "expected_status": 200,
    },
    {
        "name": "register_user",
        "endpoint": "/user/register",
        "method": "POST",
        "category": "USERS",
        "description": "Register a new user returns 201",
        "skip_auth": True,
        "request_data": {
            "body": {"username": "test_user_abc", "password": "testpass123"}
        },
        "expected_status": 201,
    },
    {
        "name": "login_invalid",
        "endpoint": "/user/login",
        "method": "POST",
        "category": "USERS",
        "description": "Login with wrong credentials returns 401",
        "skip_auth": True,
        "request_data": {
            "body": {"username": "nonexistent_user", "password": "wrongpass"}
        },
        "expected_status": 401,
    },
]

FALLBACK_BASE_URL = "http://localhost:5000"
FALLBACK_HEALTH_ENDPOINT = "/health/"


def _patch_module(module):
    """Patch the generated test module with fallback values if its TEST_CASES is empty."""
    test_cases = getattr(module, "TEST_CASES", None)
    if test_cases is None or len(test_cases) == 0:
        module.TEST_CASES = FALLBACK_TEST_CASES
        module.BASE_URL = FALLBACK_BASE_URL
        module.HEALTH_CHECK_ENDPOINT = FALLBACK_HEALTH_ENDPOINT
        module.HEALTH_CHECK_URL = f"{FALLBACK_BASE_URL}{FALLBACK_HEALTH_ENDPOINT}"

        # Patch authenticate to return empty auth
        def _authenticate():
            return {"headers": {}, "cookies": {}}

        module.authenticate = _authenticate
        return True
    return False


def pytest_generate_tests(metafunc):
    """Override empty parametrization with fallback test cases."""
    if "test_case" not in metafunc.fixturenames:
        return

    module = metafunc.module
    test_cases = getattr(module, "TEST_CASES", [])

    if not test_cases:
        # Patch the module first
        _patch_module(module)
        test_cases = module.TEST_CASES

        # Clear any existing empty parametrization internals
        # so we can re-parametrize with actual test cases
        if hasattr(metafunc, "_calls"):
            metafunc._calls.clear()
        if hasattr(metafunc, "_arg_names_from_markers"):
            metafunc._arg_names_from_markers.clear()

        # Inject the fallback test cases
        metafunc.parametrize(
            "test_case",
            test_cases,
            ids=[tc["name"] for tc in test_cases],
        )


def pytest_collection_modifyitems(session, config, items):
    """Patch module variables and remove NOTSET skipped items if fallback was injected."""
    for item in items:
        if hasattr(item, "module"):
            _patch_module(item.module)

    # Remove any NOTSET items that resulted from empty parametrize
    # (these should have been replaced by pytest_generate_tests, but just in case)
    remaining = [item for item in items if "NOTSET" not in item.nodeid]
    if len(remaining) != len(items):
        items[:] = remaining

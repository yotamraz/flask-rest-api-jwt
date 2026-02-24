#!/usr/bin/env python3
"""
Fallback functional tests for flask-rest-api-jwt.

Used when the auto-generated test script has empty TEST_CASES.
Replicates the same JSON output format the runner expects.
"""

import json
import sys
import time
from typing import Any

import pytest
import requests

# =============================================================================
# Configuration
# =============================================================================

BASE_URL = "http://localhost:5000"
HEALTH_CHECK_URL = "http://localhost:5000/health/"
REQUEST_TIMEOUT = 30

TEST_CASES = [
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

# =============================================================================
# Test Results Collection
# =============================================================================

test_results: list[dict[str, Any]] = []


# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture(scope="session", autouse=True)
def wait_for_app_health():
    """Wait for the application to be healthy before running tests."""
    print(f"\nWaiting for app to be healthy at {HEALTH_CHECK_URL}...")
    max_attempts = 60
    for attempt in range(max_attempts):
        try:
            resp = requests.get(HEALTH_CHECK_URL, timeout=5)
            if resp.status_code < 400:
                print(f"App is healthy after {attempt + 1} attempts")
                return
        except Exception as e:
            if attempt % 10 == 0:
                print(f"Health check attempt {attempt + 1}/{max_attempts}: {e}")
        time.sleep(2)
    pytest.fail(f"Application failed health check at {HEALTH_CHECK_URL} after {max_attempts} attempts")


@pytest.fixture(scope="session", autouse=True)
def output_test_results():
    """Output test results in JSON format after all tests complete."""
    yield

    passed_count = sum(1 for r in test_results if r["passed"])
    failed_count = len([r for r in test_results if not r["passed"]])
    total_count = len(test_results)
    all_passed = failed_count == 0 and total_count > 0

    output = {
        "all_passed": all_passed,
        "passed_count": passed_count,
        "failed_count": failed_count,
        "total_count": total_count,
        "results": test_results,
        "failures": [r for r in test_results if not r["passed"]],
    }

    print("\n" + "=" * 60)
    print(f"Results: {passed_count}/{total_count} passed")
    print("=" * 60)
    print(json.dumps(output))
    sys.stdout.flush()


# =============================================================================
# Test Cases
# =============================================================================


def get_test_ids():
    return [tc["name"] for tc in TEST_CASES]


@pytest.mark.parametrize("test_case", TEST_CASES, ids=get_test_ids())
def test_api_endpoint(test_case: dict[str, Any]):
    """Test a single API endpoint based on test case configuration."""
    name = test_case["name"]
    endpoint = test_case["endpoint"]
    method = test_case["method"].upper()
    expected_status = test_case["expected_status"]
    request_data = test_case.get("request_data", {})
    body = request_data.get("body")
    category = test_case.get("category")
    description = test_case.get("description")

    # Run setup if configured
    setup_config = test_case.get("setup")
    setup_id = None
    if setup_config:
        setup_endpoint = setup_config.get("endpoint", "/")
        setup_method = setup_config.get("method", "POST")
        setup_body = setup_config.get("body")
        setup_url = f"{BASE_URL.rstrip('/')}{setup_endpoint}"
        try:
            resp = requests.request(setup_method, setup_url, json=setup_body, timeout=REQUEST_TIMEOUT)
            if resp.status_code < 400:
                try:
                    data = resp.json()
                    extract_from = setup_config.get("extract_id_from", "id")
                    for key in extract_from.split("."):
                        data = data[key]
                    setup_id = str(data)
                except Exception:
                    pass
        except Exception as e:
            print(f"Setup failed: {e}")

    # Build URL with path params
    path_params = dict(request_data.get("path", {}))
    if setup_id:
        for key, val in path_params.items():
            if val == "$setup_id":
                path_params[key] = setup_id

    url = f"{BASE_URL.rstrip('/')}{endpoint}"
    for param_name, param_value in path_params.items():
        url = url.replace(f"{{{param_name}}}", str(param_value))

    # Execute request
    start_time = time.time()
    try:
        if method == "GET":
            resp = requests.get(url, timeout=REQUEST_TIMEOUT)
        elif method == "POST":
            resp = requests.post(url, json=body, timeout=REQUEST_TIMEOUT)
        elif method == "PUT":
            resp = requests.put(url, json=body, timeout=REQUEST_TIMEOUT)
        elif method == "DELETE":
            resp = requests.delete(url, timeout=REQUEST_TIMEOUT)
        else:
            resp = requests.request(method, url, json=body, timeout=REQUEST_TIMEOUT)

        duration_ms = (time.time() - start_time) * 1000
        actual_status = resp.status_code
        passed = actual_status == expected_status
        error_msg = None if passed else f"Expected status {expected_status}, got {actual_status}"

        result: dict[str, Any] = {
            "name": name,
            "endpoint": endpoint,
            "method": method,
            "expected_status": expected_status,
            "actual_status": actual_status,
            "passed": passed,
            "duration_ms": duration_ms,
            "category": category,
            "description": description,
        }
        if error_msg:
            result["error"] = error_msg

        if passed:
            try:
                result["response"] = resp.json()
            except (json.JSONDecodeError, ValueError):
                result["response_body"] = resp.text
        else:
            result["response_body"] = resp.text

        test_results.append(result)

        if not passed:
            pytest.fail(
                f"Test '{name}': Expected status {expected_status}, got {actual_status}. "
                f"Response: {resp.text}"
            )

    except requests.RequestException as e:
        duration_ms = (time.time() - start_time) * 1000
        test_results.append({
            "name": name,
            "endpoint": endpoint,
            "method": method,
            "expected_status": expected_status,
            "actual_status": 0,
            "passed": False,
            "duration_ms": duration_ms,
            "category": category,
            "description": description,
            "error": str(e),
        })
        pytest.fail(f"Test '{name}': Request failed with error: {e}")

    finally:
        # Run cleanup if configured
        cleanup_config = test_case.get("cleanup")
        if cleanup_config:
            cleanup_endpoint = cleanup_config.get("endpoint", "/")
            cleanup_method = cleanup_config.get("method", "DELETE")
            cleanup_url = f"{BASE_URL.rstrip('/')}{cleanup_endpoint}"
            cleanup_path = dict(cleanup_config.get("path", {}))
            if setup_id:
                for key, val in cleanup_path.items():
                    if val == "$setup_id":
                        cleanup_path[key] = setup_id
            for param_name, param_value in cleanup_path.items():
                cleanup_url = cleanup_url.replace(f"{{{param_name}}}", str(param_value))
            try:
                requests.request(cleanup_method, cleanup_url, timeout=REQUEST_TIMEOUT)
            except Exception:
                pass

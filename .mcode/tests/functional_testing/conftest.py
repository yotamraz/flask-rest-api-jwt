"""
Conftest: skips the generated test file when its TEST_CASES is empty,
allowing the fallback test file to run instead.
"""
import pathlib


def pytest_ignore_collect(collection_path, config):
    """Skip the generated m1-*-test_protocol_spec.py if it has empty TEST_CASES."""
    path_str = str(collection_path)
    if "test_protocol_spec" in path_str and path_str.endswith(".py"):
        try:
            content = pathlib.Path(collection_path).read_text()
            if "TEST_CASES = json.loads(r'''[]''')" in content:
                return True
        except Exception:
            pass
    return None

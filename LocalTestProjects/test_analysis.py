"""
Runs GoParser against every subdirectory that contains an expected.csv and
asserts the actual CSV output matches the expected values.

Usage:
    python3 test_analysis.py          # run all sub-projects
"""

import csv
import os
import subprocess
import sys
import unittest

GOPARSER_DIR = os.path.join(os.path.dirname(__file__), "..", "GoParser")
THIS_DIR = os.path.dirname(os.path.abspath(__file__))

COLUMNS = [
    "FuncTotal", "FuncGeneric",
    "MethodTotal", "MethodWithGenericReceiver",
    "MethodWithGenericReceiverTrivialTypeBound",
    "MethodWithGenericReceiverNonTrivialTypeBound",
    "StructTotal", "StructGeneric", "StructGenericNonTrivialBound", "StructAsTypeBound",
    "TypeDecl", "GenericTypeDecl", "GenericTypeSet",
    "GenericFuncInstantiationExplicit", "GenericFuncInstantiationInferred",
    "GenericTypeInstantiationExplicit", "GenericTypeInstantiationInferred",
    "GenericMethodInstantiationInferred",
]


def load_expected(csv_path: str) -> dict:
    expected = {}
    with open(csv_path, newline="") as f:
        for row in csv.DictReader(f):
            expected[row["key"]] = int(row["count"])
    return expected


def run_parser(project_path: str) -> dict:
    env = os.environ.copy()
    env["GOPARSER_SECRETS_PATH"] = "/dev/null"
    env["LOCAL_PROJECT_PATH"] = project_path
    env["ENABLE_TYPE_INFERENCE"] = "true"

    result = subprocess.run(
        ["go", "run", "."],
        cwd=GOPARSER_DIR,
        capture_output=True,
        text=True,
        env=env,
    )

    for line in result.stdout.splitlines():
        if line.startswith("local/"):
            parts = line.split(",")
            values = parts[1:]
            return {col: int(values[i]) for i, col in enumerate(COLUMNS)}

    raise RuntimeError(
        f"No CSV output found for {project_path}.\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
    )


def make_test_class(project_dir: str, expected: dict):
    project_name = os.path.basename(project_dir)

    def make_test(key, exp_val):
        def test(self):
            self.assertEqual(self.actual[key], exp_val, f"{key} mismatch")
        test.__name__ = f"test_{key.lower()}"
        return test

    attrs = {"project_dir": project_dir, "project_name": project_name}

    @classmethod
    def setUpClass(cls):
        cls.actual = run_parser(cls.project_dir)

    attrs["setUpClass"] = setUpClass

    for key, val in expected.items():
        test_fn = make_test(key, val)
        attrs[test_fn.__name__] = test_fn

    return type(f"Test_{project_name}", (unittest.TestCase,), attrs)


def discover_projects(root: str):
    projects = []
    # Check the root directory itself first
    root_csv = os.path.join(root, "expected.csv")
    if os.path.isfile(root_csv):
        projects.append((root, root_csv))
    # Then check subdirectories
    for entry in sorted(os.listdir(root)):
        full = os.path.join(root, entry)
        csv_path = os.path.join(full, "expected.csv")
        if os.path.isdir(full) and os.path.isfile(csv_path):
            projects.append((full, csv_path))
    return projects


def build_suite():
    suite = unittest.TestSuite()
    for project_dir, csv_path in discover_projects(THIS_DIR):
        expected = load_expected(csv_path)
        cls = make_test_class(project_dir, expected)
        suite.addTests(unittest.TestLoader().loadTestsFromTestCase(cls))
    return suite


if __name__ == "__main__":
    runner = unittest.TextTestRunner(verbosity=2)
    result = runner.run(build_suite())
    sys.exit(0 if result.wasSuccessful() else 1)
#!/usr/bin/env python3
"""
Integration test for LocalCrossPackageTestProject.

Runs the GoParser binary against the whole project (both packages together)
and asserts the expected counter values from the CSV output.

Usage:
    python3 test_analysis.py
    python3 -m pytest test_analysis.py -v
"""

import os
import subprocess
import unittest

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
GOPARSER_DIR = os.path.join(THIS_DIR, '..', 'GoParser')

# ── Expected CSV row (ground truth) ──────────────────────────────────────────
# local/LocalCrossPackageTestProject,3,2,3,3,3,0,1,1,0,0,1,1,0,2,0,4,0,0,5
#
# Column order matches PrintCSVHeader() in GoParser/utils/csvUtil.go:
#   Repository, FuncTotal, FuncGeneric,
#   MethodTotal, MethodWithGenericReceiver,
#   MethodWithGenericReceiverTrivialTypeBound, MethodWithGenericReceiverNonTrivialTypeBound,
#   StructTotal, StructGeneric, StructGenericNonTrivialBound, StructAsTypeBound,
#   TypeDecl, GenericTypeDecl, GenericTypeSet,
#   GenericFuncInstantiationExplicit, GenericFuncInstantiationInferred,
#   GenericTypeInstantiationExplicit, GenericTypeInstantiationInferred,
#   GenericMethodInstantiationExplicit, GenericMethodInstantiationInferred


def run_analysis(project_path: str) -> dict:
    """Run GoParser against *project_path* and return the parsed CSV counters."""
    env = os.environ.copy()
    env['GOPARSER_SECRETS_PATH'] = '/dev/null'   # skip secret.env
    env['LOCAL_PROJECT_PATH'] = os.path.abspath(project_path)
    env['ENABLE_TYPE_INFERENCE'] = 'true'

    result = subprocess.run(
        ['go', 'run', '.'],
        cwd=os.path.abspath(GOPARSER_DIR),
        env=env,
        capture_output=True,
        text=True,
        timeout=120,
    )

    if result.returncode != 0:
        raise RuntimeError(
            f"GoParser exited with code {result.returncode}\n"
            f"stderr:\n{result.stderr}\nstdout:\n{result.stdout}"
        )

    # stdout: CSV header line + CSV data line (+ instantiation summary lines)
    # stderr: log messages (timestamps) – ignored here
    header = None
    data = None
    for line in result.stdout.splitlines():
        if line.startswith('Repository,'):
            header = line.split(',')
        elif header and line.startswith('local/'):
            data = line.split(',')
            break

    if header is None or data is None:
        raise ValueError(
            f"Could not parse CSV output.\n"
            f"stdout:\n{result.stdout}\nstderr:\n{result.stderr}"
        )

    return {k: int(v) for k, v in zip(header[1:], data[1:])}


class TestLocalCrossPackageTestProject(unittest.TestCase):
    """
    Asserts the aggregated counter values for the whole LocalCrossPackageTestProject.

    Package layout:
      FirstPackage/  – defines Stack[T], Push/Pop/Len methods, NewStack[T], Map[T,R]
      SecondPackage/ – imports FirstPackage and instantiates Stack, NewStack, Map

    Key cross-package instantiations (require the import-path fix in astAnalyzer.go):
      GenericTypeInstantiationExplicit  +2  (&fp.Stack[int]{}, &fp.Stack[string]{})
      GenericFuncInstantiationExplicit  +2  (fp.NewStack[string](), fp.Map[int,string](...))
      GenericMethodInstantiationInferred +4  (Push×3, Pop×1 in SecondPackage)
    """

    @classmethod
    def setUpClass(cls):
        cls.c = run_analysis(THIS_DIR)

    # ── Basic counts (all from FirstPackage definitions) ──────────────────────

    def test_func_total(self):
        # NewStack, Map (FirstPackage) + UseStackFromFirstPackage (SecondPackage)
        self.assertEqual(self.c['FuncTotal'], 3)

    def test_func_generic(self):
        # NewStack[T any], Map[T, R any]
        self.assertEqual(self.c['FuncGeneric'], 2)

    def test_method_total(self):
        # Push, Pop, Len on Stack[T]
        self.assertEqual(self.c['MethodTotal'], 3)

    def test_method_with_generic_receiver(self):
        # All 3 methods have *Stack[T] receiver
        self.assertEqual(self.c['MethodWithGenericReceiver'], 3)

    def test_method_trivial_bound(self):
        # Stack[T any] – 'any' is trivial
        self.assertEqual(self.c['MethodWithGenericReceiverTrivialTypeBound'], 3)

    def test_method_non_trivial_bound(self):
        self.assertEqual(self.c['MethodWithGenericReceiverNonTrivialTypeBound'], 0)

    def test_struct_total(self):
        self.assertEqual(self.c['StructTotal'], 1)

    def test_struct_generic(self):
        # Stack[T any]
        self.assertEqual(self.c['StructGeneric'], 1)

    def test_struct_generic_non_trivial_bound(self):
        self.assertEqual(self.c['StructGenericNonTrivialBound'], 0)

    def test_struct_as_type_bound(self):
        self.assertEqual(self.c['StructAsTypeBound'], 0)

    def test_type_decl(self):
        self.assertEqual(self.c['TypeDecl'], 1)

    def test_generic_type_decl(self):
        self.assertEqual(self.c['GenericTypeDecl'], 1)

    def test_generic_type_set(self):
        self.assertEqual(self.c['GenericTypeSet'], 0)

    # ── Instantiation counts ──────────────────────────────────────────────────

    def test_generic_func_instantiation_explicit(self):
        # fp.NewStack[string]() and fp.Map[int, string](...) in SecondPackage
        self.assertEqual(self.c['GenericFuncInstantiationExplicit'], 2)

    def test_generic_func_instantiation_inferred(self):
        self.assertEqual(self.c['GenericFuncInstantiationInferred'], 0)

    def test_generic_type_instantiation_explicit(self):
        # &Stack[T]{} in NewStack body          (FirstPackage-internal)
        # &Stack[R]{} in Map body               (FirstPackage-internal)
        # &fp.Stack[int]{} in SecondPackage      (cross-package)
        # &fp.Stack[string]{} in SecondPackage   (cross-package)
        self.assertEqual(self.c['GenericTypeInstantiationExplicit'], 4)

    def test_generic_type_instantiation_inferred(self):
        self.assertEqual(self.c['GenericTypeInstantiationInferred'], 0)

    def test_generic_method_instantiation_explicit(self):
        self.assertEqual(self.c['GenericMethodInstantiationExplicit'], 0)

    def test_generic_method_instantiation_inferred(self):
        # out.Push(f(item)) in Map body          (FirstPackage-internal, parametric)
        # intStack.Push(1), intStack.Push(2)     (SecondPackage, cross-package)
        # intStack.Pop()                         (SecondPackage, cross-package)
        # strStack.Push("hello")                 (SecondPackage, cross-package)
        self.assertEqual(self.c['GenericMethodInstantiationInferred'], 5)


if __name__ == '__main__':
    unittest.main(verbosity=2)
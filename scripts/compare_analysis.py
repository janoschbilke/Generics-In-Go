#!/usr/bin/env python3
"""
compare_analysis.py – Compare two git refs of the GoParser against the same
set of GitHub repositories.

Usage:
    python3 scripts/compare_analysis.py \\
        --old-ref  main \\
        --new-ref  feature/project-wide \\
        --repos    input/repos.csv \\
        --count    10 \\
        --token    $GITHUB_TOKEN \\
        --output   output/comparison.csv

What it does:
  1. Remembers the current git branch / commit.
  2. git stash (if the working tree is dirty).
  3. Checks out --old-ref, builds the GoParser binary, runs it for every repo,
     saves results.
  4. Checks out --new-ref, builds the GoParser binary, runs it for every repo,
     saves results.
  5. Restores the original branch and pops the stash.
  6. Writes a side-by-side comparison CSV and prints a summary table.

Requirements:
  - Run from anywhere; the script locates the repo root via the location of
    this file.
  - GITHUB_TOKEN must be set via --token or the environment variable.
"""

import argparse
import csv
import os
import subprocess
import sys
import tempfile
import time
from pathlib import Path

# ── Paths ─────────────────────────────────────────────────────────────────────
REPO_ROOT = Path(__file__).resolve().parent.parent
GOPARSER_DIR = REPO_ROOT / "GoParser"
DEFAULT_INPUT = REPO_ROOT / "input" / "repos.csv"
DEFAULT_OUTPUT = REPO_ROOT / "output" / "comparison.csv"

# ── Git helpers ───────────────────────────────────────────────────────────────

def git(*args, check=True, capture=True):
    """Run a git command in the repo root."""
    result = subprocess.run(
        ["git", *args],
        cwd=str(REPO_ROOT),
        capture_output=capture,
        text=True,
        check=False,
    )
    if check and result.returncode != 0:
        print(f"[git error] git {' '.join(args)}", file=sys.stderr)
        print(result.stderr.strip(), file=sys.stderr)
        sys.exit(1)
    return result


def current_ref() -> str:
    """Return the current branch name, or the commit hash if detached HEAD."""
    r = git("symbolic-ref", "--short", "HEAD", check=False)
    if r.returncode == 0:
        return r.stdout.strip()
    return git("rev-parse", "HEAD").stdout.strip()


def is_dirty() -> bool:
    return git("status", "--porcelain").stdout.strip() != ""


# ── Build helper ──────────────────────────────────────────────────────────────

def build_binary(dest: Path) -> None:
    """Build the GoParser binary into dest."""
    print(f"  Building GoParser → {dest} ...", flush=True)
    result = subprocess.run(
        ["go", "build", "-o", str(dest), "."],
        cwd=str(GOPARSER_DIR),
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print("[build error]", result.stderr[:800], file=sys.stderr)
        sys.exit(1)


# ── Run helper ────────────────────────────────────────────────────────────────

def run_binary(binary: Path, repo_csv: Path, token: str, tmpdir: Path) -> tuple[dict[str, dict], float]:
    """
    Run the GoParser binary for the repos listed in repo_csv.
    Returns (results_by_repo, elapsed_seconds).
    """
    env = os.environ.copy()
    env["INPUT_CSV_PATH"] = str(repo_csv)
    env["GITHUB_TOKEN"] = token
    env["ENABLE_TYPE_INFERENCE"] = "true"
    env["OUTPUT_DB_PATH"] = str(tmpdir / "out.db")
    env["OUTPUT_CSV_PATH"] = str(tmpdir / "out.csv")
    # Prevent the binary from loading secret.env so our env vars win
    env["GOPARSER_SECRETS_PATH"] = "/dev/null"

    t0 = time.time()
    result = subprocess.run(
        [str(binary)],
        env=env,
        capture_output=True,
        text=True,
    )
    elapsed = time.time() - t0

    if result.returncode != 0:
        print(f"  [WARN] binary exited {result.returncode}", file=sys.stderr)
        print(result.stderr[:400], file=sys.stderr)

    rows: dict[str, dict] = {}
    for row in csv.DictReader(result.stdout.splitlines()):
        repo = row.get("Repository", "")
        if repo:
            rows[repo] = row
    return rows, elapsed


# ── Comparison helpers ────────────────────────────────────────────────────────

COUNTER_FIELDS = [
    "FuncTotal", "FuncGeneric",
    "MethodTotal", "MethodWithGenericReceiver",
    "StructTotal", "StructGeneric",
    "TypeDecl", "GenericTypeDecl", "GenericTypeSet",
    "GenericFuncInstantiationExplicit", "GenericFuncInstantiationInferred",
    "GenericTypeInstantiationExplicit", "GenericTypeInstantiationInferred",
    "GenericMethodInstantiationExplicit", "GenericMethodInstantiationInferred",
]


def build_row(repo: str, old: dict, old_time: float, new: dict, new_time: float) -> dict:
    row: dict = {
        "Repository": repo,
        "OldTime_s": f"{old_time:.2f}",
        "NewTime_s": f"{new_time:.2f}",
    }
    for f in COUNTER_FIELDS:
        o = int(old.get(f, 0) or 0)
        n = int(new.get(f, 0) or 0)
        row[f"old_{f}"] = o
        row[f"new_{f}"] = n
        row[f"diff_{f}"] = n - o
    return row


# ── Main ──────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(
        description="Compare two git refs of GoParser on the same repo set."
    )
    parser.add_argument("--old-ref", required=True,
                        help="Git ref for the 'before' state (branch, tag, commit)")
    parser.add_argument("--new-ref", required=True,
                        help="Git ref for the 'after' state (branch, tag, commit)")
    parser.add_argument("--repos", default=str(DEFAULT_INPUT),
                        help=f"CSV with repos (default: {DEFAULT_INPUT})")
    parser.add_argument("--count", type=int, default=10,
                        help="Number of repos to analyse (default: 10)")
    parser.add_argument("--token", default=os.environ.get("GITHUB_TOKEN", ""),
                        help="GitHub API token (or set GITHUB_TOKEN env var)")
    parser.add_argument("--output", default=str(DEFAULT_OUTPUT),
                        help=f"Output CSV (default: {DEFAULT_OUTPUT})")
    args = parser.parse_args()

    if not args.token:
        print("ERROR: GitHub token required (--token or GITHUB_TOKEN env var)", file=sys.stderr)
        sys.exit(1)

    repos_path = Path(args.repos)
    if not repos_path.exists():
        print(f"ERROR: repos CSV not found: {repos_path}", file=sys.stderr)
        sys.exit(1)

    # Read repo list
    with open(repos_path) as f:
        reader = csv.reader(f)
        next(reader, None)  # skip header
        all_rows = [r for r in reader if len(r) >= 2]
    selected = all_rows[: args.count]
    if not selected:
        print("ERROR: no repos found in CSV", file=sys.stderr)
        sys.exit(1)
    print(f"Repos to analyse: {len(selected)}")

    # Build a temporary single-repo CSV for each repo
    # (we run the binary once per ref for ALL repos together for speed)
    with tempfile.NamedTemporaryFile(mode="w", suffix=".csv", delete=False) as tmp:
        tmp.write("rank,repository\n")
        for i, row in enumerate(selected, 1):
            # Support "github.com/owner/repo" or "owner,repo" formats
            if row[0].startswith("github.com/"):
                tmp.write(f"{i},{row[0]}\n")
            else:
                tmp.write(f"{i},github.com/{row[0].strip()}/{row[1].strip()}\n")
        tmp_csv = Path(tmp.name)

    original_ref = current_ref()
    stashed = False

    try:
        # Stash dirty working tree
        if is_dirty():
            print("Working tree is dirty – stashing changes ...")
            git("stash", "push", "--include-untracked", "-m", "compare_analysis_autostash")
            stashed = True

        results: dict[str, tuple[dict, float]] = {}  # ref → (rows_by_repo, elapsed)

        for ref in (args.old_ref, args.new_ref):
            print(f"\n── Checking out {ref} ──")
            git("checkout", ref)

            with tempfile.TemporaryDirectory() as tmpdir:
                binary = Path(tmpdir) / "goparser"
                build_binary(binary)

                print(f"  Running analysis for {len(selected)} repos ...", flush=True)
                rows, elapsed = run_binary(binary, tmp_csv, args.token, Path(tmpdir))
                results[ref] = (rows, elapsed)
                print(f"  Done in {elapsed:.1f}s – got results for {len(rows)} repos")

    finally:
        # Always restore original state
        print(f"\nRestoring {original_ref} ...")
        git("checkout", original_ref, check=False)
        if stashed:
            git("stash", "pop", check=False)
        tmp_csv.unlink(missing_ok=True)

    old_rows, old_time = results[args.old_ref]
    new_rows, new_time = results[args.new_ref]

    # Build comparison
    comparison: list[dict] = []
    all_repos = sorted(set(old_rows) | set(new_rows))
    for repo in all_repos:
        comparison.append(build_row(
            repo,
            old_rows.get(repo, {}), old_time / max(len(old_rows), 1),
            new_rows.get(repo, {}), new_time / max(len(new_rows), 1),
        ))

    # Write CSV
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    if comparison:
        with open(output_path, "w", newline="") as f:
            writer = csv.DictWriter(f, fieldnames=list(comparison[0].keys()))
            writer.writeheader()
            writer.writerows(comparison)
        print(f"\nComparison written to {output_path}")

    # ── Summary table ──────────────────────────────────────────────────────
    inst_fields = [
        "GenericFuncInstantiationExplicit", "GenericFuncInstantiationInferred",
        "GenericTypeInstantiationExplicit", "GenericTypeInstantiationInferred",
        "GenericMethodInstantiationExplicit", "GenericMethodInstantiationInferred",
    ]

    print(f"\n{'Repository':<45} {'Old inst':>9} {'New inst':>9} {'Diff':>6}")
    print("-" * 75)
    total_old = total_new = 0
    for r in comparison:
        o = sum(int(r.get(f"old_{f}", 0)) for f in inst_fields)
        n = sum(int(r.get(f"new_{f}", 0)) for f in inst_fields)
        total_old += o
        total_new += n
        marker = " ▲" if n > o else ("" if n == o else " ▼")
        print(f"{r['Repository']:<45} {o:>9} {n:>9} {n-o:>+6}{marker}")
    print("-" * 75)
    print(f"{'TOTAL':<45} {total_old:>9} {total_new:>9} {total_new-total_old:>+6}")
    print(f"\nTotal analysis time  old={old_time:.1f}s  new={new_time:.1f}s  "
          f"speedup={old_time/max(new_time,0.001):.2f}×")


if __name__ == "__main__":
    main()
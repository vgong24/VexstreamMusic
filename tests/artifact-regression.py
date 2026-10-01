#!/usr/bin/env python3
from __future__ import annotations
import hashlib, struct, sys
from pathlib import Path

SOURCE_SHA256 = "e5c06539522bed4e265f7fca6b90cbf13a76d6220bfbec2065b225ca1455ed7f"
OUTPUT_SHA256 = "1050a2364c617bce93663b21046ffb7419e96a231f748b02e9166c3091a1a4cc"
OUTPUT_UI_SHA256 = "7c8292b290b5299beb05a53e059d2abdec75d0c199f8f7b3a436a82cb40e240f"
OLD_VERSION = b"1.1.5"
BAD_INTERMEDIATE_VERSION = b"1.1.6"
NEW_VERSION = b"1.1.7"
HTML_START = b'<html lang="en">'
HTML_END = b'</html>'
LAUNCH_GUARD_115 = bytes.fromhex("4883fb05752c8138312e312e752480780435751e")
LAUNCH_GUARD_117 = bytes.fromhex("4883fb05752c8138312e312e752480780437751e")
LAUNCH_GUARD_OFFSET = 0x2CD120
OUTSIDE_UI_DIFF_OFFSETS = {0x2CD131, 0x39C751, 0x3BDFCA}

def sha(data): return hashlib.sha256(data).hexdigest()
def fail(message): raise AssertionError(message)

def html_range(data):
    if data.count(HTML_START) != 1 or data.count(HTML_END) != 1:
        fail("artifact must contain exactly one embedded HTML document")
    start = data.index(HTML_START)
    return start, data.index(HTML_END, start) + len(HTML_END)

def pe_machine(data):
    if data[:2] != b"MZ": fail("artifact does not begin with MZ")
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    if data[pe:pe+4] != b"PE\0\0": fail("artifact has no PE signature")
    return struct.unpack_from("<H", data, pe + 4)[0]

def main():
    if len(sys.argv) != 4:
        print("usage: artifact-regression.py <source-1.1.5.exe> <candidate-1.1.7.exe> <repo-ui.html>", file=sys.stderr)
        return 2
    source_path, candidate_path, repo_ui_path = map(Path, sys.argv[1:])
    source, candidate, repo_ui = source_path.read_bytes(), candidate_path.read_bytes(), repo_ui_path.read_bytes()
    checks = []
    checks.append(("source exact SHA-256", sha(source) == SOURCE_SHA256))
    checks.append(("candidate exact SHA-256", sha(candidate) == OUTPUT_SHA256))
    checks.append(("candidate byte length preserved", len(candidate) == len(source) == 6903808))
    checks.append(("candidate remains x86-64 PE", pe_machine(candidate) == 0x8664))
    s0,s1 = html_range(source); c0,c1 = html_range(candidate)
    checks.append(("embedded UI range preserved", (s0,s1) == (c0,c1)))
    embedded = candidate[c0:c1]
    checks.append(("candidate embedded UI SHA-256", sha(embedded) == OUTPUT_UI_SHA256))
    checks.append(("repository UI equals shipped embedded UI", repo_ui == embedded))
    checks.append(("no contiguous 1.1.5 marker survives", candidate.count(OLD_VERSION) == 0))
    checks.append(("failed 1.1.6 marker does not survive", candidate.count(BAD_INTERMEDIATE_VERSION) == 0))
    checks.append(("three contiguous 1.1.7 markers present", candidate.count(NEW_VERSION) == 3))
    checks.append(("old compiled launch guard absent", candidate.count(LAUNCH_GUARD_115) == 0))
    checks.append(("1.1.7 compiled launch guard unique", candidate.count(LAUNCH_GUARD_117) == 1))
    checks.append(("1.1.7 launch guard remains at qualified code offset", candidate.find(LAUNCH_GUARD_117) == LAUNCH_GUARD_OFFSET))
    diffs = {i for i,(a,b) in enumerate(zip(source,candidate)) if a != b and not (c0 <= i < c1)}
    checks.append(("non-UI diffs are exact version-only set", diffs == OUTSIDE_UI_DIFF_OFFSETS))
    checks.append(("all non-UI diffs are ASCII 5 -> 7", all(source[i] == ord("5") and candidate[i] == ord("7") for i in diffs)))
    failed = [name for name,ok in checks if not ok]
    for name,ok in checks: print(("PASS" if ok else "FAIL") + "  " + name)
    print(f"SUMMARY {len(checks)-len(failed)}_PASS__{len(failed)}_FAIL")
    if failed: raise AssertionError("; ".join(failed))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())

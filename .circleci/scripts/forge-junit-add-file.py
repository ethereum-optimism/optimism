#!/usr/bin/env python3
"""Add a `file` attribute to every testcase in the `forge test --junit` reports in a directory.

`circleci tests split --split-by=timings` looks up timing history by each stored testcase's
`file` attribute. Forge records the file only in the suite name (`test/L1/Foo.t.sol:FooTest`),
so without this the split finds no history and falls back to splitting by name.

The parser only locates each <testcase> start tag. The attribute is inserted there, and every
other byte stays as forge wrote it. This only feeds test splitting, so it never fails the job:
a report it can't handle is left unchanged.

Usage: forge-junit-add-file.py <results-dir>
"""

import os
import pathlib
import sys
from xml.parsers import expat
from xml.sax.saxutils import quoteattr


def add_file(report: bytes) -> tuple[bytes, int]:
    """Returns the report with `file` on each testcase, and the number of testcases changed."""
    suites: list[str] = []
    inserts: list[tuple[int, str]] = []
    # Forge writes UTF-8. Forcing it makes any other encoding fail to parse, so the offsets
    # below are always byte offsets into UTF-8 text.
    parser = expat.ParserCreate("UTF-8")

    def start(tag: str, attrs: dict[str, str]) -> None:
        if tag == "testsuite":
            # The path is everything before the last colon: `test/L1/Foo.t.sol:FooTest`.
            path, sep, _ = attrs.get("name", "").rpartition(":")
            suites.append(path if sep else "")
        elif tag == "testcase" and suites and suites[-1] and "file" not in attrs:
            # CurrentByteIndex is the offset of this start tag's `<`.
            offset = parser.CurrentByteIndex + len(b"<testcase")
            if report[offset - len(b"<testcase") : offset] != b"<testcase":
                raise ValueError(f"no <testcase tag at byte {offset}")
            inserts.append((offset, suites[-1]))

    def end(tag: str) -> None:
        if tag == "testsuite":
            suites.pop()

    def declaration(version: str, encoding: str | None, standalone: int) -> None:
        if encoding and encoding.upper() not in ("UTF-8", "UTF8"):
            raise ValueError(f"declared encoding {encoding} is not UTF-8")

    parser.XmlDeclHandler = declaration
    parser.StartElementHandler = start
    parser.EndElementHandler = end
    parser.Parse(report, True)
    for offset, path in reversed(inserts):
        report = report[:offset] + b" file=" + quoteattr(path).encode() + report[offset:]
    return report, len(inserts)


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: forge-junit-add-file.py <results-dir>")
    try:
        reports = sorted(pathlib.Path(sys.argv[1]).glob("*.xml"))
    except OSError as err:
        print(f"forge-junit-add-file: can't list {sys.argv[1]}: {err}")
        return
    if not reports:
        print(f"forge-junit-add-file: no JUnit reports in {sys.argv[1]}")
    for path in reports:
        tmp = path.with_name(path.name + ".tmp")
        try:
            updated, cases = add_file(path.read_bytes())
            tmp.write_bytes(updated)
            os.replace(tmp, path)
            print(f"forge-junit-add-file: {path}: added file to {cases} testcase(s)")
        except Exception as err:  # Best effort: never fail the job over test-splitting data.
            print(f"forge-junit-add-file: {path}: left unchanged: {err!r}")
        finally:
            try:
                tmp.unlink(missing_ok=True)
            except OSError as err:
                print(f"forge-junit-add-file: can't remove {tmp}: {err!r}")


if __name__ == "__main__":
    main()

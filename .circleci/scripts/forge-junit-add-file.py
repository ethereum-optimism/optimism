#!/usr/bin/env python3
"""Add a `file` attribute to every testcase in a `forge test --junit` report.

`circleci tests split --split-by=timings` matches the names it splits against the
`file` attribute of stored testcases. Forge only records the file in the suite
name (`test/L1/Foo.t.sol:FooTest`), so without this the split never finds timing
data and falls back to splitting by name.

Usage: forge-junit-add-file.py <junit-report.xml>
"""

import sys
import xml.etree.ElementTree as ET


def main() -> None:
    if len(sys.argv) != 2:
        sys.exit("usage: forge-junit-add-file.py <junit-report.xml>")
    path = sys.argv[1]
    tree = ET.parse(path)
    for suite in tree.getroot().iter("testsuite"):
        file = suite.get("name", "").split(":", 1)[0]
        if not file:
            continue
        for case in suite.iter("testcase"):
            case.set("file", file)
    tree.write(path, encoding="utf-8", xml_declaration=True)


if __name__ == "__main__":
    main()

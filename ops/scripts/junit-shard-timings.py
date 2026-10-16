#!/usr/bin/env python3
"""Include parallel subtest costs in gotestsum JUnit timings used for sharding."""

import sys
import xml.etree.ElementTree as ET


def aggregate_timings(tree):
    for suite in tree.iter("testsuite"):
        cases = {case.attrib["name"]: case for case in suite.findall("testcase")}
        elapsed = {name: float(case.get("time", "0")) for name, case in cases.items()}
        names = set()
        for name in cases:
            parts = name.split("/")
            names.update("/".join(parts[:i]) for i in range(1, len(parts) + 1))

        children = {}
        for name in sorted(names, key=lambda name: name.count("/"), reverse=True):
            # Go excludes time spent waiting for parallel subtests from a
            # parent's elapsed time. Serial subtests are already included, so
            # take the larger value rather than adding the parent's time.
            duration = max(elapsed.get(name, 0), children.get(name, 0))
            if name in cases and elapsed[name] >= 0 and duration > elapsed[name]:
                cases[name].set("time", f"{duration:.6f}")
            if "/" in name:
                parent = name.rsplit("/", 1)[0]
                children[parent] = children.get(parent, 0) + duration


if __name__ == "__main__":
    tree = ET.parse(sys.argv[1])
    aggregate_timings(tree)
    tree.write(sys.argv[1], encoding="utf-8", xml_declaration=True)

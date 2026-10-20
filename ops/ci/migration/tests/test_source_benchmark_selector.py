import copy
import importlib.util
from pathlib import Path
import unittest

PATH = Path(__file__).resolve().parents[1] / "select-source-benchmark.py"
spec = importlib.util.spec_from_file_location("source_benchmark_selector", PATH)
selector = importlib.util.module_from_spec(spec)
spec.loader.exec_module(selector)


class SelectorTests(unittest.TestCase):
    def config(self):
        return {
            "version": 2.1,
            "jobs": {"original": {"resource_class": "large", "steps": ["checkout"]}},
            "commands": {"original": {"steps": ["checkout"]}},
            "parameters": {"unchanged": {"type": "boolean", "default": False}},
            "workflows": {
                "main": {"jobs": [{"prep": {}}, {"go-tests": {"requires": ["prep"], "parallelism": 12}}, "unrelated"]},
                "rust-ci": {"jobs": ["rust-tests", {"rust-ci-cargo-hack": {"name": "rust-cargo-hack", "parallelism": 10}}]},
                "contracts-feature-tests": {"jobs": ["prep-go-modules", {"contracts-bedrock-tests": {
                    "name": "contracts-bedrock-tests <<matrix.features>>", "features": "<<matrix.features>>",
                    "matrix": {"parameters": {"features": ["main", "OTHER"]}}, "requires": ["prep-go-modules"],
                    "test_profile": "liteci", "test_list": "find test -name '*.t.sol'"}}]},
            },
        }

    def test_preserves_definitions_and_selects_transitive_dependencies(self):
        original = self.config()
        result = selector.select(copy.deepcopy(original), "go")
        self.assertEqual({k: v for k, v in result.items() if k != "workflows"},
                         {k: v for k, v in original.items() if k != "workflows"})
        jobs = result["workflows"]["benchmark-go"]["jobs"]
        self.assertEqual(jobs[:2], original["workflows"]["main"]["jobs"][:2])
        self.assertNotIn("unrelated", jobs)

    def test_emits_four_gates_with_real_selected_dependencies(self):
        for language, gate, targets in [("go", "ci-gate", ["go-tests"]),
                                        ("rust", "required-rust-ci", ["rust-tests", "rust-cargo-hack"]),
                                        ("solidity", "required-contracts-ci", ["contracts-bedrock-tests main"])]:
            with self.subTest(language=language):
                jobs = selector.select(self.config(), language)["workflows"]["benchmark-" + language]["jobs"]
                gates = dict(selector.identity(j) for j in jobs[-4:])
                self.assertEqual(gates[gate]["requires"], [{t: "terminal"} for t in targets])
                self.assertNotIn("always-succeed", gates[gate])
                for name, params in gates.items():
                    if name != gate:
                        self.assertTrue(params["always-succeed"])
                        self.assertNotIn("requires", params)

    def test_solidity_expands_only_original_main_matrix_member(self):
        result = selector.select(self.config(), "solidity")
        kind, params = selector.identity(result["workflows"]["benchmark-solidity"]["jobs"][1])
        self.assertEqual(kind, "contracts-bedrock-tests")
        self.assertEqual(params["features"], "main")
        self.assertEqual(params["test_profile"], "liteci")
        self.assertEqual(params["test_list"], "find test -name '*.t.sol'")
        self.assertNotIn("matrix", params)

    def test_missing_prerequisite_fails_closed(self):
        config = self.config()
        config["workflows"]["main"]["jobs"].pop(0)
        with self.assertRaises(KeyError):
            selector.select(config, "go")

    def test_normal_branches_are_outside_benchmark_selection(self):
        self.assertNotIn("develop", selector.BRANCHES)
        self.assertNotIn("codex/rwx-ci-pilot", selector.BRANCHES)
        self.assertNotIn("codex/rwx-source-benchmark-base", selector.BRANCHES)
        self.assertEqual(len(selector.BRANCHES), 3)


if __name__ == "__main__":
    unittest.main()

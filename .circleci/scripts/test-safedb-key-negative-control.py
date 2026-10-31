import importlib.util
from pathlib import Path
import tempfile
import unittest

specification = importlib.util.spec_from_file_location("control", Path(__file__).with_name("verify-safedb-key-negative-control.py"))
control = importlib.util.module_from_spec(specification)
specification.loader.exec_module(control)


class NegativeControlTests(unittest.TestCase):
    def test_positive_requires_all_tests_to_pass(self):
        for action in ["fail", "pass", "skip"]:
            with self.subTest(action=action):
                result = control.validate_events([{"Action": action, "Test": control.STEADY}], 0, [control.STEADY])
                self.assertEqual(result["valid"], action == "pass")
        self.assertFalse(control.validate_events([], 0, [control.STEADY])["valid"])

    def test_negative_control_requires_test_failure_and_specific_diagnostic(self):
        for exit_code, action, message, expected in [(1, "fail", "Mismatched safe head data", True), (0, "pass", "", False), (1, "fail", "startup failure", False), (1, "skip", "Mismatched safe head data", False), (0, "fail", "Mismatched safe head data", False)]:
            with self.subTest(exit_code=exit_code, action=action, message=message):
                events = [{"Action": action, "Test": control.STEADY}, {"Action": "output", "Test": control.STEADY, "Output": message}]
                self.assertEqual(control.validate_events(events, exit_code, [control.STEADY], True)["valid"], expected)

    def test_source_restored_even_when_test_raises(self):
        with tempfile.TemporaryDirectory() as temporary:
            source = Path(temporary) / "safedb.go"
            original = "func (d *SafeDB) SafeHeadUpdated(safeHead eth.L2BlockRef, l1Head eth.BlockID) error {\n}\n"
            source.write_text(original)
            with self.assertRaisesRegex(RuntimeError, "injected failure"):
                with control.wrong_record_key(source):
                    self.assertIn("l1Head = safeHead.L1Origin", source.read_text())
                    raise RuntimeError("injected failure")
            self.assertEqual(source.read_text(), original)


if __name__ == "__main__":
    unittest.main()

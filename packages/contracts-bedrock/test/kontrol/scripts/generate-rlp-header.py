"""Expose the actual private helper without editing production source."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parents[3]
source = root / "src/libraries/rlp/RLPReader.sol"
original = source.read_text()
declaration = "library RLPReader {"
renamed = "library RLPReaderHeaderHarness {"
wrapper = """
    function decodeHeader(RLPItem memory _in)
        internal pure returns (uint256 offset_, uint256 length_, RLPItemType type_)
    {
        return _decodeLength(_in);
    }
"""
if original.count(declaration) != 1 or not original.endswith("}\n"):
    raise SystemExit("Unexpected production library structure")
generated = original.replace(declaration, renamed)[:-2] + wrapper + "}\n"
inverse = generated.replace(wrapper, "").replace(renamed, declaration)
if inverse != original:
    raise SystemExit("Generated library does not restore the exact production source")
output = root / "test/kontrol/rlp-header/generated"
output.mkdir(parents=True, exist_ok=True)
(output / "RLPReaderHeaderHarness.sol").write_text(generated)
template = root / "test/kontrol/rlp-header/RLPHeader.k.sol.in"
(output / "RLPHeader.k.sol").write_text(template.read_text())
hashes = {"production": hashlib.sha256(original.encode()).hexdigest(),
          "generated": hashlib.sha256(generated.encode()).hexdigest(),
          "fixture": hashlib.sha256(template.read_bytes()).hexdigest(),
          "inverse_source_matches": True}
logs = root / "test/kontrol/logs/rlp-header"
logs.mkdir(parents=True, exist_ok=True)
(logs / "source-provenance.json").write_text(json.dumps(hashes, indent=2) + "\n")
print(json.dumps(hashes))

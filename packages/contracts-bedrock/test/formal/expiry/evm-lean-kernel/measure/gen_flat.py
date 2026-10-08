import sys
code = bytes.fromhex(open(sys.argv[1]).read().strip().removeprefix("0x"))
body = ", ".join(f"0x{b:02x}" for b in code)
lines=[]; cur=""
for p in body.split(", "):
    if len(cur)+len(p)+2>96: lines.append(cur); cur=p
    else: cur = p if not cur else cur+", "+p
lines.append(cur)
print("import ExpiryEvm.Bytecode\n\nnamespace ExpiryEvm\n\nset_option maxRecDepth 50000000\n")
print(f"/-- `l2tol2Runtime` as one flat literal ({len(code)} bytes). -/")
print("def l2tol2RuntimeFlat : ByteArray :=\n  ⟨#[" + ",\n    ".join(lines) + "]⟩\n")
print("end ExpiryEvm")

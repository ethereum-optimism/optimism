# Proposal Validator Invariants

| Invariant | Description |
|-----------|-------------|
| VAL-001 | sum of all funding proposals amount moved to vote during a cycle <= that cycle distribution limit |
| VAL-002 | approvalCount == number of delegates having approved it |
| VAL-003 | proposal's proposer and type are immutable once submitted (ie hash generated == immutable) |
| VAL-004 | proposal are going to vote only if approval (+ time window for funding proposals and council member elections) + "additional" checks have passed |

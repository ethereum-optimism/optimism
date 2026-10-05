# Native flaky-test reporting

The optional `optimism-flaky-report-shadow` executes the original acceptance
reporting script through shared `run-main` routing. Complete same-SHA hosted
parity passed at `b472a22e8374c797ecaee5a6ef5b735fcc979028`: native run
`5620ba604af94bd38b1a710469ebbf4b`, Circle pipeline 135620/job 5637736.
The [parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-flaky-report-evidence/parity.json) binds all 16 original
files from each provider. This adds one verified Main occurrence.

The workload requests the real Circle Insights flaky-test API for this public
repository. Circle retains its existing authenticated context; RWX uses the
public endpoint anonymously and receives no Circle credential. The API is
project-wide and branch agnostic. The branch label identifies the requesting CI
branch rather than claiming that the API selected observations from that branch.

Both providers retain the entire unfiltered response, each original HTTP attempt
and failure body, the existing acceptance-only JSON, CSV, HTML and top-ten text
reports, complete stdout/stderr, source hashes and actual process metadata.
Validation independently derives every selected row from the unfiltered source
and rejects missing, duplicated or corrupt inputs. HTTP authorization/path errors
fail immediately; transient failures have six attempts with bounded backoff.
Cancellation retains a failed process verdict and never produces report credit.

`c-flaky_report_replay` defaults to false. An explicit pilot API replay with
`main_dispatch=false` selects only the original reporter. It can also run beside
the isolated selector replay. Fixtures verify normal push/main dispatch behavior,
the pilot branch guard and the complete selected workflow set. Protected
`develop` cache warming builds tools and executes zero reports or tests.

The anonymous live preflight returned all 79 source observations and all 12
acceptance observations, with every derived report checked. The retained API
fixture comes from that real response; fixture executions add zero hosted
coverage. Wrapper tests execute actual shell and child processes to verify
successful generation, original HTTP failure capture and cancellation.

Run the strict hosted comparison after collecting both complete report trees,
Circle's full job/config/log originals, the native run metadata and the same-SHA
GitHub check snapshot:

```sh
mise exec yq@4.44.5 -- python3 ops/ci/compare-flaky-report.py \
  --circle .ci/rwx-flaky-evidence/SHA/circle \
  --native .ci/rwx-flaky-evidence/SHA/native/report \
  --run .ci/rwx-flaky-evidence/SHA/native-run.json \
  --github .ci/rwx-flaky-evidence/SHA/github.json \
  --output .ci/rwx-flaky-evidence/SHA/parity.json
```

Comparison preserves every original API row, date, ordering and report cell.
Only provider workspace prefixes in output paths are normalized. Both complete
retry histories remain in the parity index. Both actual requests succeeded on
their first attempt with identical complete response bytes: 79 source
observations and 12 acceptance rows. If future live API snapshots differ,
investigate the original responses before adding coverage.

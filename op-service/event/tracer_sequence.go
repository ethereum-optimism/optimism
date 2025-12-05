package event

import (
	"fmt"
	"strings"
)

type SequenceTracer struct {
	StructTracer
}

var _ Tracer = (*SequenceTracer)(nil)

func NewSequenceTracer() *SequenceTracer {
	return &SequenceTracer{}
}

func (st *SequenceTracer) Output(showDurations bool) string {
	st.l.Lock()
	defer st.l.Unlock()
	out := new(strings.Builder)
	out.WriteString(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Sequence trace</title>
</head>
<body>
Sequence:
<pre class="mermaid">
`)

	// Docs: https://mermaid.js.org/syntax/sequenceDiagram.html
	_, _ = fmt.Fprintln(out, "sequenceDiagram")
	// make sure the System is always the left-most entry in the diagram
	_, _ = fmt.Fprintln(out, "    participant System")
	// other participants are implied by the following events

	denyList := make(map[uint64]struct{})
	for _, e := range st.Entries {
		if e.Kind == TraceDeriveEnd && !e.DeriveEnd.Effect {
			denyList[e.DerivContext] = struct{}{}
		}
	}
	for _, e := range st.Entries {
		// omit entries which just passed through but did not have any effective processing
		if e.DerivContext != 0 {
			if _, ok := denyList[e.DerivContext]; ok {
				continue
			}
		}
		switch e.Kind {
		case TraceDeriveStart:
			_, _ = fmt.Fprintf(out, "    %%%% deriver-start %d\n", e.DerivContext)
			_, _ = fmt.Fprintf(out, "    System ->> %s: derive %s (%d)\n", e.Name, e.EventName, e.EmitContext)
			_, _ = fmt.Fprintf(out, "    activate %s\n", e.Name)
		case TraceDeriveEnd:
			_, _ = fmt.Fprintf(out, "    deactivate %s\n", e.Name)
			if showDurations {
				_, _ = fmt.Fprintf(out, "    Note over %s: duration: %s\n", e.Name, strings.ReplaceAll(e.DeriveEnd.Duration.String(), "µ", "#181;"))
			}
			_, _ = fmt.Fprintf(out, "    %%%% deriver-end %d\n", e.DerivContext)
		case TraceRateLimited:
			_, _ = fmt.Fprintf(out, "    Note over %s: rate-limited\n", e.Name)
		case TraceEmit:
			_, _ = fmt.Fprintf(out, "    %%%% emit originates from %d\n", e.DerivContext)
			_, _ = fmt.Fprintf(out, "    %s -->> System: emit %s (%d)\n", e.Name, e.EventName, e.EmitContext)
			_, _ = fmt.Fprintln(out, "    activate System")
			_, _ = fmt.Fprintln(out, "    deactivate System")
		}
	}

	out.WriteString(`
</pre>
	<script src="https://cdn.jsdelivr.net/npm/mermaid@10.9.1/dist/mermaid.min.js"
		integrity="sha384-WmdflGW9aGfoBdHc4rRyWzYuAjEmDwMdGdiPNacbwfGKxBW/SO6guzuQ76qjnSlr"
		crossorigin="anonymous"></script>
    <script>
      mermaid.initialize({ startOnLoad: true, maxTextSize: 10000000 });
    </script>
  </body>
</html>
`)
	return out.String()
}

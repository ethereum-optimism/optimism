package event

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// CallGraphTracer reconstructs a hierarchical call graph of events based on derivContext and emitContext.
// It logs lines like:
// p2p.ReceivedBlockEvent [09ab623c]
// └── clsync.ReceivedUnsafePayloadEvent [9277ffe6]
//     └── engine.ForkchoiceUpdateEvent [e024a814]
//
// Depth and parent-child relationships are inferred from:
// - OnDeriveStart: maps derivContext -> emitContext for the event being processed
// - OnEmit: links a new emitContext to the parent emitContext via the current derivContext
//
// Note: This focuses on visualizing the flow; it does not attempt lifecycle/cleanup beyond basic bookkeeping.
// It is safe for concurrent use.
type CallGraphTracer struct {
	log log.Logger

	mu sync.Mutex
	// Map a derivation context to the emit-context of the event being derived
	derivToEmit map[uint64]uint64
	// Nodes, keyed by emit-context
	nodes map[uint64]*cgNode
}

type cgNode struct {
	EmitContext uint64
	Emitter    string
	EventType  string
	Parent     uint64 // 0 if root
	Children   []uint64
}

var _ Tracer = (*CallGraphTracer)(nil)

func NewCallGraphTracer(l log.Logger) *CallGraphTracer {
	return &CallGraphTracer{
		log:        l,
		derivToEmit: make(map[uint64]uint64),
		nodes:       make(map[uint64]*cgNode),
	}
}

func (t *CallGraphTracer) OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time) {
	// Link the derivation context to the emit-context so we can attach children emitted during derivation
	t.mu.Lock()
	defer t.mu.Unlock()
	t.derivToEmit[derivContext] = ev.EmitContext
}

func (t *CallGraphTracer) OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	// No-op for now; we log as events are emitted to show flow in real-time
}

func (t *CallGraphTracer) OnRateLimited(name string, derivContext uint64) {
	// No-op
}

func (t *CallGraphTracer) OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	// Build or find the node for this emission
	t.mu.Lock()
	defer t.mu.Unlock()

	// Determine the Go type name of the event for readability (e.g. ForkchoiceUpdateEvent)
	evtType := typeName(ev.Event)

	node := t.nodes[ev.EmitContext]
	if node == nil {
		node = &cgNode{
			EmitContext: ev.EmitContext,
			Emitter:    name,
			EventType:  evtType,
		}
		t.nodes[ev.EmitContext] = node
	}
	// Find parent by derivation context
	if derivContext != 0 {
		if parentEmit, ok := t.derivToEmit[derivContext]; ok {
			node.Parent = parentEmit
			parent := t.nodes[parentEmit]
			if parent != nil {
				parent.Children = append(parent.Children, node.EmitContext)
			}
		}
	}

	// Log the node within the current call graph
	t.logLine(ev.EmitContext)
}

func (t *CallGraphTracer) OnAfterProcessed(evtype string) {}

func (t *CallGraphTracer) logLine(emitCtx uint64) {
	// Walk up to compute depth, then print a single line representing this node
	n, ok := t.nodes[emitCtx]
	if !ok {
		return
	}
	depth := 0
	p := n.Parent
	for p != 0 {
		depth++
		pn, ok := t.nodes[p]
		if !ok {
			break
		}
		p = pn.Parent
	}
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("    ", depth-1) + "└── "
	}
	// Use 8-hex digits for readability like the example
	ctxStr := fmt.Sprintf("[%08x]", n.EmitContext)
	msg := fmt.Sprintf("%s%s.%s %s", indent, n.Emitter, n.EventType, ctxStr)
	// Log at trace level to avoid noise at higher levels
	if t.log != nil {
		t.log.Log(log.LevelTrace, msg)
	}
}

func typeName(ev Event) string {
	if ev == nil {
		return "(nil)"
	}
	t := reflect.TypeOf(ev)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

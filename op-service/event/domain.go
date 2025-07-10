package event

import "context"

// Domain identifies an event-handling scope.
// Event-types define compile-time scope, domains define runtime scope.
// E.g. a ChainID may be a good domain to scope handling to one particular chain.
type Domain string

var _ HandlerOption = Domain("")

func (d Domain) Apply(h *Handler) {
	h.Key.Domain = d
}

func (d Domain) String() string {
	return string(d)
}

// UndefinedDomain is the default domain,
// and is a special case where the event always applies, regardless of domain.
const UndefinedDomain = Domain("")

type domainCtxKeyType struct{}

var domainCtxKey = domainCtxKeyType{}

// CtxWithDomain annotates the context to target a specific domain
func CtxWithDomain(ctx context.Context, d Domain) context.Context {
	return context.WithValue(ctx, domainCtxKey, d)
}

// DomainFromCtx retrieves the domain annotation,
// or returns UndefinedDomain if not annotated.
func DomainFromCtx(ctx context.Context) Domain {
	v := ctx.Value(domainCtxKey)
	if v == nil {
		return UndefinedDomain
	}
	return v.(Domain)
}

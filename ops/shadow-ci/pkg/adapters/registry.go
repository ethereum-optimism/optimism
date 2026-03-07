package adapters

// Registry holds all registered language adapters.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

// Register adds an adapter for a language.
func (r *Registry) Register(language string, adapter Adapter) {
	r.adapters[language] = adapter
}

// Get returns the adapter for a language.
func (r *Registry) Get(language string) (Adapter, bool) {
	a, ok := r.adapters[language]
	return a, ok
}

// All returns all registered adapters.
func (r *Registry) All() map[string]Adapter {
	return r.adapters
}

// Languages returns all registered language names.
func (r *Registry) Languages() []string {
	langs := make([]string, 0, len(r.adapters))
	for lang := range r.adapters {
		langs = append(langs, lang)
	}
	return langs
}

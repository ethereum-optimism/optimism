package dsl

func applyOpts[C any](defaultConfig C, opts ...func(config *C)) C {
	for _, opt := range opts {
		opt(&defaultConfig)
	}
	return defaultConfig
}

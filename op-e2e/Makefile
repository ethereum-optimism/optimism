DEPRECATED_TARGETS := test test-ws test-actions test-http test-cannon test-fault-proofs cannon-prestates pre-test clean fuzz

include ../justfiles/deprecated.mk

# gen-binding needs a manual shim because CONTRACT is a positional arg in just,
# not an env var. The generic shim would produce `just CONTRACT=X gen-binding`
# but just needs `just gen-binding X`.
.PHONY: gen-binding
gen-binding:
ifndef CONTRACT
	$(error CONTRACT is required: make gen-binding CONTRACT=OPContractsManager)
endif
	@echo
	@printf '%s\n' 'Deprecated make call: make gen-binding CONTRACT=$(CONTRACT)'
	@printf '%s\n' 'Consider using just instead: just gen-binding $(CONTRACT)'
	@echo
	just gen-binding $(CONTRACT)

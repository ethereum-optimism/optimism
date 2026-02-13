DEPRECATED_TARGETS := cannon cannon64-impl cannon-embeds clean elf elf-go-current sanitize-program contract test cannon-stf-verify fuzz lint

include ../justfiles/deprecated.mk

# diff-cannon needs a manual shim because VM is a positional arg in just,
# not an env var. The generic shim would produce `just VM=X diff-cannon`
# but just needs `just diff-cannon X`.
.PHONY: diff-cannon
diff-cannon:
ifndef VM
	$(error VM is required: make diff-cannon VM=multithreaded64-5)
endif
	@echo
	@printf '%s\n' 'Deprecated make call: make diff-cannon VM=$(VM)'
	@printf '%s\n' 'Consider using just instead: just diff-cannon $(VM)'
	@echo
	just diff-cannon $(VM)

# Pattern target for backwards compatibility with make diff-<vm>-cannon invocations.
# Translates make diff-<vm>-cannon to just diff-cannon <vm>.
.PHONY: diff-%-cannon
diff-%-cannon:
	@echo
	@printf '%s\n' 'Deprecated make call: make diff-$*-cannon'
	@printf '%s\n' 'Consider using just instead: just diff-cannon $*'
	@echo
	just diff-cannon $*

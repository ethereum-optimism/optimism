DEPRECATED_TARGETS := install build build-debug \
	build-x86_64-apple-darwin build-aarch64-apple-darwin \
	build-x86_64-unknown-linux-gnu build-aarch64-unknown-linux-gnu build-x86_64-pc-windows-gnu \
	build-release-tarballs \
	test-unit cov-unit cov-report-html \
	docker-build-push docker-build-push-git-sha docker-build-push-latest \
	docker-build-push-nightly docker-build-push-nightly-edge-profiling docker-build-push-nightly-profiling \
	clean profiling maxperf maxperf-no-asm \
	fmt clippy lint-typos lint-toml lint clippy-fix fix-lint \
	rustdocs cargo-test test-doc test-all pr \
	check-udeps test test-integration check-windows examples

include ../../justfiles/deprecated.mk

# Pattern rules need manual shims because the generic deprecated-target
# macro can't handle Make's build-% / build-native-% patterns.
# These forward to the parameterized just recipes.

.PHONY: build-native-%
build-native-%:
	@echo
	@printf '%s\n' '$(call banner-style,Deprecated make call: make build-native-$*)'
	@printf '%s\n' '$(call banner-style,Consider using just instead: just build-native $*)'
	@echo
	just build-native $*

.PHONY: build-%
build-%:
	@echo
	@printf '%s\n' '$(call banner-style,Deprecated make call: make build-$*)'
	@printf '%s\n' '$(call banner-style,Consider using just instead: just build-cross $*)'
	@echo
	just build-cross $*

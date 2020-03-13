TEST_IMAGE?=vault-helm-test

test-image:
	@docker build --rm -t '$(TEST_IMAGE)' -f $(CURDIR)/test/docker/Test.dockerfile $(CURDIR)

test-unit:
	@docker run -it -v ${PWD}:/helm-test vault-helm-test bats /helm-test/test/unit

test-acceptance:
	@docker run -it -v ${PWD}:/helm-test vault-helm-test bats /helm-test/test/acceptance

test-bats: test-unit test-acceptance

test: test-image test-bats


.PHONY: test-docker

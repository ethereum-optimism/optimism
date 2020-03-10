TEST_IMAGE?=consul-helm-test

test-docker:
	@docker build --rm -t '$(TEST_IMAGE)' -f $(CURDIR)/test/docker/Test.dockerfile $(CURDIR)

.PHONY: test-docker

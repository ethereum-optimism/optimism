## Long-running system tests

Those tests are meant to be *long-running*, meaning that they're not going to stop unless user input is received.

These tests are useful to simulate realistic network conditions in sysgo networks.

### How to run them?

```
    just long-running-test {OPTIONAL_TEST_FILTER} {OPTIONAL_LOG_STORAGE_PATH}
```
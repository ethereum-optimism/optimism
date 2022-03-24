# @eth-optimism/common-ts

## 0.2.4

### Patch Changes

- f981b8da: Properly exposes metrics as part of a metrics server at port 7300

## 0.2.3

### Patch Changes

- f7761058: Update log lines for service shutdown
- 5ae15042: Update metric names to include proper snake_case for strings that include "L1" or "L2"
- 5cd1e996: Have BaseServiceV2 add spaces to environment variable names

## 0.2.2

### Patch Changes

- b3f9bdef: Have BaseServiceV2 gracefully catch exit signals
- e53b5783: Introduces the new BaseServiceV2 class.

## 0.2.1

### Patch Changes

- 243f33e5: Standardize package json file format

## 0.2.0

### Minor Changes

- 81ccd6e4: `regenesis/0.5.0` release

## 0.1.6

### Patch Changes

- 6d3e1d7f: Update dependencies

## 0.1.5

### Patch Changes

- c73c3939: Update the typescript version to `4.3.5`

## 0.1.4

### Patch Changes

- 5c89c45f: Move the metric prefix string to a label #1047

## 0.1.3

### Patch Changes

- baa3b761: Improve Sentry support, initializing as needed and ensuring ERROR logs route to Sentry

## 0.1.2

### Patch Changes

- 0c16805: add metrics server to common-ts and batch submitter

## 0.1.1

### Patch Changes

- 1d40586: Removed various unused dependencies
- 575bcf6: add environment and network to dtl, move metric init to app from base-service

## 0.1.0

### Minor Changes

- 28dc442: move metrics, logger, and base-service to new common-ts package

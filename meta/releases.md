<!-- DOCTOC SKIP -->
# Release process

1. Confirm all release features are completed
2. Run tests and verify build outputs
  ```shell
  # 2.0: Verify linting
  yarn
  yarn lint
  yarn lint:links

  # 2.1: Confirm contracts pass tests and compile
  cd packages/contracts
  yarn
  yarn build
  yarn test
  cd ../..

  # 2.2: Confirm contract bindings are up-to-date
  cd op-node/contracts
  make abi
  make binding
  cd ../..

  # 2.3: Run Go tests (including end-to-end tests)
  go test -v ../..
  ```
3. Sign an annotated release tag (make sure to prefix with `v` and follow [semantic versioning](https://semver.org/))
  ```shell
  # Opens editor for annotating the tag, give it a release title
  git tag -s -a v0.1.0
  # Push the tag
  git push origin v0.1.0
  ```
4. Create a release on GitHub with the tag, provide a description of features and usage


# Contributing

## Rebasing contributions against master

PRs in this repo are merged using the [`rebase`](https://git-scm.com/docs/git-rebase) method. This keeps
the git history clean by adding the PR commits to the most recent end of the commit history. It also has
the benefit of keeping all the relevant commits for a given PR together, rather than spread throughout the
git history based on when the commits were first created.

If the changes in your PR do not conflict with any of the existing code in the project, then Github supports
automatic rebasing when the PR is accepted into the code. However, if there are conflicts (there will be
a warning on the PR that reads "This branch cannot be rebased due to conflicts"), you will need to manually
rebase the branch on master, fixing any conflicts along the way before the code can be merged.

## Testing

The Helm chart ships with both unit and acceptance tests.

The unit tests don't require any active Kubernetes cluster and complete
very quickly. These should be used for fast feedback during development.
The acceptance tests require a Kubernetes cluster with a configured `kubectl`.

### Prequisites
* [Bats](https://github.com/bats-core/bats-core)
  ```bash
  brew install bats-core
  ```
* [yq](https://pypi.org/project/yq/)
  ```bash
  brew install python-yq
  ```
* [helm](https://helm.sh)
  ```bash
  brew install kubernetes-helm
  ```

### Running The Tests
To run the unit tests:

    bats ./test/unit

To run the acceptance tests:

    bats ./test/acceptance

If the acceptance tests fail, deployed resources in the Kubernetes cluster
may not be properly cleaned up. We recommend recycling the Kubernetes cluster to
start from a clean slate.

**Note:** There is a Terraform configuration in the
[`test/terraform/`](https://github.com/hashicorp/consul-helm/tree/master/test/terraform) directory
that can be used to quickly bring up a GKE cluster and configure
`kubectl` and `helm` locally. This can be used to quickly spin up a test
cluster for acceptance tests. Unit tests _do not_ require a running Kubernetes
cluster.

### Writing Unit Tests

Changes to the Helm chart should be accompanied by appropriate unit tests.

#### Formatting

- Put tests in the test file in the same order as the variables appear in the `values.yaml`. 
- Start tests for a chart value with a header that says what is being tested, like this:
    ```
    #--------------------------------------------------------------------
    # annotations
    ```

- Name the test based on what it's testing in the following format (this will be its first line):
    ```
    @test "<section being tested>: <short description of the test case>" {
    ```

    When adding tests to an existing file, the first section will be the same as the other tests in the file.

#### Test Details

[Bats](https://github.com/bats-core/bats-core) provides a way to run commands in a shell and inspect the output in an automated way.
In all of the tests in this repo, the base command being run is [helm template](https://docs.helm.sh/helm/#helm-template) which turns the templated files into straight yaml output.
In this way, we're able to test that the various conditionals in the templates render as we would expect.

Each test defines the files that should be rendered using the `-x` flag, then it might adjust chart values by adding `--set` flags as well.
The output from this `helm template` command is then piped to [yq](https://pypi.org/project/yq/). 
`yq` allows us to pull out just the information we're interested in, either by referencing its position in the yaml file directly or giving information about it (like its length). 
The `-r` flag can be used with `yq` to return a raw string instead of a quoted one which is especially useful when looking for an exact match.

The test passes or fails based on the conditional at the end that is in square brackets, which is a comparison of our expected value and the output of  `helm template` piped to `yq`.

The `| tee /dev/stderr ` pieces direct any terminal output of the `helm template` and `yq` commands to stderr so that it doesn't interfere with `bats`.

#### Test Examples

Here are some examples of common test patterns:

- Check that a value is disabled by default

    ```
    @test "ui/Service: no type by default" {
      cd `chart_dir`
      local actual=$(helm template \
          -x templates/ui-service.yaml  \
          . | tee /dev/stderr |
          yq -r '.spec.type' | tee /dev/stderr)
      [ "${actual}" = "null" ]
    }
    ```

    In this example, nothing is changed from the default templates (no `--set` flags), then we use `yq` to retrieve the value we're checking, `.spec.type`.
    This output is then compared against our expected value (`null` in this case) in the assertion `[ "${actual}" = "null" ]`.


- Check that a template value is rendered to a specific value
    ```
    @test "ui/Service: specified type" {
      cd `chart_dir`
      local actual=$(helm template \
          -x templates/ui-service.yaml  \
          --set 'ui.service.type=LoadBalancer' \
          . | tee /dev/stderr |
          yq -r '.spec.type' | tee /dev/stderr)
      [ "${actual}" = "LoadBalancer" ]
    }
    ```

    This is very similar to the last example, except we've changed a default value with the `--set` flag and correspondingly changed the expected value.

- Check that a template value contains several values
    ```
    @test "syncCatalog/Deployment: to-k8s only" {
      cd `chart_dir`
      local actual=$(helm template \
          -x templates/sync-catalog-deployment.yaml  \
          --set 'syncCatalog.enabled=true' \
          --set 'syncCatalog.toConsul=false' \
          . | tee /dev/stderr |
          yq '.spec.template.spec.containers[0].command | any(contains("-to-consul=false"))' | tee /dev/stderr)
      [ "${actual}" = "true" ]

      local actual=$(helm template \
          -x templates/sync-catalog-deployment.yaml  \
          --set 'syncCatalog.enabled=true' \
          --set 'syncCatalog.toConsul=false' \
          . | tee /dev/stderr |
          yq '.spec.template.spec.containers[0].command | any(contains("-to-k8s"))' | tee /dev/stderr)
      [ "${actual}" = "false" ]
    }
    ```
    In this case, the same command is run twice in the same test.
    This can be used to look for several things in the same field, or to check that something is not present that shouldn't be.

    *Note:* If testing more than two conditions, it would be good to separate the `helm template` part of the command from the `yq` sections to reduce redundant work.

- Check that an entire template file is not rendered
    ```
    @test "syncCatalog/Deployment: disabled by default" {
      cd `chart_dir`
      local actual=$(helm template \
          -x templates/sync-catalog-deployment.yaml  \
          . | tee /dev/stderr |
          yq 'length > 0' | tee /dev/stderr)
      [ "${actual}" = "false" ]
    }
    ```
    Here we are check the length of the command output to see if the anything is rendered.
    This style can easily be switched to check that a file is rendered instead.

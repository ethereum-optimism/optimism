# Requirements

This is intended to run on x86-64 architecture.

## Purpose

This recipe, `kona-node-dev`, is different from the `kona-node` recipe in that
it builds a local container image of `kona-node` instead of pulling a nightly
image of `main`.  This is useful, because it allows developers to checkout a
development branch and see how it behaves on a network.

## Set up

Assuming you are on Ubuntu and your user is member of the group `docker`, first time run

    git clone 'https://github.com/op-rs/kona.git'
    cd kona/docker/recipes/kona-node-dev/
    just init

If the last step fails due to missing packages, you can run `just setup-ubuntu`
and then run `just init` again.  This will install the required packages for
Ubuntu.  `just init` will also set up a virtual network, and finally spin up
`kona-node`, `op-reth`, `prometheus` and `grafana`.

## Normal usage

For future invocation it suffices to spin the system up and down with:

    just up
    just down

You can also run `just upd` if you want to detach from the docker logs.
If you want to update the `kona` submodule, you can run `just update`.

A typical workflow after init could look like this:

    # remove existing images causing them to be rebuild
    just rmi
    # pull latest commits
    just update
    # checkout dev branch
    just checkout <my-branch>
    # build images and start containers
    just upd
    # visit Grafana
    just stop

For more info on the commands please refer to `justfile`.

## Environment

This setup uses `publicnode.com` as default L1, and the environment is configured in `publicnode.env`.
To use different RPC servers or ports, you can copy the file and make modifications.  Then run:

    just up myenv.env
    just down myenv.env

or change the default in the `justfile.

## Services and observability

The following services are provided:

    http://localhost:3000

Default credentials are `admin:admin` and you should change that if you plan to
use this instance over longer time.

## Storage

The data is stored in current directory `./datadirs`, but you can modify the
`volume` mapping in `docker-compose.yml` to use a different volume.

## Caveats

The port numbers are fixed, so it would not be possible to run more than one
instance on a machine at the same time.  Please bear this in mind when running
an instance for longer time.  You can check if ports are in use with `docker
ps`.

## Bugs and development

Everything is orchestrated from `justfile`.  Feel free to edit and submit PRs.

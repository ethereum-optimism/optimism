# Docker Builds

Guidance for writing and editing Dockerfiles in the monorepo.

The overriding rule: **every external network fetch must retry.** The package registries
and CDNs these builds depend on — `dl-cdn.alpinelinux.org`, `apk.cgr.dev`, the Debian/Ubuntu
mirrors, GitHub releases — intermittently drop connections or return transient server errors.
Unwrapped, each one is a CI flake.

Use each tool's own retry mechanism; there is no shared retry script to keep in sync.

## apt / apt-get

Add `-o Acquire::Retries=8` to every invocation (`update`, `upgrade`, `install`):

```dockerfile
RUN apt-get -o Acquire::Retries=8 update \
 && apt-get -o Acquire::Retries=8 install -y --no-install-recommends ca-certificates
```

## curl / wget

Use the built-in retry flags. They may appear before or after the URL:

```dockerfile
RUN curl -fL --retry 5 --retry-all-errors --retry-delay 2 -o out.tgz "$URL"
RUN wget --tries=5 --waitretry=5 --retry-connrefused -O out.tgz "$URL"
```

`--retry-all-errors` requires curl ≥ 7.71 (true for every base image used here). For a
`curl ... | sh` install, the retry still covers the download of the script.

## apk

apk has no built-in download retry, so wrap it in a bounded loop that **exits non-zero when
the budget is exhausted** — so a genuine breakage (e.g. a renamed package) still fails the
build instead of being masked:

```dockerfile
# ~5 min budget: up to 15 attempts, 20s apart.
RUN n=0; until apk add --no-cache ca-certificates openssl; do n=$((n+1)); [ "$n" -ge 15 ] && exit 1; echo "apk add retry $n/15 in 20s" >&2; sleep 20; done
```

Do **not** write `for i in ...; do apk add ... && break; done` — that loop exits `0` even when
every attempt failed, masking real errors.

Keep `--no-cache`: it makes each build pull the latest package index, and a BuildKit cache
mount would defeat that. apk is the registry that flakes most in CI, which is why it gets the
most generous (~5 min) budget.

## Checklist when touching a Dockerfile

- Every `apt-get`/`apt` install carries `-o Acquire::Retries=8`.
- Every `curl`/`wget` carries its retry flags.
- Every `apk add` is wrapped in the retry loop above.
- Retries wrap only network steps — never build/compile steps.

# grinderz/envdoc

Fork of [g4s8/envdoc](https://github.com/g4s8/envdoc) used to generate `envs.md`
in a set of Go services configured with cleanenv. The fork is permanent: its changes are not sent
upstream. This file is fork-only, so it never conflicts with upstream `README.md`.

## What the fork adds

On top of the upstream release tag:

- **Output written after a successful run.** `-output` is written only once
  the sources parsed and rendered; a failed run leaves the previous file as it
  was instead of an empty one.
- **Multiple dirs.** Positional arguments after the flags are parsed together
  with `-dir`, so a type from one dir resolves a field in another:
  `envdoc -dir ./internal ../shared-lib`.
- **cleanenv fixes** for `-target cleanenv`:
  - `env-description` overrides the field doc comment;
  - `env-separator` defaults to `,` for slices and maps, as in cleanenv;
  - a field with `env-prefix` and no `env` gets no env name of its own, so
    no `PREFIX_` rows appear.
- **`markdown-table` format.** One table per section with
  `env variable | type | description | options` columns. Nested structs are
  flattened into rows.
- **Enum values.** For a named int/uint/string type, the exported constants of
  that type are listed in the options column as `one of: a, b, c`. A value is
  the stringer `-linecomment` comment or the string literal. Constants named
  `_`, unexported constants of an exported type (such as a `typeCount`
  sentinel) and the `unknown` value are skipped.
- **Map format.** Map fields show `` `key:value` pairs `` in the options column.
- **`-template <file>`** renders with a template file instead of the built-in
  one of `-format`. The option strings and the helper templates
  (`item.options`) still come from `-format`. The template gets the fork-only
  item fields `.Type`, `.Values` and `.Map`.
- **Module path** `github.com/grinderz/envdoc`: no `replace` in the services,
  and `go run github.com/grinderz/envdoc@<version>` works (see
  [Module path](#module-path)).
- **Fork CI** in `.github/workflows/fork.yml`: upstream `go.yml` runs on
  `master` only.
- **Makefile** with the checks of the fork CI and the fork maintenance
  targets below (`make help` lists them).

Fork-only code lives in new files where possible: `enums/`,
`render/template_file.go`, `render/templ/markdown_table.tmpl`, `fork_test.go`,
`.github/workflows/fork.yml`, `Makefile`, `local.mk.example`.

## Branches and tags

| Ref | What it is |
|-----|------------|
| `master` | Pure mirror of `upstream/master`, fast-forward only. No fork commits. Pushed directly, not through PRs. |
| `fork/main` | Fork branch: an upstream release tag with the fork commits on top, and later upstream tags merged in. Changed only through PRs. |
| `feat/*`, `fix/*` | Fork changes, a PR into `fork/main` each. |
| `sync/vX.Y.Z` | Merge of upstream tag `vX.Y.Z`, a PR into `fork/main`. |
| `vX.Y.Z-fork.N` | Fork releases: the `N`-th release of `fork/main` on upstream `vX.Y.Z`. |
| `legacy/v1.2.3-cleanenv`, tag `legacy-v1.2.3-cleanenv` | The old fork on upstream v1.2.3 (commit `5fdc116`). Services pin it as `v0.0.0-20250716173300-5fdc11656a59`. **Do not delete** until no service uses it. |

`fork/main` follows upstream release tags, not `upstream/master`: a tag is a
released state whose `go` directive and dependencies are known, while
`master` may already raise `go` for every consumer.

Upstream tags are merged into `fork/main`, not rebased onto: a rebase rewrites
the branch, so it cannot go through a PR and needs a force push. Merges keep
`fork/main` append-only, so it can be protected against force pushes, and
every fork tag stays on the branch.

**Never push upstream tags to `origin`.** Push branches and fork tags by name,
never with `--tags`. An upstream tag such as `v1.12.0` is a stable version
newer than any `v1.12.0-fork.N` by semver, so `@latest` and renovate would
pick it, and its commit does not have the fork changes.

## Repository setup

Once, locally:

```sh
git remote add upstream https://github.com/g4s8/envdoc.git
git config rerere.enabled true      # reuse conflict resolutions between merges
git config fetch.pruneTags false    # see below
gh repo set-default grinderz/envdoc
cp local.mk.example local.mk        # newest go the services run
```

`fetch.pruneTags` (often set globally together with `fetch.prune`) deletes the
local tags the fetched remote lacks: a fetch of `origin` drops the upstream
tags, a fetch of `upstream` drops the fork tags. The fork needs both, so it is
off in this repository; the make targets fetch with it off anyway.

`gh repo set-default` matters: in a GitHub fork, `gh pr create` and the web
"Compare & pull request" button target the parent repository `g4s8/envdoc` by
default. Always check that a new PR goes to `grinderz/envdoc` and `fork/main`;
`make fork.pr` passes both explicitly.

On GitHub, `grinderz/envdoc` settings:

- **General:** default branch `fork/main`. Pull requests: allow merge commits
  and squash merging; disable rebase merging, it would copy the upstream
  commits of a `sync/*` PR onto `fork/main` one by one.
- **Actions:** enable workflows; GitHub disables them in a fresh fork.
- **Rules:** two branch rulesets for `fork/main` (the default branch):
  - require a pull request and the `fork CI` checks (`check`,
    `test-windows`), with Repository admin in the bypass list. The
    fast-forward merge (see below) is a direct push, which "require a pull
    request" blocks otherwise;
  - block force pushes and restrict deletions, with an empty bypass list.
    A bypass actor skips every rule of its ruleset, so in one shared ruleset
    the admin could force push `fork/main` too.

## Make targets

| Target | What it does |
|--------|--------------|
| `make check` | Build, `go vet` (also for windows), golangci-lint of upstream CI, `_examples`, tests, coverage tests: what the fork CI runs. |
| `make fork.status` | The upstream tag `fork/main` is on, the newest upstream tag, the fork releases and what is not released yet. |
| `make fork.mirror` | Fetch upstream with its tags, fast-forward `master` locally and on `origin`. Pushes `master` only, never tags. |
| `make fork.check.go REF=<ref>` | The `go` lines of `go.mod` at a ref; fails if `go` is newer than `FORK_GO_MAX`. |
| `make fork.sync [NEW=<tag>]` | `sync/<tag>` off `origin/fork/main`, merge of the tag (the newest by default) after `fork.check.go`, then `fork.imports`. |
| `make fork.imports` | Rewrite the module path and imports from `g4s8` to `grinderz`. Does not commit. |
| `make fork.delta` | Compare the fork delta of `origin/fork/main` on its upstream tag with the one of `HEAD` on its tag, into `artifacts/fork-delta.diff`. |
| `make fork.pr` | Push the current branch and open its PR into `grinderz/envdoc:fork/main`. |
| `make fork.ff [BRANCH=<branch>]` | Check that the branch is pushed, on top of `origin/fork/main` and its required PR checks passed, then fast-forward `fork/main` to it. |
| `make fork.release` | Tag `origin/fork/main` as the next `<upstream tag>-fork.N`, push the tag, then `fork.release.check`. |
| `make fork.release.check [TAG=<tag>]` | Resolve the tag through proxy.golang.org as a consumer would. |

The targets that push (`fork.mirror`, `fork.pr`, `fork.ff`, `fork.release`)
name every ref they push: no `--tags`, no force.

## Fork changes

```sh
git fetch origin
git switch -c feat/<topic> origin/fork/main
# ... commits ...
make check
make fork.pr
```

Merge with a fast-forward, to keep the PR commits with their SHAs, once the
checks are green:

```sh
git rebase origin/fork/main && git push --force-with-lease   # only if fork/main moved on; wait for the checks again
make fork.ff
```

GitHub has no fast-forward button (**Rebase and merge** recreates the
commits), so `fork.ff` pushes the PR head to `fork/main`: `git push origin
<head>:refs/heads/fork/main`, which git rejects unless it is a fast-forward.
GitHub marks the PR as merged once its head commit is on `fork/main`. The
push needs the bypass from [Repository setup](#repository-setup).

Or merge with **Squash and merge**: one commit per feature on `fork/main`.

Then release on the same upstream base, e.g. `v1.12.0-fork.2`
(see [Releasing](#releasing)).

## Updating to a new upstream release

```sh
make fork.mirror
make fork.status       # fork/main is on v1.12.0, the newest upstream tag is v1.13.0
make fork.sync         # or NEW=v1.13.0; stops on a go line newer than FORK_GO_MAX
```

The `go` line of the fork is the minimum Go version for its consumers. It must
not be newer than the Go toolchain the services pin: their CI runs with
`GOTOOLCHAIN=local` and will not download a newer toolchain. `FORK_GO_MAX` in
`local.mk` is that version. Do not remove the `go` line from `go.mod` to work
around it: without it the module is treated as Go 1.16 and generics stop
compiling. Lowering it in a separate commit is fine if the code and the
dependencies still build with the older toolchain.

On a conflict `fork.sync` stops. Resolve it (imports: take the upstream side),
`git commit`, then `make fork.imports`. Commit the rewritten imports if it
changed anything, then:

```sh
make check
make fork.delta        # the fork delta should look the same as before the merge
# check the output against the services, see below
make fork.pr
make fork.ff           # or Create a merge commit in the PR
```

The fast-forward keeps the `Merge upstream <tag>` commit made by `fork.sync`;
**Create a merge commit** adds a second merge on top. Either keeps the
upstream ancestry. Never squash or rebase a `sync/*` PR: that drops the
ancestry, and the next sync would conflict on the same changes again.

## Releasing

```sh
make fork.release      # v1.13.0-fork.1 after a sync, v1.12.0-fork.2 after a fork change
```

`-fork.N` starts at 1 after each sync and grows with every fork release on
the same upstream base. Right after the push the proxy may still answer
`fork.release.check` with a cached 404 for a version asked for earlier; run
`make fork.release.check` again later.

## Keeping merges cheap

- Put new code in new files. Change upstream files only where the new code
  is hooked in.
- Keep fork-only declarations outside upstream blocks, e.g.
  `OutFormatMarkdownTable` is a separate `const`, not a line in the upstream
  `const (...)`.
- Add rather than change upstream API: `Generate(dir)` is kept,
  `GenerateDirs(dirs)` is added. Upstream tests stay untouched; fork tests go
  to `fork_test.go` and new `_test.go` files.
- Upstream has no `Makefile`. If it adds one, move the fork targets to
  `fork.mk` and include it, instead of merging the two.
- Do not edit upstream `go.yml`; fork CI lives in `fork.yml`. On a sync, align
  the golangci-lint version in `fork.yml` with `go.yml`.
- Do not fix upstream lint findings (newer golangci-lint reports gofumpt in
  `utils/caseconv.go` and SA1019 in `ast/parser.go`): it only adds conflicts.
- Keep `go.mod` free of `replace` and `exclude`: `go run pkg@version` refuses
  a module with them.

## Module path

The fork's module path is `github.com/grinderz/envdoc`, so it is used without
a `replace`:

```sh
go get -tool github.com/grinderz/envdoc@v1.12.0-fork.1     # pin in go.mod
go run github.com/grinderz/envdoc@v1.12.0-fork.1 ...       # or no go.mod entry at all
```

A `replace github.com/g4s8/envdoc => github.com/grinderz/envdoc ...` does not
work with these versions: Go rejects it with `module declares its path as:
github.com/grinderz/envdoc but was required as: github.com/g4s8/envdoc`. Only
the legacy pseudo-version still declares the `g4s8` path.

The rename is the commit `chore: rename module to github.com/grinderz/envdoc`:
`go.mod` and the `.go` files. Upstream `README.md` and `docenv/README.md` keep
the `g4s8` path on purpose: they are upstream docs, and editing them would
conflict on every sync.

Every sync merge brings upstream files with the old import path. In a sync
PR, resolve import conflicts by taking the upstream side, then rewrite the
imports again and commit it on the sync branch before the PR:

```sh
grep -rl --include='*.go' github.com/g4s8/envdoc . \
  | xargs sed -i 's#github.com/g4s8/envdoc#github.com/grinderz/envdoc#g'
go build ./... && go test ./...
git commit -am "chore: rewrite imports to github.com/grinderz/envdoc after $NEW"
```

If upstream changes `go.mod` itself, keep `module github.com/grinderz/envdoc`
when resolving the conflict.

## Checking against the services

The committed `envs.md` in the services is the reference output. Before
tagging, regenerate it with a local build in one or more services and review
the diff:

```sh
make build             # in this repo: artifacts/envdoc

cd <service>
sed -Ei 's#go run github.com/(g4s8|grinderz)/envdoc(@[^ ]*)?#<path to this repo>/artifacts/envdoc#' generate.go
go generate ./...      # or the make target that runs it
git diff -- '*.md'
git checkout -- generate.go
```

The local binary takes the place of `go run`, so this works whether the
service pins envdoc in its `go.mod` or runs it with `go run ...@version`.

An empty diff, or a diff limited to what the new release is meant to change,
is the expected result.

A service still on the legacy fork calls envdoc with the old
`-tag-default`, `-tag-prefix`, `-tag-description`, `-tag-separator` and
`-tag-required` flags, which `fork/main` does not have. Switch its
`generate.go` to `-target cleanenv` first.

The services render with a copy of `render/templ/markdown_table.tmpl` kept in
their shared build files and passed with `-template`. If the template or the
item fields change, update that copy in the same release.

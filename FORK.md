# grinderz/envdoc

Fork of [g4s8/envdoc](https://github.com/g4s8/envdoc) used to generate `envs.md`
in a set of Go services configured with cleanenv. The fork is permanent: its changes are not sent
upstream. This file is fork-only, so it never conflicts with upstream `README.md`.

## What the fork adds

On top of the upstream release tag:

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
- **Fork CI** in `.github/workflows/fork.yml`: upstream `go.yml` runs on
  `master` only.

Fork-only code lives in new files where possible: `enums/`,
`render/template_file.go`, `render/templ/markdown_table.tmpl`, `fork_test.go`,
`.github/workflows/fork.yml`.

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
git config rerere.enabled true    # reuse conflict resolutions between merges
gh repo set-default grinderz/envdoc
```

`gh repo set-default` matters: in a GitHub fork, `gh pr create` and the web
"Compare & pull request" button target the parent repository `g4s8/envdoc` by
default. Always check that a new PR goes to `grinderz/envdoc` and `fork/main`.

On GitHub, `grinderz/envdoc` settings:

- **General:** default branch `fork/main`. Pull requests: allow merge commits
  (for `sync/*`) and squash merging (for `feat/*`, `fix/*`); disable rebase
  merging, it would copy the upstream commits of a `sync/*` PR onto
  `fork/main` one by one.
- **Actions:** enable workflows; GitHub disables them in a fresh fork.
- **Rules:** two branch rulesets for `fork/main` (the default branch):
  - require a pull request and the `fork CI` checks (`check`,
    `test-windows`), with Repository admin in the bypass list. The
    fast-forward merge of `feat/*` (see below) is a direct push, which
    "require a pull request" blocks otherwise;
  - block force pushes and restrict deletions, with an empty bypass list.
    A bypass actor skips every rule of its ruleset, so in one shared ruleset
    the admin could force push `fork/main` too.

## Fork changes

```sh
git fetch origin
git switch -c feat/<topic> origin/fork/main
# ... commits ...
go build ./... && go test ./...
git push -u origin feat/<topic>
gh pr create --repo grinderz/envdoc --base fork/main --fill
```

Merge with **Squash and merge**: one commit per feature on `fork/main`.

Or merge with a fast-forward, to keep the PR commits with their SHAs. GitHub
has no fast-forward button (**Rebase and merge** recreates the commits), so it
is a push of the PR branch, once its checks are green:

```sh
git fetch origin
git switch feat/<topic>
git rebase origin/fork/main             # only if fork/main moved on
git push --force-with-lease             # after a rebase; wait for the checks again
git push origin feat/<topic>:fork/main  # rejected unless it is a fast-forward
```

GitHub marks the PR as merged once its head commit is on `fork/main`. The push
needs the bypass from [Repository setup](#repository-setup). Do not use it for
`sync/*`: those are merged with a merge commit.

Then tag the next release on the same upstream base, e.g. `v1.12.0-fork.2`
(see [Releasing](#releasing)).

## Updating to a new upstream release

```sh
git fetch upstream --tags

# 1. Mirror upstream master.
git switch master
git merge --ff-only upstream/master
git push origin master

# 2. Pick the new tag and check what it requires.
OLD=v1.12.0                       # the upstream tag fork/main is on now
NEW=v1.13.0
git log --oneline $OLD..$NEW
git show $NEW:go.mod | grep -E '^(go|toolchain) '
```

The `go` line of the fork is the minimum Go version for its consumers. It must
not be newer than the Go toolchain the services pin: their CI runs
with `GOTOOLCHAIN=local` and will not download a newer toolchain. Do not
remove the `go` line from `go.mod` to work around it: without it the module is
treated as Go 1.16 and generics stop compiling. Lowering it in a separate
commit is fine if the code and the dependencies still build with the older
toolchain.

```sh
# 3. Merge the tag on a sync branch.
git fetch origin
git switch -c sync/$NEW origin/fork/main
git merge --no-ff $NEW -m "Merge upstream $NEW"
# resolve conflicts, then:
go build ./... && go test ./...
./_examples/clean.sh && ./_examples/build-examples.sh && git status --short _examples

# 4. Review the fork delta: it should look the same as before the merge.
git diff $OLD origin/fork/main > /tmp/fork-old.diff
git diff $NEW sync/$NEW        > /tmp/fork-new.diff
diff /tmp/fork-old.diff /tmp/fork-new.diff

# 5. PR.
git push -u origin sync/$NEW
gh pr create --repo grinderz/envdoc --base fork/main --title "Merge upstream $NEW" --body ""
```

Merge with **Create a merge commit**, never squash: a squash drops the upstream
ancestry, and the next sync PR would conflict on the same changes again.

Check the output against the services (see below) before releasing.

## Releasing

```sh
git fetch origin
git switch fork/main
git merge --ff-only origin/fork/main
git tag v1.13.0-fork.1
git push origin v1.13.0-fork.1
```

`-fork.N` starts at 1 after each sync and grows with every fork release on
the same upstream base.

## Keeping merges cheap

- Put new code in new files. Change upstream files only where the new code
  is hooked in.
- Keep fork-only declarations outside upstream blocks, e.g.
  `OutFormatMarkdownTable` is a separate `const`, not a line in the upstream
  `const (...)`.
- Add rather than change upstream API: `Generate(dir)` is kept,
  `GenerateDirs(dirs)` is added. Upstream tests stay untouched; fork tests go
  to `fork_test.go` and new `_test.go` files.
- Do not edit upstream `go.yml`; fork CI lives in `fork.yml`. On a sync, align
  the golangci-lint version in `fork.yml` with `go.yml`.
- Do not fix upstream lint findings (newer golangci-lint reports gofumpt in
  `utils/caseconv.go` and SA1019 in `ast/parser.go`): it only adds conflicts.
- Keep `go.mod` free of `replace` and `exclude`: `go run pkg@version` refuses
  a module with them.

## Module path

`go.mod` still declares `module github.com/g4s8/envdoc`, so the services
connect the fork with a `replace`:

```
replace github.com/g4s8/envdoc => github.com/grinderz/envdoc v1.12.0-fork.1
```

To drop the `replace` and run the fork as
`go run github.com/grinderz/envdoc@<version>`, the module path has to become
`github.com/grinderz/envdoc`: one `feat/*` PR with

```sh
go mod edit -module github.com/grinderz/envdoc
grep -rl --include='*.go' github.com/g4s8/envdoc . \
  | xargs sed -i 's#github.com/g4s8/envdoc#github.com/grinderz/envdoc#g'
```

After that, every sync merge brings upstream files with the old import path.
In a sync PR, resolve import conflicts by taking the upstream side, then run
the same `sed` again and commit it on the sync branch before the PR:

```sh
grep -rl --include='*.go' github.com/g4s8/envdoc . \
  | xargs sed -i 's#github.com/g4s8/envdoc#github.com/grinderz/envdoc#g'
go build ./... && go test ./...
git commit -am "chore: rewrite imports to github.com/grinderz/envdoc after $NEW"
```

## Checking against the services

The committed `envs.md` in the services is the reference output. Before
tagging, regenerate it with the local fork in one or more services and review
the diff:

```sh
cd <service>
go mod edit -replace github.com/g4s8/envdoc=<path to this repo>
go generate ./...   # or the make target that runs it
git diff -- '*.md'
git checkout -- go.mod go.sum
```

An empty diff, or a diff limited to what the new release is meant to change,
is the expected result.

A service still on the legacy fork calls envdoc with the old
`-tag-default`, `-tag-prefix`, `-tag-description`, `-tag-separator` and
`-tag-required` flags, which `fork/main` does not have. Switch its
`generate.go` to `-target cleanenv` first. After the module rename, replace
`github.com/grinderz/envdoc` instead: a directory replace must match the
module path declared in its `go.mod`.

The services render with a copy of `render/templ/markdown_table.tmpl` kept in
their shared build files and passed with `-template`. If the template or the
item fields change, update that copy in the same release.

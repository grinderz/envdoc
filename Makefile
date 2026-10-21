MAKEFLAGS += --warn-undefined-variables

.ONESHELL:
.SHELLFLAGS := -o errtrace -o pipefail -o noclobber -o errexit -o nounset -c
SHELL := bash

# Fork-only file, upstream g4s8/envdoc has no Makefile; FORK.md describes the
# fork targets. Machine-specific settings go to a repo-local local.mk
# (gitignored, see local.mk.example). CLI variables still win
-include local.mk

GO_MOD_ID := github.com/grinderz/envdoc

# golangci-lint: the version of upstream .github/workflows/go.yml. Its
# staticcheck fails on a go newer than the one it was built with, so a local
# run uses the go of upstream CI, not the newest installed toolchain. In ci
# setup-go already installs the right one
GO_GOLANGCI_LINT_VERSION ?= 2.4.0
GO_GOLANGCI_LINT_TIMEOUT ?= 5m
GO_TOOLCHAIN ?= go1.25.14

ifndef CI
export GOTOOLCHAIN = $(GO_TOOLCHAIN)
endif

GO_GOLANGCI_LINT_CMD ?= go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GO_GOLANGCI_LINT_VERSION)

# test
GO_GOTESTSUM_VERSION ?= 1.13.0
GO_GOTESTSUM_ARGS ?= --format testname
GO_GOTESTSUM_CMD ?= go run gotest.tools/gotestsum@v$(GO_GOTESTSUM_VERSION) $(GO_GOTESTSUM_ARGS) --
GO_TEST_CMD ?= $(GO_GOTESTSUM_CMD)

ARTIFACTS_DIR ?= artifacts
BIN ?= $(ARTIFACTS_DIR)/envdoc

# fork
FORK_REPO ?= grinderz/envdoc
FORK_REMOTE ?= origin
FORK_UPSTREAM_REMOTE ?= upstream
FORK_UPSTREAM_MOD_ID := github.com/g4s8/envdoc
FORK_BRANCH ?= fork/main
# upstream tag to merge by fork.sync, the newest one when empty
NEW ?=
# branch to fast-forward FORK_BRANCH to by fork.ff, the current one when empty
BRANCH ?=
# fork tag to check by fork.release.check, the newest one when empty
TAG ?=
# git ref fork.check.go reads go.mod from
REF ?= HEAD
# newest go the services run, fork.check.go fails on a newer go line; no
# check when empty
FORK_GO_MAX ?=

# shell commands, not $(shell): the recipes run them after their git fetch
# the upstream tag FORK_BRANCH is on: the nearest upstream tag merged into it
FORK_BASE_CMD = git describe --tags --abbrev=0 --match 'v*' --exclude '*-fork.*' $(FORK_REMOTE)/$(FORK_BRANCH)
FORK_UPSTREAM_LATEST_CMD = git tag -l 'v*' --sort=-v:refname | grep -v -e '-fork\.' | head -1
# fetch.pruneTags would drop the local tags the fetched remote lacks: the
# upstream tags on a fetch of FORK_REMOTE, the fork tags on a fetch of upstream
FORK_GIT_FETCH = git -c fetch.pruneTags=false fetch
FORK_CLEAN_CHECK = git diff --quiet && git diff --cached --quiet || { echo "the working tree has changes, commit or stash them" >&2; exit 1; }

ARGS ?=


.DEFAULT_GOAL := help


##@ General


.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9\-\\.%]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

$(ARTIFACTS_DIR):
	mkdir -p $@

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf "$(ARTIFACTS_DIR)"


##@ Build


.PHONY: build
build: $(ARTIFACTS_DIR) ## Build envdoc into artifacts/
	go build -o $(BIN) .

.PHONY: install
install: ## Install envdoc into GOBIN
	go install .

.PHONY: examples
examples: ## Regenerate _examples with the local sources
	./_examples/clean.sh >/dev/null
	./_examples/build-examples.sh


##@ Lint


.PHONY: lint
lint: lint.vet lint.vet.windows lint.golangci lint.examples ## Run all linters

.PHONY: lint.vet
lint.vet: ## Run go vet
	go vet ./...

.PHONY: lint.vet.windows
lint.vet.windows: ## Run go vet for windows, as the windows job of the fork CI builds it
	GOOS=windows go vet ./...

.PHONY: lint.golangci
lint.golangci: ## Run golangci-lint (ARGS=... for extra flags)
	$(GO_GOLANGCI_LINT_CMD) run --timeout=$(GO_GOLANGCI_LINT_TIMEOUT) --show-stats $(ARGS)

.PHONY: lint.examples
lint.examples: examples ## Check that _examples are up to date
	@if [ -n "$$(git status --porcelain -- _examples)" ]; then
		git status --short -- _examples
		echo "examples are dirty, commit the regenerated _examples" >&2
		exit 1
	fi


##@ Test


.PHONY: test
test: ## Run tests
	$(GO_TEST_CMD) -race $(ARGS) ./...

# the coverage tag builds debug_coverage.go, as in upstream CI
.PHONY: test.coverage
test.coverage: $(ARTIFACTS_DIR) ## Run tests with the coverage tag and a coverage report
	$(GO_TEST_CMD) -tags coverage -covermode=count -coverprofile=$(ARTIFACTS_DIR)/coverage.out $(ARGS) ./...
	go tool cover -func=$(ARTIFACTS_DIR)/coverage.out

.PHONY: check
check: build lint test test.coverage ## Run what the fork CI runs


##@ Fork


.PHONY: fork.status
fork.status: ## Show the upstream tag of FORK_BRANCH, the newest upstream tag and the fork releases
	@$(FORK_GIT_FETCH) --quiet $(FORK_REMOTE)
	$(FORK_GIT_FETCH) --quiet --tags $(FORK_UPSTREAM_REMOTE)
	base=$$($(FORK_BASE_CMD))
	latest=$$($(FORK_UPSTREAM_LATEST_CMD))
	echo "$(FORK_BRANCH) is on upstream $$base, the newest upstream tag is $$latest"
	echo "master is $$(git rev-list --count $(FORK_REMOTE)/master..$(FORK_UPSTREAM_REMOTE)/master) commits behind upstream/master"
	echo "releases on $$base: $$(git tag -l "$$base-fork.*" --sort=v:refname | tr '\n' ' ')"
	last=$$(git describe --tags --abbrev=0 --match 'v*-fork.*' $(FORK_REMOTE)/$(FORK_BRANCH) 2>/dev/null || :)
	if [ -n "$$last" ]; then
		echo "not released since $$last:"
		git log --oneline --first-parent "$$last..$(FORK_REMOTE)/$(FORK_BRANCH)"
	fi

# the refspecs name master only: upstream tags never reach FORK_REMOTE
.PHONY: fork.mirror
fork.mirror: ## Fetch upstream with its tags, fast-forward master locally and on FORK_REMOTE
	@$(FORK_GIT_FETCH) $(FORK_UPSTREAM_REMOTE) --tags
	git push $(FORK_REMOTE) refs/remotes/$(FORK_UPSTREAM_REMOTE)/master:refs/heads/master
	if [ "$$(git branch --show-current)" = master ]; then
		git merge --ff-only $(FORK_UPSTREAM_REMOTE)/master
	else
		$(FORK_GIT_FETCH) . refs/remotes/$(FORK_UPSTREAM_REMOTE)/master:refs/heads/master
	fi

.PHONY: fork.check.go
fork.check.go: ## Print the go lines of go.mod at REF, fail if go is newer than FORK_GO_MAX
	@git show "$(REF):go.mod" | grep -E '^(go|toolchain) ' || :
	max='$(FORK_GO_MAX)'
	if [ -z "$$max" ]; then
		exit 0
	fi
	go=$$(git show "$(REF):go.mod" | awk '$$1 == "go" { print $$2 }')
	if [ "$$(printf '%s\n' "$$go" "$$max" | sort -V | tail -1)" != "$$max" ]; then
		echo "go $$go at $(REF) is newer than FORK_GO_MAX=$$max" >&2
		exit 1
	fi

.PHONY: fork.sync
fork.sync: ## Merge upstream tag NEW (default: the newest) on a sync/NEW branch off FORK_BRANCH
	@$(FORK_CLEAN_CHECK)
	$(FORK_GIT_FETCH) $(FORK_REMOTE)
	base=$$($(FORK_BASE_CMD))
	new='$(NEW)'
	if [ -z "$$new" ]; then
		new=$$($(FORK_UPSTREAM_LATEST_CMD))
	fi
	if [ -z "$$new" ] || [ "$$new" = "$$base" ]; then
		echo "$(FORK_BRANCH) is already on the newest upstream tag $$base, run make fork.mirror first" >&2
		exit 1
	fi
	echo "$(FORK_BRANCH): $$base -> $$new"
	git log --oneline --first-parent "$$base..$$new"
	$(MAKE) --no-print-directory fork.check.go REF="$$new"
	git switch --no-track -c "sync/$$new" $(FORK_REMOTE)/$(FORK_BRANCH)
	if ! git merge --no-ff "$$new" -m "Merge upstream $$new"; then
		echo "resolve the conflicts (imports: take the upstream side), git commit, then make fork.imports" >&2
		exit 1
	fi
	$(MAKE) --no-print-directory fork.imports

# upstream files come with the upstream module path, every sync rewrites them
.PHONY: fork.imports
fork.imports: ## Rewrite the module path and the imports from upstream to the fork
	@changed=
	if [ "$$(awk '$$1 == "module" { print $$2 }' go.mod)" != '$(GO_MOD_ID)' ]; then
		go mod edit -module $(GO_MOD_ID)
		changed=go.mod
	fi
	files=$$(grep -rl --include='*.go' '$(FORK_UPSTREAM_MOD_ID)' . || :)
	if [ -n "$$files" ]; then
		sed -i 's#$(FORK_UPSTREAM_MOD_ID)#$(GO_MOD_ID)#g' $$files
		changed="$$changed $$files"
	fi
	go build ./...
	if [ -z "$$changed" ]; then
		echo "the module path and the imports are already the fork ones"
		exit 0
	fi
	echo "rewritten:" $$changed
	echo 'review, then: git commit -am "chore: rewrite imports to $(GO_MOD_ID)"'

.PHONY: fork.delta
fork.delta: $(ARTIFACTS_DIR) ## Compare the fork delta of FORK_BRANCH with the one of HEAD, after fork.sync
	@$(FORK_GIT_FETCH) --quiet $(FORK_REMOTE)
	old=$$($(FORK_BASE_CMD))
	new=$$(git describe --tags --abbrev=0 --match 'v*' --exclude '*-fork.*' HEAD)
	# the blob hashes of the index lines differ whenever upstream touched the file
	git diff "$$old" $(FORK_REMOTE)/$(FORK_BRANCH) | grep -v '^index ' >| $(ARTIFACTS_DIR)/fork-delta-old.diff
	git diff "$$new" HEAD | grep -v '^index ' >| $(ARTIFACTS_DIR)/fork-delta-new.diff
	echo "fork delta: $$old..$(FORK_REMOTE)/$(FORK_BRANCH) vs $$new..HEAD"
	if diff -u $(ARTIFACTS_DIR)/fork-delta-old.diff $(ARTIFACTS_DIR)/fork-delta-new.diff >| $(ARTIFACTS_DIR)/fork-delta.diff; then
		echo "the fork delta is the same"
	else
		echo "the fork delta changed, review $(ARTIFACTS_DIR)/fork-delta.diff"
	fi

.PHONY: fork.pr
fork.pr: ## Push the current branch and open its PR into FORK_BRANCH of FORK_REPO
	@branch=$$(git branch --show-current)
	case "$$branch" in
	"" | master | $(FORK_BRANCH))
		echo "switch to a feat/*, fix/* or sync/* branch" >&2
		exit 1
		;;
	esac
	git push -u $(FORK_REMOTE) "$$branch"
	gh pr create --repo $(FORK_REPO) --base $(FORK_BRANCH) --head "$$branch" --fill

# a fast-forward keeps the PR commits and, for sync/*, the merge of the
# upstream tag. The push needs the ruleset bypass, see FORK.md
.PHONY: fork.ff
fork.ff: ## Fast-forward FORK_BRANCH to BRANCH (default: the current one) once its PR checks pass
	@branch='$(BRANCH)'
	if [ -z "$$branch" ]; then
		branch=$$(git branch --show-current)
	fi
	$(FORK_GIT_FETCH) $(FORK_REMOTE)
	head=$$(git rev-parse "$$branch")
	if [ "$$head" != "$$(git rev-parse "$(FORK_REMOTE)/$$branch")" ]; then
		echo "$$branch differs from $(FORK_REMOTE)/$$branch, push it first" >&2
		exit 1
	fi
	if ! git merge-base --is-ancestor $(FORK_REMOTE)/$(FORK_BRANCH) "$$head"; then
		echo "$$branch is not on top of $(FORK_REMOTE)/$(FORK_BRANCH): rebase feat/* onto it, merge it into sync/*" >&2
		exit 1
	fi
	gh pr checks "$$branch" --repo $(FORK_REPO) --required
	git push $(FORK_REMOTE) "$$head:refs/heads/$(FORK_BRANCH)"

.PHONY: fork.release
fork.release: ## Tag FORK_BRANCH on FORK_REMOTE as the next <upstream tag>-fork.N and push the tag
	@$(FORK_GIT_FETCH) --tags $(FORK_REMOTE)
	commit=$$(git rev-parse $(FORK_REMOTE)/$(FORK_BRANCH))
	tagged=$$(git tag --points-at "$$commit" -l 'v*-fork.*' | head -1)
	if [ -n "$$tagged" ]; then
		echo "$(FORK_REMOTE)/$(FORK_BRANCH) is already released as $$tagged" >&2
		exit 1
	fi
	base=$$($(FORK_BASE_CMD))
	n=$$(git tag -l "$$base-fork.*" | sed 's/.*-fork\.//' | sort -n | tail -1)
	tag="$$base-fork.$$(( $${n:-0} + 1 ))"
	# -m: with tag.gpgSign the tag is annotated and needs a message
	git tag -m "$$tag: upstream $$base with the fork changes, see FORK.md" "$$tag" "$$commit"
	git push $(FORK_REMOTE) "refs/tags/$$tag"
	$(MAKE) --no-print-directory fork.release.check TAG="$$tag"

# outside the module, so the tag resolves as for any consumer: go mod download
# reads it from the proxy and verifies it against the checksum database. Both
# cache a not found answer for a while, a check right after the push may fail
.PHONY: fork.release.check
fork.release.check: ## Check that the Go proxy and checksum database serve TAG (default: the newest fork tag)
	@tag='$(TAG)'
	if [ -z "$$tag" ]; then
		tag=$$(git tag -l 'v*-fork.*' --sort=-v:refname | head -1)
	fi
	cd "$$(mktemp -d)"
	GOFLAGS= GOPROXY=https://proxy.golang.org GOPRIVATE= GONOSUMDB= GONOPROXY= GOSUMDB=sum.golang.org \
		go mod download -json "$(GO_MOD_ID)@$$tag" | grep -E '"(Version|Sum|Error)"'

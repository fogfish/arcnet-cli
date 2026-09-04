# Quickstart: Validating Graph Initialization Inside an Existing Repository

**Feature**: `031-init-existing-git-repo` | **Plan**: [plan.md](plan.md)

Runnable scenarios proving the feature end-to-end. Each maps to acceptance
scenarios in [spec.md](spec.md); contract details live in
[contracts/](contracts/) and are not repeated here.

## Prerequisites

```sh
go build -o /tmp/arc ./cmd/arc
git --version    # required in both modes
```

`GRAPH=$(mktemp -d)` style scratch dirs throughout; nothing writes into the
repository under test.

---

## S1 — Standalone init still works (FR-007, SC-008)

```sh
D=$(mktemp -d)/graph
/tmp/arc init "$D"
git -C "$D" log --oneline          # exactly one commit
git -C "$D" status --porcelain     # empty — clean tree
ls "$D/.arc/.gitignore"            # exclusion is here now…
test ! -e "$D/.gitignore"          # …and NOT at the graph root
git -C "$D" status --porcelain --ignored | grep '\.arc'   # .arc/ present but ignored
```

**Expected**: one commit; clean tree; `.arc/` excluded from tracking. The
exclusion file moved (clarification 1) — the observable outcome did not.

---

## S2 — Refusal inside an existing repository (User Story 1)

```sh
H=$(mktemp -d); git -C "$H" init -q; echo hi > "$H/README.md"
git -C "$H" -c user.email=a@b.c -c user.name=t add -A
git -C "$H" -c user.email=a@b.c -c user.name=t commit -qm init

/tmp/arc init "$H";              echo "exit=$?"   # at repo root
/tmp/arc init "$H/sub";          echo "exit=$?"   # non-existent subfolder
test ! -e "$H/sub" && echo "sub/ was NOT left behind"
git -C "$H" status --porcelain                     # unchanged
```

**Expected**: both fail with non-zero exit. The message names `$H` and names
`--skip-git-init`. No nested repository, no `sub/` left on disk (FR-005), host
repository untouched.

---

## S3 — Adding a graph to an existing project (User Story 2)

```sh
H=$(mktemp -d); git -C "$H" init -q
echo hi > "$H/README.md"; printf 'node_modules/\n' > "$H/.gitignore"
git -C "$H" -c user.email=a@b.c -c user.name=t add -A
git -C "$H" -c user.email=a@b.c -c user.name=t commit -qm init

# unrelated pending work the user has in flight
echo dirty >> "$H/README.md"
echo staged > "$H/staged.txt"; git -C "$H" add staged.txt
echo untracked > "$H/loose.txt"

/tmp/arc init --skip-git-init "$H/notes"

test ! -e "$H/notes/.git" && echo "no nested repository"
git -C "$H" show --name-only --format="" HEAD      # only notes/* paths
git -C "$H" status --porcelain                     # README modified, staged.txt staged, loose.txt untracked
git -C "$H" log --oneline | wc -l                  # 2
```

**Expected**: no nested repository; the new commit contains **only** files under
`notes/`; the user's modified, staged and untracked files are all exactly as
they were (FR-014 — this is the case a plain `git commit -m` would break); host
history grew by exactly one commit.

---

## S4 — The host project's ignore file is never touched (User Story 3)

```sh
# continuing from S3
git -C "$H" show HEAD --name-only --format="" | grep -c '^\.gitignore$'   # 0
cat "$H/.gitignore"                                                       # still just node_modules/
grep -c 'arc' "$H/.gitignore"                                             # 0
cat "$H/notes/.arc/.gitignore"                                            # *
git -C "$H" status --porcelain | grep -c 'notes/.arc'                     # 0 — excluded
```

**Expected**: host `.gitignore` byte-for-byte unchanged and absent from the
commit (FR-017); graph local state nonetheless excluded, via `.arc/.gitignore`
alone (FR-016).

---

## S5 — Layout collisions are refused, subfolder is the recovery (FR-015, FR-031)

```sh
H=$(mktemp -d); git -C "$H" init -q
echo hi > "$H/README.md"
git -C "$H" -c user.email=a@b.c -c user.name=t add -A
git -C "$H" -c user.email=a@b.c -c user.name=t commit -qm init
mkdir -p "$H/Source"; echo mine > "$H/Source/mine.txt"

/tmp/arc init --skip-git-init "$H";  echo "exit=$?"   # collides on Source/
cat "$H/Source/mine.txt"                              # untouched
/tmp/arc init --skip-git-init "$H/graph"              # recovery: subfolder
```

**Expected**: the first fails naming `Source` and pointing at the subfolder
recovery; `mine.txt` untouched; the second succeeds. An **empty** `Source/`
must fail identically (clarification 2).

---

## S6 — Flag without an enclosing repository (FR-012)

```sh
D=$(mktemp -d)/graph
/tmp/arc init --skip-git-init "$D"; echo "exit=$?"
test ! -e "$D" && echo "nothing created"
```

**Expected**: non-zero exit, message states there is no parent project. No
fallback to creating a repository, no history-less graph (clarification 3).

---

## S7 — Machine-readable contract (FR-027, FR-028)

```sh
# standalone: repository == path
D=$(mktemp -d)/g; /tmp/arc init --json "$D" | jq '{path, repository, same: (.path == .repository)}'

# nested: repository is an ancestor
/tmp/arc init --json --skip-git-init "$H/g2" | jq '{path, repository, same: (.path == .repository)}'
```

**Expected**: `repository` present and non-empty in both; `same: true`
standalone, `same: false` nested. `path`, `commit`, `foldersCreated` unchanged
in name and meaning.

---

## S8 — Target excluded by the host's ignore rules (FR-020)

```sh
H=$(mktemp -d); git -C "$H" init -q; printf 'private/\n' > "$H/.gitignore"
echo hi > "$H/README.md"
git -C "$H" -c user.email=a@b.c -c user.name=t add -A
git -C "$H" -c user.email=a@b.c -c user.name=t commit -qm init

/tmp/arc init --skip-git-init "$H/private/graph"; echo "exit=$?"
```

**Expected**: refused upfront with an explanatory message — not the confusing
late "nothing to commit" failure that would otherwise occur.

---

## S9 — Cross-command parity in a nested graph (FR-021–FR-025, research D7)

The verification-first scenarios. Run the same operations against a nested graph
and a standalone one; results must match.

```sh
# nested graph inside a host repo with unrelated history
/tmp/arc init --skip-git-init "$H/kb"
/tmp/arc apply <patch.md>     # commit contains only kb/* paths
/tmp/arc revert <...>         # only kb/* affected
/tmp/arc lint                 # history checks scoped to kb/ subtree
```

**Expected**: identical outcomes to the same sequence on a standalone graph.
Most should pass on the existing adapter (BUG-002 fixes); each failure becomes
its own narrowly-scoped fix task rather than speculative hardening.

---

## Full suite

```sh
go test ./...            # Principle VI/VIII gates
go vet ./...
```

Existing `cmd/arc/ctrl/init_test.go` assertions about a root `.gitignore` are
expected to change (research D4). SC-008 defines what must stay true: local
state excluded, working tree clean, one commit.

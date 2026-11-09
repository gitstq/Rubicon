# Contributing to Rubicon

Thanks for helping improve Rubicon. This project keeps a deliberately small
surface: **Go standard library only, no third-party dependencies, no network
calls**. Please preserve those invariants in every contribution.

## Development setup

| Requirement | Version |
| --- | --- |
| Go | 1.21 or newer |
| make | any POSIX make |
| bash / zip / tar | for `make release` |

```bash
git clone https://github.com/gitstq/Rubicon.git
cd Rubicon
make all          # gofmt + go vet + go test -race + build
make demo         # run a deterministic end-to-end demo
```

## Ground rules

1. **Zero dependencies.** Do not add a `require` line to `go.mod`. If a feature
   needs a library, implement the minimal subset in `internal/`.
2. **Offline & deterministic.** No network I/O, no wall-clock reads outside the
   injectable `Engine.Now`. Given `--now`, two runs must produce identical output.
3. **Event sourcing is the source of truth.** Never mutate a card in place;
   always append an event carrying a full snapshot.
4. **Tests are mandatory.** Add a unit test for every new rule, command and
   hash-chain behaviour. `make test` must pass with `-race`.
5. **Comments in English, docs in three languages.** Code comments are English;
   user-facing docs stay in sync across 简体中文 / 繁體中文 / English in `README.md`.

## Commit messages

We follow the [Angular / Conventional Commits](https://www.conventionalcommits.org/)
style used throughout the repository:

```
feat(cli): add handoff resume summary
fix(store): reject blank lines in ledger
test(engine): cover deadline expiry
docs: sync all three README language versions
build: cross-compile darwin/arm64
```

Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `build`, `chore`, `ci`.

## Pull request checklist

- [ ] `make all` passes locally (including `go vet` and `-race`)
- [ ] New behaviour has a unit test
- [ ] No new third-party dependency
- [ ] `README.md` updated in **all three** language versions if user-facing
- [ ] `rubicon verify` and `rubicon lint` still exit 0 on the demo workspace

## Reporting issues

Open an issue with: the exact command, `--now` value, `--format`, the ledger
(`rubicon report --format json` output is enough — please redact any private
content), and the expected vs actual result. Never paste credentials.

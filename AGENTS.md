# Repository Guidelines

## Project Structure & Module Organization
- `shardeumd/` builds the CLI (entry in `cmd/shardeumd`); shared chain logic lives across `x/`, `server/`, `rpc/`, `mempool/`, and `ethereum/`.
- Smart-contract artifacts and client tooling live in `contracts/`, `precompiles/`, `client/`, `wallets/`, and `indexer/`.
- Tests, fixtures, and automation sit in `tests/`, `testutil/`, and `scripts/`, with reference material under `docs/`, `SECURITY.md`, and `CHANGELOG.md`.

## Build, Test, and Development Commands
- `make build` compiles `shardeumd` into `build/shardeumd`; enable `COSMOS_BUILD_OPTIONS=nooptimization` when you need debuggable binaries.
- `make start-network [NODES=4]` boots a local validator set; extend it with `./scripts/add_node.sh node4` and stop it via `pkill -f 'shardeumd.*shardeum-testnet'`.
- `make lint` runs `golangci-lint`, `pylint`, and Solidity checks; `make lint-fix` applies Go formatting and import cleanups.
- `make proto-all` formats, lints, and regenerates protobuf stubs in `proto/` and `shardeumd/proto`.
- `make install` drops the CLI into `$GOPATH/bin`; confirm with `go run ./shardeumd/cmd/shardeumd version`.

## Coding Style & Naming Conventions
- `make lint` enforces `gofumpt`, `gci`, `revive`, and security checks; keep Go code tab-indented with descriptive CamelCase exports (e.g. `MsgMint`).
- Stick to lowercase package names (`x/evmkeeper`, `rpc/http`) and locate shared interfaces in files like `interfaces.go`.
- Proto files follow the repo `buf`/`protolint` config—rerun `make proto-all` after edits and commit regenerated `.pb.go` files.
- Python helpers must satisfy `.pylintrc`, while Solidity sources follow `.solhint.json`; `make lint` or `npx solhint "contracts/**/*.sol"` keep them aligned.

## Testing Guidelines
- `make test-unit` runs the Go suite (sans simulations) and `make test-shardeumd` targets the embedded module set with `-tags test`.
- `make test-unit-cover` refreshes `coverage.txt` by merging root and `shardeumd` data—mention coverage shifts in your PR notes.
- For end-to-end flows, use `make test-scripts` (`pytest`) and `make test-solidity` (`scripts/run-solidity-tests.sh`); name new Go tests `*_test.go` and share helpers in `testutil/`.

## Commit & Pull Request Guidelines
- Reference an open issue and sign every commit (`git commit -S`); unsigned pushes are rejected per `CONTRIBUTING.md`.
- Use Conventional Commit prefixes (`fix:`, `feat:`, `chore:`) as reflected in `fix: start network script can be from project home also (#10)`.
- Keep PRs reviewable: group related changes, list the validation commands you ran, and flag breaking or UX-visible updates.

## Security & Configuration Tips
- Review `SECURITY.md` before handling disclosures and keep secrets out of git—`.gitleaks.toml` supports scanning.
- Run `make vulncheck` on release branches, tidy modules with `go mod tidy`, and use `docker-compose.yml` or `shell.nix` for reproducible local chains.

# anthro-env Code Review Audit

Date: 2026-04-11

## Scope

This audit is based on the current working tree of `anthro-env`, including uncommitted changes present during review.

Review focus:

- macOS CLI behavior correctness
- SSH fallback behavior
- shell hook behavior
- release-readiness of the current branch

## Verification Performed

Commands executed during review:

```bash
go test ./...
go build ./cmd/anthro-env
```

Observed result:

- `go build ./cmd/anthro-env` passed
- `go test ./...` failed in `internal/core`

## Current Assessment

The branch includes meaningful fixes compared with the previous review:

- `edit` now triggers shell sync in the hook
- `doctor` now detects the current shell and checks the matching rc file
- MiniMax token handling support was expanded to include `ANTHROPIC_API_KEY`

However, the branch is **not ready to merge** because the current implementation and tests still contain regressions and invalid assertions.

## Findings

### 1. High Risk: SSH fallback token export is broken

Files:

- [internal/core/manager.go](/Users/kelaocai/code/anthro-env/internal/core/manager.go:66)
- [internal/core/manager.go](/Users/kelaocai/code/anthro-env/internal/core/manager.go:307)

Problem:

`buildExportSnippet()` now calls `clearStoredToken(vars)` when `token == ""`.

That breaks the documented fallback path used when Keychain cannot be read. In SSH mode, tokens may exist only in the profile file. `ExportSnippet()` is supposed to preserve that plaintext token when Keychain read fails, but the current implementation clears it before generating exports.

Impact:

- `anthro-env env`
- shell hook auto-sync
- any SSH/plaintext fallback workflow

In these cases, authentication variables may disappear entirely from the exported environment.

Why this matters:

The code comment in `ExportSnippet()` still states that plaintext token fallback should be preserved when Keychain read fails. The current behavior contradicts that design.

### 2. Medium Risk: one new test locks in the wrong behavior

File:

- [internal/core/env_test.go](/Users/kelaocai/code/anthro-env/internal/core/env_test.go:127)

Problem:

`TestBuildExportSnippetClearsOldTokenOnEmpty` asserts that when `buildExportSnippet(vars, "")` is called, no token should be exported.

That is not a safe assumption in this project. In the current design, an empty `token` argument may mean "Keychain did not return a token", not "the user deleted the token". In SSH fallback mode, the token is expected to remain available from the profile file.

Impact:

If this test is made to pass as written, it will hard-code the regression described above.

### 3. Medium Risk: MiniMax switch test does not model a MiniMax switch

File:

- [internal/core/env_test.go](/Users/kelaocai/code/anthro-env/internal/core/env_test.go:143)

Problem:

`TestBuildExportSnippetClearsAuthTokenOnMiniMaxSwitch` expects `ANTHROPIC_API_KEY`, but the test data keeps:

```text
ANTHROPIC_BASE_URL=https://api.anthropic.com
```

The production logic decides between `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` partly from `ANTHROPIC_BASE_URL`. With an Anthropic URL still present, the implementation is correct to keep exporting `ANTHROPIC_AUTH_TOKEN`.

Impact:

The test currently fails for the right reason: the fixture does not represent the scenario named by the test.

### 4. Medium Risk: hook script test checks behavior that does not exist

File:

- [internal/core/env_test.go](/Users/kelaocai/code/anthro-env/internal/core/env_test.go:197)

Problem:

`TestHookScriptContainsUnsetCommands` expects `HookScript("zsh")` to contain `unset ANTHROPIC_*` lines.

But the hook script does not embed those exports directly. Its job is only to execute:

```sh
eval "$(command anthro-env env 2>/dev/null || true)"
```

The actual `unset` and `export` lines are produced by `buildExportSnippet()`, not by `HookScript()`.

Impact:

This test will fail regardless of whether the hook implementation is correct, so it is not testing a real contract.

## What Was Fixed Well

These changes are directionally correct and should remain:

- [internal/core/hook.go](/Users/kelaocai/code/anthro-env/internal/core/hook.go:83): `edit:` now triggers `_anthro_env_sync`
- [internal/core/manager.go](/Users/kelaocai/code/anthro-env/internal/core/manager.go:361): `doctor` now checks the current shell and matching rc file instead of hardcoding zsh
- [internal/core/manager.go](/Users/kelaocai/code/anthro-env/internal/core/manager.go:37): token variable selection now recognizes MiniMax/API-key flow

## Recommended Next Actions

1. Restore the SSH/plaintext fallback contract in `ExportSnippet()` and `buildExportSnippet()`.
2. Rewrite `TestBuildExportSnippetClearsOldTokenOnEmpty` so it does not treat Keychain miss as token deletion.
3. Fix `TestBuildExportSnippetClearsAuthTokenOnMiniMaxSwitch` so its fixture actually uses a MiniMax base URL.
4. Replace `TestHookScriptContainsUnsetCommands` with a test for the real hook contract:
   the hook should invoke `anthro-env env`, and `buildExportSnippet()` should be tested separately for exported content.

## Bottom Line

The branch is closer to correct than the previous revision, but it still has one real behavior regression and multiple invalid tests. It should not be merged until the SSH fallback export path is restored and the failing tests are corrected to reflect actual project behavior.

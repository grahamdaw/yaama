# 12 - Profile Config Root and End-User Examples

## Goal
Finalize profile configuration ergonomics for end users by standardizing on `~/.config/yaama/` and shipping copy-ready profile examples with documentation.

## Scope
- configuration root normalization to `~/.config/yaama/` only
- removal of legacy migration/compatibility behavior
- end-user profile examples and setup guidance in repository docs
- clarification of runtime behavior for tmux working directory and script hook execution

## Summary of Work Completed
1. Updated profile config root to `~/.config/yaama` in profile loading and validation flows.
2. Removed legacy compatibility/migration behavior for `~/.config/yaam`.
3. Updated plan/spec/docs references to use `~/.config/yaama`.
4. Added end-user profile examples:
   - `examples/profiles/default.toml`
   - `examples/profiles/dev.toml`
5. Added end-user profile guide:
   - `examples/profiles/README.md`
6. Updated top-level `README.md` with profile setup instructions and copy commands.

## Definition of Done
- Profile discovery/loading/validation references only `~/.config/yaama`.
- No legacy `~/.config/yaam` compatibility path remains in runtime code.
- End users have copy-ready profile examples and a clear setup walkthrough.
- Docs and plans are consistent with the new config root.

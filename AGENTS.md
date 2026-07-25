# Agent Working Agreement

These instructions apply to every agent working in the Volt repository.

## Development Workflow

- Prefer small, iterative commits that each represent one coherent change.
- Prefer single-file commits when practical. Save and commit each small,
  independently understandable step instead of accumulating an entire feature
  before committing.
- Follow a test-driven development loop whenever practical:
  1. Add or update a test that demonstrates the missing behavior or bug.
  2. Run it and confirm that it fails for the expected reason.
  3. Implement the smallest change that makes it pass.
  4. Refactor while keeping the relevant tests green.
- Keep each committed state buildable and passing its relevant tests.
- Do not combine unrelated implementation, refactoring, formatting, or
  documentation changes in one commit.
- Use focused commit messages that describe the behavior changed.

## Commit Attribution

- Use the repository owner's configured Git name and email.
- Never add Codex, OpenAI, ChatGPT, or another AI system as a commit author or
  co-author.
- Never add AI-generated attribution trailers such as `Co-authored-by` or
  `Generated-by`.
- Do not rewrite published commit history unless the repository owner explicitly
  requests it.

# ADR-0001: Adopt label-gated Renovate automerge

## Status

Accepted

## Context

This repository already receives Renovate pull requests and runs branch protection checks through GitHub Actions.

Recent Renovate pull requests show two distinct intents:

- some pull requests are explicitly marked for automerge with the `renovate-automerge` label
- other pull requests remain manual and should still receive human review before merge

The repository also has supporting automation for common dependency-update follow-up work, such as updating the Nix `vendorHash` when `go.mod` or `go.sum` changes.

We want to opt this repository into the shared `renovate-automerge-plugin` without widening the current merge scope beyond pull requests that are already explicitly marked as safe to automate.

## Decision

Opt this repository into the shared Renovate automerge plugin with `.github/renovate-automerge.json`.

The initial repository policy is:

- allow only `renovate[bot]` authored pull requests
- require the `renovate-automerge` label before a pull request is eligible
- keep `excludedLabels` and `excludedBranches` empty for now
- rely on existing branch protection, required checks, and repository workflows as the source of truth for merge readiness

## Consequences

### Pros

- Preserves the existing distinction between manual Renovate updates and explicitly automergeable ones
- Lets the shared plugin merge dependency updates after repository checks pass without duplicating CI policy
- Works with the repository's existing follow-up automation for Nix `vendorHash` updates

### Cons

- Pull requests without the `renovate-automerge` label will still need manual merge handling
- The repository now depends on label hygiene in Renovate policy to express automerge intent
- Scheduling and merge credentials still need to be managed outside this repository

## Alternatives Considered

### Option 1: Automerge every Renovate pull request from `renovate[bot]`

- Summary:
  Use author-only eligibility and let the shared plugin consider every Renovate pull request.

- Reason not adopted:
  Recent repository history shows that only a subset of Renovate pull requests is intended for automerge, so widening scope would change policy rather than preserve it.

### Option 2: Keep all Renovate merges manual

- Summary:
  Continue reviewing and merging every Renovate pull request by hand.

- Reason not adopted:
  The repository already has a clear automerge signal and supporting CI automation, so keeping everything manual would preserve toil without adding much safety for low-risk updates.

### Option 3: Use native Renovate automerge instead of the shared plugin

- Summary:
  Depend on Renovate's own automerge behavior and avoid repository opt-in to the shared Codex plugin.

- Reason not adopted:
  The shared plugin provides one consistent scheduled merge path with bounded repair and stop-note behavior across repositories, which is the operating model this repository is aligning with.

## Links

- [Repository opt-in config](../../.github/renovate-automerge.json)
- [renovate-automerge-plugin](https://github.com/daaa1k/renovate-automerge-plugin)

## Notes

If the repository later wants to broaden or narrow the eligible update classes, record that policy change in a new ADR.

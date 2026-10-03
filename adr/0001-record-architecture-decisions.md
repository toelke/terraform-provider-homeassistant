# 1. Record architecture decisions

Date: 2026-10-03 · Status: Accepted

## Context

The first spec (commit `bdd93dc`, files `0x-*.md`) was written in one go by an earlier AI
session. It mixed decisions, API facts (some of them wrong), and implementation notes. It was then
reviewed in a grilling session that checked the API claims against home-assistant/core.

## Decision

- `adr/` records *why* each decision was made, in the Nygard format (Context, Decision,
  Consequences). ADRs are numbered and never rewritten; a later ADR supersedes an earlier one.
- `spec/` records *what* the provider does. It must agree with the ADRs.
- Decisions are made by the human maintainer. An agent may draft an ADR only after it has put the
  decision and its options to the human and recorded the answer.
- `GLOSSARY.md` defines the vocabulary. `tickets/` is the implementation plan.
- The original `0x-*.md` files are removed once this structure covers their content. Git keeps
  their history.

## Consequences

Each decision has one place where it is written down. Changing a decision means writing a new ADR
and updating the spec in the same change.

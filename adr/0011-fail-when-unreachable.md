# 11. Fail when Home Assistant is unreachable

Date: 2026-10-03 · Status: Accepted

## Context

The original spec suggested returning the last known state during plan when HA is down. That
hides both the outage and any drift, and no mainstream provider behaves that way.

## Decision

A connection failure during refresh, plan, or apply is an error diagnostic. The diagnostic names
the URL and suggests `-refresh=false` when the user deliberately wants to plan offline. HTTP 401
fails fast with a token hint.

## Consequences

Plans are only "green" against a reachable HA.

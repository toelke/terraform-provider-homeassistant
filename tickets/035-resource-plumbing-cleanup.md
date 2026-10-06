---
status: done
depends_on: [025, 026]
issues: [43]
adrs: [0002]
spec: []
---

# Resource plumbing cleanup

From issue #43 (findings 9, 10, 15, and 17). A refactor without behaviour change, plus the
glossary.

## Scope
- `resources.haClient`, the mirror of the data-source helper, so each `Configure` is three lines.
  `datasources/integration.go` uses the data-source `haClient`.
- `homeassistant_integration` embeds `entryModel` and uses `entryAttributes()`, `set`, and
  `unread`, which it shares with the typed integrations.
- Dead and duplicated code from finding 15: `dyntype.NewUnknown`, `Value.goValue`, and the
  `FromGoValue` wrapper go; `newHelper` returns the resource; one package doc comment for
  `resources`; `maps.Keys` and `maps.Copy` instead of hand-written loops; one attribute decode
  for the entity data sources.
- `GLOSSARY.md` defines **Config item**. (**Stored baseline** was added with ticket 031.)

## Acceptance criteria
- [x] No `Configure` in `resources` or `datasources` repeats the provider-data type assertion.
- [x] The four config-entry attributes are declared once.
- [x] `go test -short ./...` is green without test changes beyond the deleted `NewUnknown`.
- [x] No user-visible change, so no `CHANGELOG.md` entry; `docs/` regenerates unchanged.

## Notes
- `client.EntityRegistry.List` stays: the acceptance tests use it to wait for entities. Its
  comment, like `WSClient.Close`'s, says only tests use it.
- `datasources.stringSet` and the resources' `types.SetValueFrom` stay as they are: they live in
  different packages, and sharing them would need a new package for one function.
- Finding 13 (validators) is ticket 036.

## Unreleased

## v0.5.0 (Released 2026-08-26)

IMPROVEMENTS

- feat: match masks on partial element paths (`DbtrAcct/Id`, `ct:Id`) so identifiers such as `<Rpt><Id>` can be left unmasked
- rewrite pretty-print so documents keep their namespaces, prefixes, self-closing tags, attribute quoting, and CDATA wrappers
- pretty-print and mask well-formed XML inside CDATA without promoting it to real elements
- leave escaped markup in ordinary text as text
- mask by Unicode rune so multi-byte values stay valid XML; ShowWordStart keeps original spacing

## v0.4.3 (Released 2026-02-11)

BUILD

- build: set version with goreleaser

## v0.4.2 (Released 2026-02-11)

BUILD

- build: use a separate goreleaser config file

## v0.4.1 (Released 2026-02-11)

BUILD

- build: release `./cmd/xmlq`

## v0.4.0 (Released 2026-02-11)

ADDITIONS

- docs: add install and CLI usage

BUILD

- build: compile version into release
- build: set required coverage at 50%

## v0.3.0 (Released 2026-02-02)

ADDITIONS

- feat: add CLI for xmlq
- docs: add CLI help text

## v0.2.4 (Released 2025-11-18)

BUG FIXES

- xmlq: fix extra whitespace lines

## v0.2.3 (Released 2025-09-18)

BUILD

- build: fix make task names

## v0.2.2 (Released 2025-09-18)

ADDITIONS

- docs: add initial website / web UI

BUILD

- build: set up automatic web UI release

## v0.2.1 (Released 2025-04-29)

BUG FIXES

- meta: bugfixes

## v0.2.0 (Released 2024-02-27)

IMPROVEMENTS

- feat: mask and indent CDATA

## v0.1.0 (Released 2024-02-12)

Initial release, please try out xmlq and let @adamdecaf know of any bugs or issues.

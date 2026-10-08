# SDD research notes (2026-03-05)

## Sources reviewed

- ISO/IEC/IEEE 42010 standard page: https://www.iso.org/standard/74393.html
- IEEE 1016 standard page: https://standards.ieee.org/standard/1016-2009.html
- arc42 overview: https://arc42.org/overview
- C4 model: https://c4model.com/
- MADR: https://adr.github.io/madr/
- IETF RFC process (reference for proposal rigor): https://www.ietf.org/process/rfcs/
- ADR background (Fowler): https://martinfowler.com/articles/architecture-decision-records.html

## Distilled best practices

1. Document **stakeholders + concerns + views** explicitly (42010 mindset).
2. Keep architecture docs **structured and short**, but complete (arc42 principle).
3. Use **multiple views** (context/container/component) instead of one giant diagram (C4).
4. Separate **proposals** (RFC) from **accepted design** (SDD).
5. Record irreversible choices in **ADRs** with consequences and alternatives.
6. Keep design docs **traceable to implementation** (each behavior mapped to CLI/API contract).
7. Use **stable output contracts + exit codes** for automation-first tools.
8. Treat docs as code: versioned, reviewed, and updated in the same PR as behavior changes.

## How this was applied in this repo

- SDD established at `docs/sdd/0001-gdrivectl-sdd.md`.
- RFC and ADR folders + templates created.
- Documentation workflow and quality gates defined in `docs/README.md`.

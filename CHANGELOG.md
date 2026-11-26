# Changelog

All notable changes to **SchemaPulse** are documented in this file.

## [0.2.0-dev] - Unreleased

### Added
- Added rule `PG008` (`DROP INDEX` without `CONCURRENTLY` ACCESS EXCLUSIVE lock warning).
- Added rule `PG007` (`ALTER COLUMN ... SET NOT NULL` table scan lock warning).
- Added `schemapulse rules` command to list built-in PostgreSQL rules and severities.
- Added `--only` and `--skip` flags to `schemapulse check`.
- Added severity summary counts (`high`, `medium`, `low`) to analysis results and text reports.
- Added unit tests across `analyzer`, `config`, `dialect/postgres`, `migration`, `reporter`, and `rules`.
- Added `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `ROADMAP.md`, `CHANGELOG.md`, `docs/ARCHITECTURE.md`, and `.github/` templates and CI workflow.

## [0.1.0-dev] - Initial Core Implementation

### Added
- Initialized Go module (`github.com/phantom-mbx1/schemapulse`) and modular package structure.
- Implemented PostgreSQL SQL statement parser and classifier respecting quotes, comments, and line numbers.
- Implemented rules `PG001`–`PG006`, migration sequence scanner, config loader/`init`, and `text`, `json`, and `markdown` reporters.

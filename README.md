# SchemaPulse

> Fast, local-first PostgreSQL migration safety and lock-risk analyzer for CLI and CI workflows.

SchemaPulse inspects `.sql` migration files and migration directories before they reach production—flagging table locks, destructive schema changes, non-concurrent index builds, and migration sequence issues in seconds without requiring a live database or cloud account.

## Why SchemaPulse?

Bad database migrations cause downtime, table locks, blocked writes, and data loss. SchemaPulse helps catch high-risk changes early during local development and pull request checks:

- **Local-first:** Analyze SQL migration files without needing a live PostgreSQL instance or cloud SaaS agent.
- **Cross-platform standalone binary:** Written in Go for Windows, Linux, and macOS.
- **Deterministic & fast:** Runs cleanly in CLI and CI with clear exit codes (`0` = pass, `1` = threshold breach, `2` = usage/config/parse error).
- **Actionable PostgreSQL guidance:** Explains why a statement is risky and suggests safer migration patterns.
- **Multiple report formats:** Supports terminal text, JSON (`--format json`), and PR-friendly Markdown (`--format markdown`).
- **Modular architecture:** Separates migration scanning, dialect parsing, rule evaluation, policy thresholds, and output reporters.

## Built-In PostgreSQL Rules

| Rule | Default Severity | What It Detects | Safer Pattern |
| :--- | :--- | :--- | :--- |
| **PG001** | High | `CREATE INDEX` without `CONCURRENTLY` | Use `CREATE INDEX CONCURRENTLY` outside a transaction block |
| **PG002** | High | `DROP TABLE` or `DROP SCHEMA` | Verify backups/usage and deprecate in phases before dropping |
| **PG003** | Medium | `DROP COLUMN` in `ALTER TABLE` | Stop reading/writing the column in app code first, then drop |
| **PG004** | High | `ALTER COLUMN ... TYPE` | Prefer expand-and-contract / phased column migration for large tables |
| **PG005** | Medium | `ADD COLUMN ... NOT NULL` without safe `DEFAULT` | Add with a constant `DEFAULT` or backfill + constraint validation |
| **PG006** | High | `ADD FOREIGN KEY` without `NOT VALID` | Add with `NOT VALID` first, then run `VALIDATE CONSTRAINT` |
| **PG007** | Medium | `ALTER COLUMN ... SET NOT NULL` | Validate via `CHECK ... NOT VALID` / staged rollout to limit lock impact |
| **PG008** | High | `DROP INDEX` without `CONCURRENTLY` | Use `DROP INDEX CONCURRENTLY` outside a transaction block |

## Quick Start

### 1. Build from Source (Go 1.23+)

```powershell
git clone https://github.com/phantom-mbx1/schemapulse.git
cd schemapulse
go build -ldflags="-s -w" -o schemapulse.exe ./cmd/schemapulse
```

### 2. Run CLI Commands

```powershell
# Check a safe migration file or directory
.\schemapulse.exe check .\testdata\migrations\safe

# Check a risky migration without failing on threshold
.\schemapulse.exe check --fail-on none .\testdata\migrations\risky

# Inspect migration sequence ordering and filenames
.\schemapulse.exe inspect .\testdata\migrations\risky

# List all built-in PostgreSQL rules
.\schemapulse.exe rules

# Generate a starter config file
.\schemapulse.exe init
```

## Documentation

- [Architecture](./docs/ARCHITECTURE.md)
- [Contributing Guide](./CONTRIBUTING.md)
- [Code of Conduct](./CODE_OF_CONDUCT.md)
- [Security Policy](./SECURITY.md)
- [Roadmap](./ROADMAP.md)
- [Changelog](./CHANGELOG.md)

## License

Licensed under the [MIT License](./LICENSE).

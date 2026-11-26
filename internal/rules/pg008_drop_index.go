package rules

import (
	"fmt"

	"github.com/phantom-mbx1/schemapulse/internal/config"
	"github.com/phantom-mbx1/schemapulse/internal/dialect"
)

type pg008DropIndex struct{}

func NewPG008DropIndex() Rule { return &pg008DropIndex{} }

func (r *pg008DropIndex) ID() string { return "PG008" }

func (r *pg008DropIndex) Title() string {
	return "DROP INDEX without CONCURRENTLY"
}

func (r *pg008DropIndex) DefaultSeverity() config.Severity { return config.SeverityHigh }

func (r *pg008DropIndex) Evaluate(stmt dialect.Statement) []Finding {
	if stmt.Kind != dialect.KindDropIndex || stmt.Concurrent {
		return nil
	}
	target := stmt.ObjectName
	if target == "" {
		target = "target index"
	}
	return []Finding{
		{
			RuleID:      r.ID(),
			Title:       r.Title(),
			Severity:    r.DefaultSeverity(),
			File:        stmt.File,
			Line:        stmt.Line,
			Object:      target,
			Message:     fmt.Sprintf("Dropping index %q without CONCURRENTLY acquires an ACCESS EXCLUSIVE lock on its table that blocks reads and writes.", target),
			Remediation: "Use DROP INDEX CONCURRENTLY ... (and run outside a transaction block in your migration runner).",
			Statement:   stmt.RawSQL,
		},
	}
}

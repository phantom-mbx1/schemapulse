package rules_test

import (
	"testing"

	"github.com/phantom-mbx1/schemapulse/internal/dialect"
	"github.com/phantom-mbx1/schemapulse/internal/dialect/postgres"
	"github.com/phantom-mbx1/schemapulse/internal/rules"
)

func TestPG007SetNotNull(t *testing.T) {
	adapter := postgres.New()
	stmts, err := adapter.ParseStatements("0010_nn.sql", "ALTER TABLE orders ALTER COLUMN customer_id SET NOT NULL;")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(stmts) != 1 || stmts[0].Kind != dialect.KindAlterTable {
		t.Fatalf("unexpected parsed statements: %+v", stmts)
	}

	r := rules.NewPG007SetNotNull()
	findings := r.Evaluate(stmts[0])
	if len(findings) != 1 || findings[0].RuleID != "PG007" {
		t.Fatalf("expected 1 PG007 finding, got %+v", findings)
	}
}

func TestPG008DropIndex(t *testing.T) {
	adapter := postgres.New()
	r := rules.NewPG008DropIndex()

	cases := []struct {
		name       string
		sql        string
		wantObject string
		wantFinds  int
	}{
		{name: "plain drop", sql: "DROP INDEX idx_users_email;", wantObject: "idx_users_email", wantFinds: 1},
		{name: "if exists cascade", sql: "drop index if exists public.idx_users_email cascade;", wantObject: "public.idx_users_email", wantFinds: 1},
		{name: "concurrently", sql: "DROP INDEX CONCURRENTLY idx_users_email;", wantFinds: 0},
		{name: "concurrently if exists", sql: "DROP INDEX CONCURRENTLY IF EXISTS idx_users_email;", wantFinds: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := adapter.ParseStatements("0011_drop_idx.sql", tc.sql)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if len(stmts) != 1 || stmts[0].Kind != dialect.KindDropIndex {
				t.Fatalf("unexpected parsed statements: %+v", stmts)
			}

			findings := r.Evaluate(stmts[0])
			if len(findings) != tc.wantFinds {
				t.Fatalf("expected %d PG008 findings, got %+v", tc.wantFinds, findings)
			}
			if tc.wantFinds == 1 {
				f := findings[0]
				if f.RuleID != "PG008" || f.Severity != r.DefaultSeverity() || f.Object != tc.wantObject || f.Line != 1 {
					t.Fatalf("unexpected PG008 finding: %+v", f)
				}
			}
		})
	}
}

func TestPG008IgnoresOtherStatements(t *testing.T) {
	adapter := postgres.New()
	stmts, err := adapter.ParseStatements("0012_other.sql", "DROP TABLE old_sessions; CREATE INDEX idx_a ON a (b);")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	r := rules.NewPG008DropIndex()
	for _, stmt := range stmts {
		if findings := r.Evaluate(stmt); len(findings) != 0 {
			t.Fatalf("expected no PG008 findings for %q, got %+v", stmt.RawSQL, findings)
		}
	}
}

func TestDefaultRulesIncludesPG008(t *testing.T) {
	for _, r := range rules.DefaultRules() {
		if r.ID() == "PG008" {
			return
		}
	}
	t.Fatalf("expected DefaultRules to include PG008")
}

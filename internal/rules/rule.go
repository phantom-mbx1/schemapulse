package rules

import (
	"github.com/phantom-mbx1/schemapulse/internal/config"
	"github.com/phantom-mbx1/schemapulse/internal/dialect"
)

type Finding struct {
	RuleID      string          `json:"ruleId"`
	Title       string          `json:"title"`
	Severity    config.Severity `json:"severity"`
	File        string          `json:"file"`
	Line        int             `json:"line"`
	Object      string          `json:"object,omitempty"`
	Message     string          `json:"message"`
	Remediation string          `json:"remediation"`
	Statement   string          `json:"statement"`
}

type Rule interface {
	ID() string
	Title() string
	DefaultSeverity() config.Severity
	Evaluate(stmt dialect.Statement) []Finding
}

func DefaultRules() []Rule {
	return []Rule{
		NewPG001ConcurrentIndex(),
		NewPG002DropTable(),
		NewPG003DropColumn(),
		NewPG004AlterColumnType(),
		NewPG005NotNullColumn(),
		NewPG006NotValidForeignKey(),
		NewPG007SetNotNull(),
		NewPG008DropIndex(),
	}
}

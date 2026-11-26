package dialect

type StatementKind string

const (
	KindOther              StatementKind = "OTHER"
	KindCreateIndex        StatementKind = "CREATE_INDEX"
	KindDropIndex          StatementKind = "DROP_INDEX"
	KindDropTable          StatementKind = "DROP_TABLE"
	KindDropSchema         StatementKind = "DROP_SCHEMA"
	KindAlterTable         StatementKind = "ALTER_TABLE"
	KindValidateConstraint StatementKind = "VALIDATE_CONSTRAINT"
)

type Statement struct {
	File        string
	Line        int
	RawSQL      string
	Normalized  string
	Kind        StatementKind
	ObjectName  string
	Concurrent  bool
	HasNotValid bool
	HasDefault  bool
	HasNotNull  bool
}

type Adapter interface {
	Name() string
	ParseStatements(filePath string, sqlContent string) ([]Statement, error)
}

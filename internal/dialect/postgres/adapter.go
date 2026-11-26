package postgres

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/phantom-mbx1/schemapulse/internal/dialect"
)

type Adapter struct{}

func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Name() string {
	return "postgres"
}

var (
	whitespaceRe    = regexp.MustCompile(`\s+`)
	createIndexRe   = regexp.MustCompile(`(?i)^CREATE\s+(UNIQUE\s+)?INDEX\s+(CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z0-9_."]+)\s+ON\s+(?:ONLY\s+)?([A-Za-z0-9_."]+)`)
	dropIndexRe     = regexp.MustCompile(`(?i)^DROP\s+INDEX\s+(CONCURRENTLY\s+)?(?:IF\s+EXISTS\s+)?(.+?)(?:\s+(?:CASCADE|RESTRICT))?\s*;?$`)
	dropTableRe     = regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(.+)`)
	dropSchemaRe    = regexp.MustCompile(`(?i)^DROP\s+SCHEMA\s+(?:IF\s+EXISTS\s+)?(.+)`)
	alterTableRe    = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?([A-Za-z0-9_."]+)\s+(.+)`)
	validateConsRe  = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?([A-Za-z0-9_."]+)\s+VALIDATE\s+CONSTRAINT\s+([A-Za-z0-9_."]+)`)
	addNotNullColRe = regexp.MustCompile(`(?i)\bADD\s+(?:COLUMN\s+)?(?:IF\s+NOT\s+EXISTS\s+)?[A-Za-z0-9_."]+\s+[^,;]+\bNOT\s+NULL\b`)
)

func (a *Adapter) ParseStatements(filePath string, sqlContent string) ([]dialect.Statement, error) {
	var statements []dialect.Statement

	var buf strings.Builder
	stmtStartLine := 1
	line := 1
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false
	dollarTag := ""

	flush := func(endLine int) error {
		trimmed := strings.TrimSpace(buf.String())
		buf.Reset()
		if trimmed == "" {
			return nil
		}
		if inSingleQuote || inDoubleQuote || dollarTag != "" {
			return fmt.Errorf("%s:%d: unterminated string or dollar-quoted literal in SQL", filePath, stmtStartLine)
		}
		if inBlockComment {
			return fmt.Errorf("%s:%d: unterminated block comment /* ... */ in SQL", filePath, stmtStartLine)
		}

		norm := normalizeSQL(trimmed)
		stmt := classifyStatement(filePath, stmtStartLine, trimmed, norm)
		statements = append(statements, stmt)
		stmtStartLine = endLine
		return nil
	}

	n := len(sqlContent)
	for i := 0; i < n; i++ {
		ch := sqlContent[i]
		var next byte
		if i+1 < n {
			next = sqlContent[i+1]
		}

		if strings.TrimSpace(buf.String()) == "" && !inSingleQuote && !inDoubleQuote && !inLineComment && !inBlockComment && dollarTag == "" {
			if !unicode.IsSpace(rune(ch)) {
				stmtStartLine = line
			}
		}

		if inLineComment {
			buf.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
				line++
			}
			continue
		}

		if inBlockComment {
			buf.WriteByte(ch)
			if ch == '\n' {
				line++
			} else if ch == '*' && next == '/' {
				buf.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}

		if dollarTag != "" {
			buf.WriteByte(ch)
			if ch == '\n' {
				line++
			}
			if ch == '$' && strings.HasPrefix(sqlContent[i:], dollarTag) {
				buf.WriteString(sqlContent[i+1 : i+len(dollarTag)])
				i += len(dollarTag) - 1
				dollarTag = ""
			}
			continue
		}

		if inSingleQuote {
			buf.WriteByte(ch)
			if ch == '\n' {
				line++
			} else if ch == '\'' && next == '\'' {
				buf.WriteByte(next)
				i++
			} else if ch == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			buf.WriteByte(ch)
			if ch == '\n' {
				line++
			} else if ch == '"' && next == '"' {
				buf.WriteByte(next)
				i++
			} else if ch == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if ch == '-' && next == '-' {
			inLineComment = true
			buf.WriteByte(ch)
			buf.WriteByte(next)
			i++
			continue
		}

		if ch == '/' && next == '*' {
			inBlockComment = true
			buf.WriteByte(ch)
			buf.WriteByte(next)
			i++
			continue
		}

		if ch == '\'' {
			inSingleQuote = true
			buf.WriteByte(ch)
			continue
		}

		if ch == '"' {
			inDoubleQuote = true
			buf.WriteByte(ch)
			continue
		}

		if ch == '$' {
			tag := matchDollarTag(sqlContent[i:])
			if tag != "" {
				dollarTag = tag
				buf.WriteString(tag)
				i += len(tag) - 1
				continue
			}
		}

		if ch == '\n' {
			line++
			buf.WriteByte(ch)
			continue
		}

		if ch == ';' {
			nextLine := line
			if err := flush(nextLine); err != nil {
				return nil, err
			}
			continue
		}

		buf.WriteByte(ch)
	}

	if err := flush(line); err != nil {
		return nil, err
	}

	return statements, nil
}

func matchDollarTag(s string) string {
	if !strings.HasPrefix(s, "$") {
		return ""
	}
	if strings.HasPrefix(s, "$$") {
		return "$$"
	}
	for i := 1; i < len(s) && i < 64; i++ {
		c := s[i]
		if c == '$' {
			if i == 1 {
				return ""
			}
			return s[:i+1]
		}
		if !(unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || c == '_') {
			return ""
		}
	}
	return ""
}

func normalizeSQL(sql string) string {
	var out strings.Builder
	n := len(sql)
	inLine := false
	inBlock := false
	inSingle := false
	inDouble := false

	for i := 0; i < n; i++ {
		ch := sql[i]
		var next byte
		if i+1 < n {
			next = sql[i+1]
		}
		if inLine {
			if ch == '\n' {
				inLine = false
				out.WriteByte(' ')
			}
			continue
		}
		if inBlock {
			if ch == '*' && next == '/' {
				inBlock = false
				i++
				out.WriteByte(' ')
			}
			continue
		}
		if inSingle {
			out.WriteByte(ch)
			if ch == '\'' && next == '\'' {
				out.WriteByte(next)
				i++
			} else if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			out.WriteByte(ch)
			if ch == '"' && next == '"' {
				out.WriteByte(next)
				i++
			} else if ch == '"' {
				inDouble = false
			}
			continue
		}
		if ch == '-' && next == '-' {
			inLine = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			inBlock = true
			i++
			continue
		}
		if ch == '\'' {
			inSingle = true
			out.WriteByte(ch)
			continue
		}
		if ch == '"' {
			inDouble = true
			out.WriteByte(ch)
			continue
		}
		out.WriteByte(ch)
	}

	return whitespaceRe.ReplaceAllString(strings.TrimSpace(out.String()), " ")
}

func classifyStatement(filePath string, line int, rawSQL string, norm string) dialect.Statement {
	stmt := dialect.Statement{
		File:       filePath,
		Line:       line,
		RawSQL:     rawSQL,
		Normalized: norm,
		Kind:       dialect.KindOther,
	}

	upper := strings.ToUpper(norm)

	if m := createIndexRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindCreateIndex
		stmt.Concurrent = strings.TrimSpace(m[2]) != ""
		stmt.ObjectName = strings.Trim(m[4], "\"")
		return stmt
	}

	if m := dropIndexRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindDropIndex
		stmt.Concurrent = strings.TrimSpace(m[1]) != ""
		stmt.ObjectName = strings.Trim(strings.TrimSpace(m[2]), "\"")
		return stmt
	}

	if m := dropTableRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindDropTable
		stmt.ObjectName = strings.TrimSpace(m[1])
		return stmt
	}

	if m := dropSchemaRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindDropSchema
		stmt.ObjectName = strings.TrimSpace(m[1])
		return stmt
	}

	if m := validateConsRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindValidateConstraint
		stmt.ObjectName = strings.Trim(m[1], "\"")
		return stmt
	}

	if m := alterTableRe.FindStringSubmatch(norm); len(m) > 0 {
		stmt.Kind = dialect.KindAlterTable
		stmt.ObjectName = strings.Trim(m[1], "\"")
		subAction := m[2]
		stmt.HasNotValid = strings.Contains(strings.ToUpper(subAction), "NOT VALID")
		stmt.HasDefault = strings.Contains(strings.ToUpper(subAction), " DEFAULT ")
		stmt.HasNotNull = addNotNullColRe.MatchString(subAction)
		return stmt
	}

	stmt.HasNotValid = strings.Contains(upper, "NOT VALID")
	stmt.HasDefault = strings.Contains(upper, " DEFAULT ")
	stmt.HasNotNull = addNotNullColRe.MatchString(norm)

	return stmt
}

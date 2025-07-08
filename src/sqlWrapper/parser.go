package sqlWrapper

import (
	"io"
	"strings"

	"github.com/rqlite/sql"
)

type ParsedQuery struct {
	Returning bool
	Query     string
}

func Parse(query string) (bool, []ParsedQuery, error) {
	write := false
	var parsedQueries []ParsedQuery
	p := sql.NewParser(strings.NewReader(query))
	for {
		stmt, err := p.ParseStatement()
		if err == io.EOF {
			break
		}
		if err != nil {
			return write, parsedQueries, err
		}

		switch stmt.(type) {
		case *sql.InsertStatement, *sql.UpdateStatement, *sql.DeleteStatement:
			parsedQueries = append(parsedQueries, ParsedQuery{true, stmt.String() + " RETURNING *"})
			write = true
		case *sql.SelectStatement:
			parsedQueries = append(parsedQueries, ParsedQuery{true, stmt.String()})
		default:
			parsedQueries = append(parsedQueries, ParsedQuery{false, stmt.String()})
			write = true
		}
	}

	return write, parsedQueries, nil
}

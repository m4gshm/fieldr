package squirrel

import (
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/m4gshm/gollections/collection/immutable/set"
	"github.com/m4gshm/gollections/slice"
)

var (
	placeholder = sq.Dollar
)

func sqlSelect[T ~string](table string, columns []T) sq.SelectBuilder {
	return sq.Select(slice.BehaveAsStrings(columns)...).PlaceholderFormat(placeholder).From(table)
}

func sqlSelectWhere[T ~string](table string, columns []T, condition sq.Eq) sq.SelectBuilder {
	return sqlSelect(table, columns).Where(condition)
}

func sqlDeleteWhere(table string, condition sq.Eq) sq.DeleteBuilder {
	return sq.Delete(table).Where(condition).PlaceholderFormat(placeholder)
}

func sqlUpsert[T ~string](table string, pkColumns []T, columns []T, values []any) sq.Sqlizer {
	insert := sq.Insert(table).Columns(slice.BehaveAsStrings(columns)...).Values(values...)

	update := sq.Update(" ")

	pks := set.Of(pkColumns...)
	for i, column := range columns {
		if pks.Contains(column) {
			continue
		}
		update = update.Set(string(columns[i]), values[i])
	}

	conflictSql := strings.Builder{}
	conflictSql.WriteString("ON CONFLICT (")
	for i, pkColumn := range pkColumns {
		if i > 0 {
			conflictSql.WriteString(",")
		}
		conflictSql.WriteString(string(pkColumn))
	}
	conflictSql.WriteString(") DO")

	return newPlaceholderWrapper(insert.Suffix(conflictSql.String()).SuffixExpr(update), placeholder)
}

func newPlaceholderWrapper(builder sq.Sqlizer, placeholder sq.PlaceholderFormat) *PlaceholderWrapper {
	return &PlaceholderWrapper{Sqlizer: builder, placeholder: placeholder}
}

type PlaceholderWrapper struct {
	sq.Sqlizer
	placeholder sq.PlaceholderFormat
}

func (p *PlaceholderWrapper) ToSql() (string, []interface{}, error) {
	sql, i, err := p.Sqlizer.ToSql()
	if err == nil {
		sql, err = p.placeholder.ReplacePlaceholders(sql)
	}

	return sql, i, err
}

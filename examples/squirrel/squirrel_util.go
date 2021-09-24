package squirrel

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

var (
	placeholder = sq.Dollar
)

func sqlSelect(table string, columns []string) sq.SelectBuilder {
	return sq.Select(columns...).PlaceholderFormat(placeholder).From(table)
}

func sqlSelectWhere(table string, columns []string, condition sq.Eq) sq.SelectBuilder {
	return sqlSelect(table, columns).Where(condition)
}

func sqlDeleteWhere(table string, condition sq.Eq) sq.DeleteBuilder {
	return sq.Delete(table).Where(condition).PlaceholderFormat(placeholder)
}

func sqlUpsert(table string, pkColumn string, columns []string, values []interface{}) sq.Sqlizer {
	insert := sq.Insert(table).Columns(columns...).Values(values...)

	update := sq.Update(" ")
	for i, column := range columns {
		if column == pkColumn {
			continue
		}
		update = update.Set(columns[i], values[i])
	}

	conflictSql := fmt.Sprintf("DO ON CONFLICT (%s)", pkColumn)

	return newPlaceholderWrapper(insert.Suffix(conflictSql).SuffixExpr(update), placeholder)
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

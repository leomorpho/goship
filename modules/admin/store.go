package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

type trashTableSummary struct {
	Table string
	Count int64
}

func List(ctx context.Context, db *sql.DB, res AdminResource, page, perPage int) ([]AdminRow, int, error) {
	if db == nil {
		return nil, 0, errors.New("db is nil")
	}
	table, err := safeIdentifier(res.TableName)
	if err != nil {
		return nil, 0, err
	}
	idField := res.IDField
	if idField == "" {
		idField = "id"
	}
	idField, err = safeIdentifier(idField)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	var total int
	if err := db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := fmt.Sprintf("SELECT * FROM %s ORDER BY %s DESC LIMIT ? OFFSET ?", table, idField)
	rows, err := db.QueryContext(ctx, bindQuery(db, listQuery), perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out, err := scanRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func Get(ctx context.Context, db *sql.DB, res AdminResource, id any) (AdminRow, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	table, err := safeIdentifier(res.TableName)
	if err != nil {
		return nil, err
	}
	idField := res.IDField
	if idField == "" {
		idField = "id"
	}
	idField, err = safeIdentifier(idField)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ? LIMIT 1", table, idField)
	rows, err := db.QueryContext(ctx, bindQuery(db, query), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, sql.ErrNoRows
	}
	return all[0], nil
}

func Create(ctx context.Context, db *sql.DB, res AdminResource, values map[string]any) error {
	if db == nil {
		return errors.New("db is nil")
	}
	table, err := safeIdentifier(res.TableName)
	if err != nil {
		return err
	}
	columns, args, err := sortedColumnsAndArgs(values)
	if err != nil {
		return err
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), placeholders)
	_, err = db.ExecContext(ctx, bindQuery(db, query), args...)
	return err
}

func Update(ctx context.Context, db *sql.DB, res AdminResource, id any, values map[string]any) error {
	if db == nil {
		return errors.New("db is nil")
	}
	table, err := safeIdentifier(res.TableName)
	if err != nil {
		return err
	}
	idField := res.IDField
	if idField == "" {
		idField = "id"
	}
	idField, err = safeIdentifier(idField)
	if err != nil {
		return err
	}
	columns, args, err := sortedColumnsAndArgs(values)
	if err != nil {
		return err
	}

	setParts := make([]string, 0, len(columns))
	for _, col := range columns {
		setParts = append(setParts, fmt.Sprintf("%s = ?", col))
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, strings.Join(setParts, ", "), idField)
	_, err = db.ExecContext(ctx, bindQuery(db, query), args...)
	return err
}

func Delete(ctx context.Context, db *sql.DB, res AdminResource, id any) error {
	if db == nil {
		return errors.New("db is nil")
	}
	table, err := safeIdentifier(res.TableName)
	if err != nil {
		return err
	}
	idField := res.IDField
	if idField == "" {
		idField = "id"
	}
	idField, err = safeIdentifier(idField)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", table, idField)
	_, err = db.ExecContext(ctx, bindQuery(db, query), id)
	return err
}

func sortedColumnsAndArgs(values map[string]any) ([]string, []any, error) {
	if len(values) == 0 {
		return nil, nil, errors.New("values cannot be empty")
	}

	columns := make([]string, 0, len(values))
	for key := range values {
		col, err := safeIdentifier(key)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, col)
	}
	sort.Strings(columns)

	args := make([]any, 0, len(columns))
	for _, col := range columns {
		args = append(args, values[col])
	}
	return columns, args, nil
}

func bindQuery(db *sql.DB, query string) string {
	if !isPostgresDriver(db) || strings.Count(query, "?") == 0 {
		return query
	}

	var b strings.Builder
	index := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			b.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func isPostgresDriver(db *sql.DB) bool {
	if db == nil {
		return false
	}
	driverType := strings.ToLower(reflect.TypeOf(db.Driver()).String())
	return strings.Contains(driverType, "pgx") || strings.Contains(driverType, "postgres")
}

func safeIdentifier(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("identifier is empty")
	}
	for _, r := range v {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "", fmt.Errorf("invalid identifier: %q", v)
	}
	return v, nil
}

func scanRows(rows *sql.Rows) ([]AdminRow, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := make([]AdminRow, 0)
	for rows.Next() {
		values := make([]any, len(cols))
		dest := make([]any, len(cols))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		row := AdminRow{}
		for i, col := range cols {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			default:
				row[col] = v
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func listSoftDeleteTableSummaries(ctx context.Context, db *sql.DB) ([]trashTableSummary, error) {
	if db == nil {
		return nil, nil
	}

	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, nil
	}

	tableNames := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			_ = rows.Close()
			return nil, err
		}
		tableNames = append(tableNames, table)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	summaries := make([]trashTableSummary, 0)
	for _, table := range tableNames {
		query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s" WHERE deleted_at IS NOT NULL`, strings.ReplaceAll(table, `"`, `""`))
		var count int64
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			continue
		}
		if count > 0 {
			summaries = append(summaries, trashTableSummary{Table: table, Count: count})
		}
	}

	return summaries, nil
}

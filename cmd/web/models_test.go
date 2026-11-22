package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

func newMockDB(t *testing.T) *sql.DB {
	t.Helper()
	db := sql.OpenDB(mockConnector{})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	return db
}

const mockUserEmail = "rider@example.com"
const mockUserPassword = "test-password"

type mockConnector struct{}

func (mockConnector) Connect(context.Context) (driver.Conn, error) { return mockConnection{}, nil }
func (mockConnector) Driver() driver.Driver                        { return mockDriver{} }

type mockDriver struct{}

func (mockDriver) Open(string) (driver.Conn, error) { return mockConnection{}, nil }

type mockConnection struct{}

func (mockConnection) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("unexpected prepared statement")
}
func (mockConnection) Close() error              { return nil }
func (mockConnection) Begin() (driver.Tx, error) { return nil, fmt.Errorf("unexpected transaction") }

func (mockConnection) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	ride := []driver.Value{
		int64(1), "Lakeside loop", "Meet at the north entrance.",
		time.Date(2024, 3, 17, 10, 15, 0, 0, time.UTC), time.Date(2099, 3, 17, 10, 15, 0, 0, time.UTC),
	}
	rideColumns := []string{"id", "title", "description", "created", "departure"}
	if strings.Contains(query, "FROM rides WHERE departure > UTC_TIMESTAMP() ORDER BY created DESC, id DESC LIMIT 10") && len(args) == 0 {
		return &mockRows{columns: rideColumns, values: [][]driver.Value{ride}}, nil
	}
	if strings.Contains(query, "FROM rides WHERE departure > UTC_TIMESTAMP() AND id = ?") && len(args) == 1 {
		rows := &mockRows{columns: rideColumns}
		if args[0].Value == int64(1) {
			rows.values = [][]driver.Value{ride}
		}
		return rows, nil
	}
	if strings.Contains(query, "SELECT id, hashed_password FROM users WHERE email = ?") && len(args) == 1 && args[0].Value == mockUserEmail {
		hash, err := bcrypt.GenerateFromPassword([]byte(mockUserPassword), bcrypt.MinCost)
		if err != nil {
			return nil, err
		}
		return &mockRows{
			columns: []string{"id", "hashed_password"},
			values:  [][]driver.Value{{int64(1), hash}},
		}, nil
	}
	if strings.Contains(query, "SELECT EXISTS(SELECT true FROM users WHERE id = ?)") && len(args) == 1 {
		return &mockRows{
			columns: []string{"exists"},
			values:  [][]driver.Value{{args[0].Value == int64(1)}},
		}, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", query)
}

func (mockConnection) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if strings.Contains(query, "INSERT INTO users (name, email, hashed_password, created)") && len(args) == 3 {
		if args[1].Value == "dupe@example.com" {
			return nil, &mysql.MySQLError{Number: 1062, Message: "Duplicate entry for key 'users_uc_email'"}
		}
		return driver.RowsAffected(1), nil
	}
	if strings.Contains(query, "INSERT INTO rides (title, description, created, departure)") && len(args) == 3 {
		if args[0].Value == "Weekend loop" && args[1].Value == "Meet by the bridge." {
			departure, ok := args[2].Value.(time.Time)
			if ok && departure.Location() == time.UTC && departure.After(time.Now()) {
				return mockRideInsertResult{}, nil
			}
		}
	}
	return nil, fmt.Errorf("unexpected statement: %s", query)
}

type mockRideInsertResult struct{}

func (mockRideInsertResult) LastInsertId() (int64, error) { return 3, nil }
func (mockRideInsertResult) RowsAffected() (int64, error) { return 1, nil }

type mockRows struct {
	columns []string
	values  [][]driver.Value
}

func (r *mockRows) Columns() []string { return r.columns }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if len(r.values) == 0 {
		return io.EOF
	}
	copy(dest, r.values[0])
	r.values = r.values[1:]
	return nil
}

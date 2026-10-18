package mysql

import (
	"context"
	"crypto/md5"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"webapp/models"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

// passwordStore exercises the real DAO functions without a running database.
type passwordStore struct {
	user              models.User
	missing           bool
	execCount         int
	queryErr, execErr error
	rowsErr           error
	conflictPassword  *string
}

func usePasswordStore(t *testing.T, store *passwordStore) {
	t.Helper()
	previous := db
	db = sqlx.NewDb(sql.OpenDB(passwordConnector{store}), "mysql")
	testDB := db
	t.Cleanup(func() { db = previous; _ = testDB.Close() })
}

func useLegacySecret(t *testing.T, secret string) {
	t.Helper()
	previous := viper.Get("mysql.secret")
	viper.Set("mysql.secret", secret)
	t.Cleanup(func() { viper.Set("mysql.secret", previous) })
}

type passwordConnector struct{ store *passwordStore }

func (c passwordConnector) Connect(context.Context) (driver.Conn, error) {
	return passwordConn{c.store}, nil
}
func (c passwordConnector) Driver() driver.Driver { return passwordDriver{} }

type passwordDriver struct{}

func (passwordDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("use the test connector")
}

type passwordConn struct{ store *passwordStore }

func (passwordConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (passwordConn) Begin() (driver.Tx, error)           { return nil, driver.ErrSkip }
func (passwordConn) Close() error                        { return nil }

func (c passwordConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	s := c.store
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	switch query {
	case "select user_id,username,password from user where username=?":
		rows := &passwordRows{columns: []string{"user_id", "username", "password"}}
		if !s.missing && args[0].Value == s.user.Username {
			rows.values = []driver.Value{s.user.UserID, s.user.Username, s.user.Password}
		}
		return rows, nil
	case "select password from user where user_id=?":
		rows := &passwordRows{columns: []string{"password"}}
		if !s.missing && args[0].Value == s.user.UserID {
			rows.values = []driver.Value{s.user.Password}
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
}

func (c passwordConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	s := c.store
	s.execCount++
	if s.execErr != nil {
		return nil, s.execErr
	}
	switch strings.TrimSpace(query) {
	case "insert into user(user_id,username,password) values(?,?,?)":
		s.user = models.User{UserID: args[0].Value.(int64), Username: args[1].Value.(string), Password: args[2].Value.(string)}
	case "update user set password=? where user_id=? and password=?":
		if s.conflictPassword != nil {
			s.user.Password = *s.conflictPassword
			return passwordResult{changed: 0, err: s.rowsErr}, nil
		}
		if args[1].Value != s.user.UserID || args[2].Value != s.user.Password {
			return passwordResult{changed: 0, err: s.rowsErr}, nil
		}
		s.user.Password = args[0].Value.(string)
	default:
		return nil, fmt.Errorf("unexpected exec: %s", query)
	}
	return passwordResult{changed: 1, err: s.rowsErr}, nil
}

type passwordResult struct {
	changed int64
	err     error
}

func (passwordResult) LastInsertId() (int64, error)   { return 0, nil }
func (r passwordResult) RowsAffected() (int64, error) { return r.changed, r.err }

type passwordRows struct {
	columns []string
	values  []driver.Value
}

func (r *passwordRows) Columns() []string { return r.columns }
func (*passwordRows) Close() error        { return nil }
func (r *passwordRows) Next(dest []driver.Value) error {
	if r.values == nil {
		return io.EOF
	}
	copy(dest, r.values)
	r.values = nil
	return nil
}

func bcryptPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

func legacyPassword(password, secret string) string {
	digest := md5.Sum([]byte(secret))
	return hex.EncodeToString(append([]byte(password), digest[:]...))
}

func TestInsertUserStoresSaltedBcrypt(t *testing.T) {
	for _, password := range []string{"test-password", strings.Repeat("密", 24)} {
		t.Run(fmt.Sprintf("%d_bytes", len(password)), func(t *testing.T) {
			store := new(passwordStore)
			usePasswordStore(t, store)
			user := &models.User{UserID: 42, Username: "test-user", Password: password}
			if err := InsertUser(user); err != nil {
				t.Fatal(err)
			}
			firstHash := store.user.Password
			if store.user.UserID != user.UserID || store.user.Username != user.Username {
				t.Fatal("insert changed user identity")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(firstHash), []byte(password)); err != nil {
				t.Fatalf("stored password is not a valid bcrypt hash: %v", err)
			}
			if cost, err := bcrypt.Cost([]byte(firstHash)); err != nil || cost != bcrypt.DefaultCost {
				t.Fatalf("bcrypt cost = %d, err = %v", cost, err)
			}
			if user.Password != password {
				t.Fatal("insert mutated the caller password")
			}
			if err := InsertUser(user); err != nil {
				t.Fatal(err)
			}
			if store.user.Password == firstHash {
				t.Fatal("equal passwords must use independent random salts")
			}
		})
	}
}

func TestInsertUserRejectsOversizedPasswordAndDatabaseFailure(t *testing.T) {
	store := new(passwordStore)
	usePasswordStore(t, store)
	if err := InsertUser(&models.User{Password: strings.Repeat("密", 25)}); !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Fatalf("oversized password error = %v", err)
	}
	if store.execCount != 0 {
		t.Fatal("oversized password reached the database")
	}
	store.execErr = errors.New("insert unavailable")
	if err := InsertUser(&models.User{Password: "test-password"}); !errors.Is(err, store.execErr) {
		t.Fatalf("database error = %v", err)
	}
}

func TestLoginBcrypt(t *testing.T) {
	useLegacySecret(t, "") // New accounts do not require the legacy secret.
	password := strings.Repeat("密", 24)
	hash := bcryptPassword(t, password)
	for _, tc := range []struct {
		name, password string
		missing        bool
		queryErr, want error
	}{
		{name: "correct_72_byte_password", password: password},
		{name: "wrong_password", password: "wrong", want: ErrorInvalidPassword},
		{name: "over_72_bytes", password: password + "a", want: ErrorInvalidPassword},
		{name: "missing_user", password: password, missing: true, want: ErrorUserNotExits},
		{name: "query_failure", password: password, queryErr: io.ErrUnexpectedEOF, want: io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &passwordStore{user: models.User{UserID: 42, Username: "test-user", Password: hash}, missing: tc.missing, queryErr: tc.queryErr}
			usePasswordStore(t, store)
			err := Login(&models.User{Username: "test-user", Password: tc.password})
			if !errors.Is(err, tc.want) {
				t.Fatalf("login error = %v, want %v", err, tc.want)
			}
			if store.execCount != 0 {
				t.Fatal("bcrypt login wrote to the database")
			}
		})
	}
}

func TestLoginLegacyMigration(t *testing.T) {
	const password, secret = "test-password", "test-legacy-secret"
	for _, tc := range []struct {
		name, supplied, secret, stored string
		want                           error
	}{
		{name: "successful_upgrade", supplied: password, secret: secret, stored: legacyPassword(password, secret)},
		{name: "wrong_password", supplied: "wrong", secret: secret, stored: legacyPassword(password, secret), want: ErrorInvalidPassword},
		{name: "missing_legacy_secret", supplied: password, stored: legacyPassword(password, secret), want: ErrorInvalidPassword},
		{name: "malformed_legacy_hash", supplied: password, secret: secret, stored: "not-hex", want: ErrorInvalidPassword},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useLegacySecret(t, tc.secret)
			store := &passwordStore{user: models.User{UserID: 42, Username: "test-user", Password: tc.stored}}
			usePasswordStore(t, store)
			user := &models.User{Username: "test-user", Password: tc.supplied}
			if err := Login(user); !errors.Is(err, tc.want) {
				t.Fatalf("login error = %v, want %v", err, tc.want)
			}
			if tc.want != nil {
				if store.execCount != 0 || store.user.Password != tc.stored {
					t.Fatal("failed legacy authentication modified the password")
				}
				return
			}
			if store.execCount != 1 || user.Password != store.user.Password {
				t.Fatal("legacy password was not replaced in the database and result")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(store.user.Password), []byte(password)); err != nil {
				t.Fatalf("upgraded password mismatch: %v", err)
			}
		})
	}
}

func TestLoginLegacyConcurrentUpgrade(t *testing.T) {
	const password, secret = "test-password", "test-legacy-secret"
	useLegacySecret(t, secret)
	for _, tc := range []struct {
		name, currentPassword string
		want                  error
	}{
		{name: "another_request_upgraded_same_password", currentPassword: password},
		{name: "password_changed_concurrently", currentPassword: "changed-password", want: ErrorInvalidPassword},
	} {
		t.Run(tc.name, func(t *testing.T) {
			currentHash := bcryptPassword(t, tc.currentPassword)
			store := &passwordStore{user: models.User{UserID: 42, Username: "test-user", Password: legacyPassword(password, secret)}, conflictPassword: &currentHash}
			usePasswordStore(t, store)
			user := &models.User{Username: "test-user", Password: password}
			if err := Login(user); !errors.Is(err, tc.want) {
				t.Fatalf("login error = %v, want %v", err, tc.want)
			}
			if store.user.Password != currentHash {
				t.Fatal("login overwrote a concurrent password change")
			}
			if tc.want == nil && user.Password != currentHash {
				t.Fatal("successful concurrent upgrade did not return current hash")
			}
		})
	}
}

func TestLoginLegacyMigrationWriteFailure(t *testing.T) {
	const password, secret = "test-password", "test-legacy-secret"
	useLegacySecret(t, secret)
	for _, rowsFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("rows_affected_failure_%t", rowsFailure), func(t *testing.T) {
			failure := errors.New("migration write failed")
			store := &passwordStore{user: models.User{UserID: 42, Username: "test-user", Password: legacyPassword(password, secret)}}
			if rowsFailure {
				store.rowsErr = failure
			} else {
				store.execErr = failure
			}
			usePasswordStore(t, store)
			if err := Login(&models.User{Username: "test-user", Password: password}); !errors.Is(err, failure) {
				t.Fatalf("migration error = %v, want write failure", err)
			}
		})
	}
}

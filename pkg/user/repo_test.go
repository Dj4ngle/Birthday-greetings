package user

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthorize(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name     string
		username string
		password string
		mockFunc func()
		expected error
	}{
		{
			name:     "Valid user",
			username: "user1",
			password: "password1",
			mockFunc: func() {
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.DefaultCost)
				if err != nil {
					return
				}
				rows := sqlmock.NewRows([]string{"id", "username", "password"}).
					AddRow(1, "user1", string(hashedPassword))
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password FROM users WHERE username = ?")).
					WithArgs("user1").
					WillReturnRows(rows)
			},
			expected: nil,
		},
		{
			name:     "Invalid user",
			username: "user2",
			password: "password2",
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password FROM users WHERE username = ?")).
					WithArgs("user2").
					WillReturnError(sql.ErrNoRows)
			},
			expected: ErrNoUser,
		},
		{
			name:     "Invalid password",
			username: "user1",
			password: "wrongpassword",
			mockFunc: func() {
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.DefaultCost)
				if err != nil {
					return
				}
				rows := sqlmock.NewRows([]string{"id", "username", "password"}).
					AddRow(1, "user1", string(hashedPassword))
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, password FROM users WHERE username = ?")).
					WithArgs("user1").
					WillReturnRows(rows)
			},
			expected: ErrBadPass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.Authorize(tt.username, tt.password)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestMakeUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name     string
		username string
		password string
		mockFunc func()
		expected error
	}{
		{
			name:     "Create user",
			username: "user1",
			password: "password1",
			mockFunc: func() {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (`username`, `password`, `firstname`, `middlename`, `lastname`, `birthday`, `telegram`) VALUES (?, ?, ?, ?, ?, ?, ?)")).
					WithArgs("user1", sqlmock.AnyArg(), "John", "M", "Doe", "1990-01-01", "@john").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expected: nil,
		},
		{
			name:     "User exists",
			username: "user1",
			password: "password1",
			mockFunc: func() {
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (`username`, `password`, `firstname`, `middlename`, `lastname`, `birthday`, `telegram`) VALUES (?, ?, ?, ?, ?, ?, ?)")).
					WithArgs("user1", sqlmock.AnyArg(), "John", "M", "Doe", "1990-01-01", "@john").
					WillReturnError(ErrExists)
			},
			expected: ErrExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.MakeUser(tt.username, tt.password, "John", "M", "Doe", "1990-01-01", "@john")
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestGetUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name     string
		mockFunc func()
		expected error
	}{
		{
			name: "Get all users",
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "firstname", "middlename", "lastname", "birthday", "telegram"}).
					AddRow(1, "user1", "John", "M", "Doe", "1990-01-01", "@john").
					AddRow(2, "user2", "Jane", "D", "Smith", "1991-02-02", "@jane")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, firstname, middlename, lastname, birthday, telegram FROM users")).
					WillReturnRows(rows)
			},
			expected: nil,
		},
		{
			name: "No users",
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, firstname, middlename, lastname, birthday, telegram FROM users")).
					WillReturnError(sql.ErrNoRows)
			},
			expected: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.GetUsers()
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestSubscribe(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name         string
		userID       int64
		subscriberID int64
		typeOf       int
		mockFunc     func()
		expectedErr  error
	}{
		{
			name:         "Subscribe user",
			userID:       1,
			subscriberID: 2,
			typeOf:       1,
			mockFunc: func() {
				userRows := sqlmock.NewRows([]string{"id", "username", "telegram"}).
					AddRow(1, "user1", "@user1")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE id = ?")).
					WithArgs(1).
					WillReturnRows(userRows)
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO subscribes (`userID`, `subscriberID`) VALUES (?, ?)")).
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedErr: nil,
		},
		{
			name:         "Unsubscribe user",
			userID:       1,
			subscriberID: 2,
			typeOf:       0,
			mockFunc: func() {
				userRows := sqlmock.NewRows([]string{"id", "username", "telegram"}).
					AddRow(1, "user1", "@user1")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE id = ?")).
					WithArgs(1).
					WillReturnRows(userRows)
				mock.ExpectExec(regexp.QuoteMeta("DELETE FROM subscribes WHERE `userID` = ? and `subscriberID` = ?")).
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedErr: nil,
		},
		{
			name:         "Invalid user ID",
			userID:       3,
			subscriberID: 2,
			typeOf:       1,
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE id = ?")).
					WithArgs(3).
					WillReturnError(sql.ErrNoRows)
			},
			expectedErr: ErrNoUser,
		},
		{
			name:         "Invalid type",
			userID:       1,
			subscriberID: 2,
			typeOf:       2,
			mockFunc: func() {
				userRows := sqlmock.NewRows([]string{"id", "username", "telegram"}).
					AddRow(1, "user1", "@user1")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE id = ?")).
					WithArgs(1).
					WillReturnRows(userRows)
			},
			expectedErr: errors.New("not valid type"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			_, err := repo.Subscribe(tt.userID, tt.subscriberID, tt.typeOf)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestGetSubscribedUsers(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name        string
		userID      int64
		mockFunc    func()
		expected    []User
		expectedErr error
	}{
		{
			name:   "Get subscribed users",
			userID: 1,
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "firstname", "middlename", "lastname", "birthday", "telegram", "telegramID"}).
					AddRow(2, "user2", "John", "M", "Doe", "1990-01-01", "@john", 1234).
					AddRow(3, "user3", "Jane", "D", "Smith", "1991-02-02", "@jane", 5678)
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT u.id, u.username, u.firstname, u.middlename, u.lastname, u.birthday, u.telegram, u.telegramID
					FROM users u
					JOIN subscribes s ON u.id = s.subscriberID
					WHERE s.userID = ?`)).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expected: []User{
				{ID: 2, Username: "user2", FirstName: "John", MiddleName: "M", LastName: "Doe", Birthday: "1990-01-01", Telegram: "@john", TelegramID: 1234},
				{ID: 3, Username: "user3", FirstName: "Jane", MiddleName: "D", LastName: "Smith", Birthday: "1991-02-02", Telegram: "@jane", TelegramID: 5678},
			},
			expectedErr: nil,
		},
		{
			name:   "No subscribed users",
			userID: 1,
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT u.id, u.username, u.firstname, u.middlename, u.lastname, u.birthday, u.telegram, u.telegramID
					FROM users u
					JOIN subscribes s ON u.id = s.subscriberID
					WHERE s.userID = ?`)).
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows(nil))
			},
			expected:    nil,
			expectedErr: ErrNoUser,
		},
		{
			name:   "Query error",
			userID: 1,
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT u.id, u.username, u.firstname, u.middlename, u.lastname, u.birthday, u.telegram, u.telegramID
					FROM users u
					JOIN subscribes s ON u.id = s.subscriberID
					WHERE s.userID = ?`)).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			expected:    nil,
			expectedErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			users, err := repo.GetSubscribedUsers(tt.userID)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expected, users)
		})
	}
}

func TestGetUserByTelegram(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name        string
		telegram    string
		mockFunc    func()
		expected    *User
		expectedErr error
	}{
		{
			name:     "User exists",
			telegram: "@john",
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "telegram"}).
					AddRow(1, "user1", "@john")
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE telegram = ?")).
					WithArgs("@john").
					WillReturnRows(rows)
			},
			expected:    &User{ID: 1, Username: "user1", Telegram: "@john"},
			expectedErr: nil,
		},
		{
			name:     "User does not exist",
			telegram: "@nonexistent",
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, telegram FROM users WHERE telegram = ?")).
					WithArgs("@nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expected:    nil,
			expectedErr: ErrNoUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			user, err := repo.GetUserByTelegram(tt.telegram)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expected, user)
		})
	}
}

func TestGetUserByBirthday(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name        string
		month, day  int
		mockFunc    func()
		expected    []User
		expectedErr error
	}{
		{
			name:  "Users with birthday",
			month: 1,
			day:   1,
			mockFunc: func() {
				rows := sqlmock.NewRows([]string{"id", "username", "firstname", "middlename", "lastname", "birthday", "telegram"}).
					AddRow(1, "user1", "John", "M", "Doe", "1990-01-01", "@john").
					AddRow(2, "user2", "Jane", "D", "Smith", "1991-01-01", "@jane")
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, username, firstname, middlename, lastname, birthday, telegram
					FROM users
					WHERE MONTH(birthday) = ? AND DAY(birthday) = ?`)).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expected: []User{
				{ID: 1, Username: "user1", FirstName: "John", MiddleName: "M", LastName: "Doe", Birthday: "1990-01-01", Telegram: "@john"},
				{ID: 2, Username: "user2", FirstName: "Jane", MiddleName: "D", LastName: "Smith", Birthday: "1991-01-01", Telegram: "@jane"},
			},
			expectedErr: nil,
		},
		{
			name:  "No users with birthday",
			month: 1,
			day:   2,
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, username, firstname, middlename, lastname, birthday, telegram
					FROM users
					WHERE MONTH(birthday) = ? AND DAY(birthday) = ?`)).
					WithArgs(1, 2).
					WillReturnRows(sqlmock.NewRows(nil))
			},
			expected:    nil,
			expectedErr: ErrNoUser,
		},
		{
			name:  "Query error",
			month: 1,
			day:   1,
			mockFunc: func() {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT id, username, firstname, middlename, lastname, birthday, telegram
					FROM users
					WHERE MONTH(birthday) = ? AND DAY(birthday) = ?`)).
					WithArgs(1, 1).
					WillReturnError(sql.ErrConnDone)
			},
			expected:    nil,
			expectedErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			users, err := repo.GetUserByBirthday(tt.month, tt.day)
			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expected, users)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewMysqlRepo(db)

	tests := []struct {
		name        string
		telegramID  int64
		telegram    string
		mockFunc    func()
		expectedErr error
	}{
		{
			name:       "Update user",
			telegramID: 1234,
			telegram:   "john",
			mockFunc: func() {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET telegramID = ? WHERE telegram = ?")).
					WithArgs(1234, "@john").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedErr: nil,
		},
		{
			name:       "No rows updated",
			telegramID: 1234,
			telegram:   "john",
			mockFunc: func() {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET telegramID = ? WHERE telegram = ?")).
					WithArgs(1234, "@john").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedErr: fmt.Errorf("no rows updated"),
		},
		{
			name:       "Update error",
			telegramID: 1234,
			telegram:   "john",
			mockFunc: func() {
				mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET telegramID = ? WHERE telegram = ?")).
					WithArgs(1234, "@john").
					WillReturnError(sql.ErrConnDone)
			},
			expectedErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()
			err := repo.UpdateUser(tt.telegramID, tt.telegram)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

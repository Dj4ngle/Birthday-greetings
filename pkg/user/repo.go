package user

import (
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

var (
	ErrNoUser  = errors.New("no user found")
	ErrBadPass = errors.New("invalid password")
	ErrExists  = errors.New("already exists")
)

type UserMysqlRepository struct {
	DB *sql.DB
}

func NewMysqlRepo(db *sql.DB) *UserMysqlRepository {
	return &UserMysqlRepository{DB: db}
}

func (repo *UserMysqlRepository) Authorize(username, pass string) (*User, error) {
	user := &User{}

	err := repo.DB.
		QueryRow("SELECT id, username, password FROM users WHERE username = ?", username).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, ErrNoUser
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pass))
	if err != nil {
		return nil, ErrBadPass
	}

	return user, nil
}

func (repo *UserMysqlRepository) MakeUser(username, pass, firstname, middlename, lastname, birthday, telegram string) (*User, error) {
	hashedPass, err := hashPassword(pass)
	if err != nil {
		return nil, err
	}

	result, err := repo.DB.Exec(
		"INSERT INTO users (`username`, `password`, `firstname`, `middlename`, `lastname`, `birthday`, `telegram`) VALUES (?, ?, ?, ?, ?, ?, ?)",
		username,
		hashedPass,
		firstname,
		middlename,
		lastname,
		birthday,
		telegram,
	)
	if err != nil {
		return nil, ErrExists
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &User{ID: userID, Username: username}, nil
}

func hashPassword(password string) (string, error) {
	cost := bcrypt.DefaultCost
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func (repo *UserMysqlRepository) GetUsers() ([]User, error) {
	rows, err := repo.DB.Query("SELECT id, username, firstname, middlename, lastname, birthday, telegram FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.MiddleName, &user.LastName, &user.Birthday, &user.Telegram); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrNoUser
	}

	return users, nil
}

func (repo *UserMysqlRepository) Subscribe(userID int64, subscriberID int64, typeOf int) (*User, error) {
	user := &User{}

	err := repo.DB.
		QueryRow("SELECT id, username, telegram FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Username, &user.Telegram)
	if err != nil {
		return nil, ErrNoUser
	}

	switch typeOf {
	case 1:
		_, err = repo.DB.Exec(
			"INSERT INTO subscribes (`userID`, `subscriberID`) VALUES (?, ?)",
			userID,
			subscriberID,
		)
		if err != nil {
			return nil, ErrExists
		}
	case 0:
		_, err = repo.DB.Exec(
			"DELETE FROM subscribes WHERE `userID` = ? and `subscriberID` = ?",
			userID,
			subscriberID,
		)
		if err != nil {
			return nil, ErrExists
		}
	default:
		return nil, errors.New("not valid type")

	}

	return user, nil
}

func (repo *UserMysqlRepository) GetSubscribedUsers(userID int64) ([]User, error) {
	rows, err := repo.DB.Query(`
		SELECT u.id, u.username, u.firstname, u.middlename, u.lastname, u.birthday, u.telegram, u.telegramID
		FROM users u
		JOIN subscribes s ON u.id = s.subscriberID
		WHERE s.userID = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.MiddleName, &user.LastName, &user.Birthday, &user.Telegram, &user.TelegramID); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrNoUser
	}

	return users, nil
}

func (repo *UserMysqlRepository) GetUserByTelegram(telegram string) (*User, error) {
	user := &User{}

	err := repo.DB.
		QueryRow("SELECT id, username, telegram FROM users WHERE telegram = ?", telegram).
		Scan(&user.ID, &user.Username, &user.Telegram)
	if err != nil {
		return nil, ErrNoUser
	}

	return user, nil
}

func (repo *UserMysqlRepository) GetUserByBirthday(month, day int) ([]User, error) {
	rows, err := repo.DB.Query(`
		SELECT id, username, firstname, middlename, lastname, birthday, telegram
		FROM users
		WHERE MONTH(birthday) = ? AND DAY(birthday) = ?`, month, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err = rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.MiddleName, &user.LastName, &user.Birthday, &user.Telegram); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, ErrNoUser
	}

	return users, nil
}

func (repo *UserMysqlRepository) UpdateUser(telegramID int64, telegram string) error {
	result, err := repo.DB.Exec(
		"UPDATE users SET telegramID = ? WHERE telegram = ?",
		telegramID,
		"@"+telegram,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated")
	}

	return nil
}

package user

type User struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	FirstName  string `json:"firstname"`
	MiddleName string `json:"middlename"`
	LastName   string `json:"lastname"`
	Password   string `json:"password,omitempty" bson:"password,omitempty"`
	Birthday   string `json:"birthday"`
	Telegram   string `json:"telegram"`
	TelegramID int64  `json:"telegramid"`
}

type UserRepo interface {
	Authorize(username, pass string) (*User, error)
	MakeUser(username, pass, firstname, middlename, lastname, birthday, telegram string) (*User, error)
	GetUsers() ([]User, error)
	Subscribe(userID int64, subscriberID int64, typeOf int) (*User, error)
	GetSubscribedUsers(userID int64) ([]User, error)
	GetUserByTelegram(telegram string) (*User, error)
	UpdateUser(telegramID int64, telegram string) error
}

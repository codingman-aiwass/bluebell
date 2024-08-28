package models

type User struct {
	UserId   int64  `db:"user_id"`
	Username string `db:"username"`
	Password string `db:"password"`
	Gender   string `db:"gender"`
	Email    string `db:"email"`
	Verified int    `db:"verified"`
}

package model

import "time"

type User struct {
	ID                int64     `db:"id"`
	UserLogin         string    `db:"user_login"`
	UserPass          string    `db:"user_pass"`
	UserNicename      string    `db:"user_nicename"`
	UserEmail         string    `db:"user_email"`
	UserURL           string    `db:"user_url"`
	UserRegistered    time.Time `db:"user_registered"`
	UserActivationKey string    `db:"user_activation_key"`
	UserStatus        int       `db:"user_status"`
	DisplayName       string    `db:"display_name"`
}

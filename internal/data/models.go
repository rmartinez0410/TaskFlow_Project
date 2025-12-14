package data

import (
	"database/sql"
	"errors"
)

var ErrNoRecord = errors.New("no record found")

type Models struct {
	UserModel
	SessionModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		UserModel: UserModel{
			DB: db,
		},

		SessionModel: SessionModel{
			DB: db,
		},
	}
}

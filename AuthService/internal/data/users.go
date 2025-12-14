package data

import (
	"auth/internal/validator"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrDuplicateEmail = errors.New("duplicate email")

type User struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Password  password `json:"-"`
	Activated bool     `json:"activated"`
	Username  string   `json:"username"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func ValidateUser(u *User, v *validator.Validator) {
	v.Check(u.Username != "", "username", "must be provided")
	v.Check(len(u.Username) <= 100, "username", "must not be more than 100 characters")
	v.Check(len(u.Username) >= 4, "username", "must not be less than 4 characters")

	ValidateEmail(u.Email, v)
	if u.Password.plaintext != nil {
		ValidatePasswordPlainText(*u.Password.plaintext, v)
	}
}

func ValidateEmail(email string, v *validator.Validator) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid address")
}

func ValidatePasswordPlainText(p string, v *validator.Validator) {
	v.Check(p != "", "password", "must be provided")
	v.Check(len(p) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(p) <= 72, "password", "must not be more than 72 bytes long")
}

type password struct {
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintext *string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(*plaintext), 12)
	if err != nil {
		return err
	}

	p.plaintext = plaintext
	p.hash = hash

	return nil
}

func (p *password) Matches(plaintext *string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(*plaintext))

	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

type UserModel struct {
	DB *sql.DB
}

func (u *UserModel) Insert(user *User) error {
	query := `INSERT INTO users (email, password_hash, username) 
	VALUES ($1, $2, $3)
	RETURNING id, created_at, updated_at`

	err := u.DB.QueryRow(query, user.Email, user.Password.hash, user.Username).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return ErrDuplicateEmail
		}
		return err
	}

	return nil
}

func (u *UserModel) Delete(id string) error {
	query := `DELETE FROM users WHERE id = $1`

	_, err := u.DB.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

func (u *UserModel) Update(user *User) error {
	query := `UPDATE users SET
	username = $1,
	activated = $2,
	password_hash = $3
	WHERE id = $4`

	_, err := u.DB.Exec(query, user.Username, user.Activated, user.Password.hash, user.ID)
	if err != nil {
		return err
	}

	return nil
}

func (u *UserModel) GetByEmail(email string) (*User, error) {
	query := `SELECT id, email, username, password_hash, activated, created_at, updated_at 
	FROM users WHERE email = $1`
	var user User

	err := u.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Password.hash,
		&user.Activated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return &user, nil
}

func (u *UserModel) GetByID(id string) (*User, error) {
	query := `SELECT id, email, username, password_hash, activated, created_at, updated_at 
	FROM users WHERE id = $1`
	var user User

	err := u.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Password.hash,
		&user.Activated,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return &user, nil
}

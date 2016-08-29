package models

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	passwordSaltBytes = 32
	passwordHashBytes = 64
)

// User contains the information of an User
type User struct {
	ID           int            `db:"id"`
	FirstName    string         `db:"first_name"`
	LastName     string         `db:"last_name"`
	Email        string         `db:"email"`
	Address      sql.NullString `db:"address"`
	Invites      int            `db:"invites"`
	Credit       int            `db:"credit"`
	Confirmed    bool           `db:"confirmed"`
	ReferrerHash string         `db:"referrer_hash"`
	ReferredBy   sql.NullInt64  `db:"referred_by"`
	PasswordSalt string         `db:"password_salt"`
	PasswordHash string         `db:"password_hash"`
}

// Update updates the current User struct into the database
func (u User) Update(fields ...string) error {
	return nil
}

// Insert inserts the current User struct into the database and returns an error
// if something goes wrong.
func (u User) Insert() error {
	if u.ID != 0 {
		return nil
	}

	_, err := db.NamedExec(`INSERT INTO users
            (first_name,
             last_name,
             email,
             address,
             invites,
             credit,
             confirmed,
             referrer_hash,
             referred_by,
             password_salt,
             password_hash)
VALUES      (:first_name,
             :last_name,
             :email,
             :address,
             :invites,
             :credit,
             :confirmed,
             :referrer_hash,
             :referred_by,
             :password_salt,
             :password_hash)`, u)

	return err
}

// SetPassword generates the salt and the hash of the user password
func (u *User) SetPassword(password string) error {
	salt := make([]byte, passwordSaltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return err
	}

	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, passwordHashBytes)
	if err != nil {
		return err
	}

	u.PasswordSalt = hex.EncodeToString(salt)
	u.PasswordHash = hex.EncodeToString(hash)
	return nil
}

// CheckPassword checks if the password of the user is correct
func (u User) CheckPassword(password string) (bool, error) {
	salt, err := hex.DecodeString(u.PasswordSalt)
	if err != nil {
		return false, err
	}

	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, passwordHashBytes)
	if err != nil {
		return false, err
	}

	return (hex.EncodeToString(hash) == u.PasswordHash), nil
}

// ChangePassword allows you to change the password of the user
func (u *User) ChangePassword(oldHash, newHash string) error {
	return nil
}

// GenerateReferrerHash generates a new referrer hash for a new user
func (u *User) GenerateReferrerHash() {
	if u.ReferrerHash != "" {
		return
	}

	data := u.Email + time.Now().Format(time.ANSIC)
	hash := sha256.Sum256([]byte(data))
	u.ReferrerHash = hex.EncodeToString(hash[:])
}

// DeleteUser deletes a user from the database using its email
func DeleteUser(email string) error {
	_, err := db.Exec("DELETE FROM users WHERE email=?", email)
	return err
}

// GetUser retrieves a user from the database using its email
func GetUser(email string) (*User, error) {
	user := User{}
	err := db.Get(&user, "SELECT * FROM users WHERE email=?", email)
	return &user, err
}
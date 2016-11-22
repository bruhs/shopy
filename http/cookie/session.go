package cookie

import (
	"encoding/gob"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/upframe/fest"
)

// SessionCookie ...
type session struct {
	Logged bool
	UserID int
}

func init() {
	gob.Register(&session{})
}

// SessionService ...
type SessionService struct {
	Store       *securecookie.SecureCookie
	UserService fest.UserService
	Secure      bool
}

// Save ...
func (s *SessionService) Save(w http.ResponseWriter, sess *fest.Session) error {
	obj := &session{Logged: sess.Logged}
	if sess.Logged {
		obj.UserID = sess.User.ID
	}

	encoded, err := s.Store.Encode("session", obj)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     "session",
		Value:    encoded,
		Path:     "/",
		MaxAge:   10800,
		Secure:   s.Secure,
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	return nil
}

// Get ...
func (s *SessionService) Get(w http.ResponseWriter, r *http.Request) (*fest.Session, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return &fest.Session{Logged: false}, s.Reset(w)
	}

	var value *session
	// if the decryption keys aren't right
	err = s.Store.Decode("session", cookie.Value, &value)
	if err != nil {
		return &fest.Session{Logged: false}, s.Reset(w)
	}

	object := &fest.Session{Logged: value.Logged}
	if value.Logged {
		object.User = &fest.User{}
	}

	if value.Logged {
		object.User, err = s.UserService.Get(value.UserID)
		if err != nil {
			return object, err
		}
	}

	return object, nil
}

// Reset ...
func (s *SessionService) Reset(w http.ResponseWriter) error {
	return s.Save(w, &fest.Session{Logged: false})
}

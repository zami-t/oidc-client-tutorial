package model

import "time"

// AppSession represents a logged-in user's server-side session.
type AppSession struct {
	id         string
	subject    string
	issuer     string
	email      string
	name       string
	picture    string
	createdAt  time.Time
	lastSeenAt time.Time
}

// NewAppSession creates a new AppSession.
func NewAppSession(id, subject, issuer, email, name, picture string) AppSession {
	now := time.Now()
	return AppSession{
		id:         id,
		subject:    subject,
		issuer:     issuer,
		email:      email,
		name:       name,
		picture:    picture,
		createdAt:  now,
		lastSeenAt: now,
	}
}

func (s AppSession) Id() string          { return s.id }
func (s AppSession) Subject() string     { return s.subject }
func (s AppSession) Issuer() string      { return s.issuer }
func (s AppSession) Email() string       { return s.email }
func (s AppSession) Name() string        { return s.name }
func (s AppSession) Picture() string     { return s.picture }
func (s AppSession) CreatedAt() time.Time  { return s.createdAt }
func (s AppSession) LastSeenAt() time.Time { return s.lastSeenAt }

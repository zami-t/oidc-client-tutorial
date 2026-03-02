package model

// AppSession represents a logged-in user's server-side session.
type AppSession struct {
	id      string
	subject string
	issuer  string
	email   string
	name    string
	picture string
}

// NewAppSession creates a new AppSession.
func NewAppSession(id, subject, issuer, email, name, picture string) AppSession {
	return AppSession{
		id:      id,
		subject: subject,
		issuer:  issuer,
		email:   email,
		name:    name,
		picture: picture,
	}
}

func (s AppSession) Id() string      { return s.id }
func (s AppSession) Subject() string { return s.subject }
func (s AppSession) Issuer() string  { return s.issuer }
func (s AppSession) Email() string   { return s.email }
func (s AppSession) Name() string    { return s.name }
func (s AppSession) Picture() string { return s.picture }

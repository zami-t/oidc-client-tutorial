package model

import "time"

// AuthorizationTransaction holds the state stored during an authorization flow.
// It is persisted between the /login redirect and the /callback.
type AuthorizationTransaction struct {
	state     string
	nonce     string
	returnTo  string
	idp       string
	createdAt time.Time
}

// NewAuthorizationTransaction creates a new AuthorizationTransaction.
func NewAuthorizationTransaction(state, nonce, returnTo, idp string) AuthorizationTransaction {
	return AuthorizationTransaction{
		state:     state,
		nonce:     nonce,
		returnTo:  returnTo,
		idp:       idp,
		createdAt: time.Now(),
	}
}

func (t AuthorizationTransaction) State() string       { return t.state }
func (t AuthorizationTransaction) Nonce() string       { return t.nonce }
func (t AuthorizationTransaction) ReturnTo() string    { return t.returnTo }
func (t AuthorizationTransaction) Idp() string         { return t.idp }
func (t AuthorizationTransaction) CreatedAt() time.Time { return t.createdAt }

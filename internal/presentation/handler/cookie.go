package handler

import "net/http"

// CookieManager issues and clears the session cookie using the configured attributes.
type CookieManager struct {
	secure   bool
	sameSite http.SameSite
}

// NewCookieManager creates a CookieManager with the given cookie attributes.
func NewCookieManager(secure bool, sameSite http.SameSite) *CookieManager {
	return &CookieManager{secure: secure, sameSite: sameSite}
}

func (p *CookieManager) newSessionCookie(sessionId string) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    sessionId,
		Path:     "/",
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: p.sameSite,
	}
}

func (p *CookieManager) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: p.sameSite,
		MaxAge:   -1,
	}
}

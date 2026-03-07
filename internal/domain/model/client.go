package model

// RedirectUri represents the URI to which the OP redirects after authorization.
type RedirectUri string

// NewRedirectUri creates a RedirectUri value object.
func NewRedirectUri(value string) RedirectUri { return RedirectUri(value) }

// String returns the redirect URI string.
func (r RedirectUri) String() string { return string(r) }

// Client holds the RP credentials registered with an OpenID Provider.
type Client struct {
	id          string
	secret      string
	redirectUri RedirectUri
}

// NewClient creates a Client value object.
func NewClient(id, secret string, redirectUri RedirectUri) Client {
	return Client{id: id, secret: secret, redirectUri: redirectUri}
}

func (c Client) Id() string               { return c.id }
func (c Client) Secret() string           { return c.secret }
func (c Client) RedirectUri() RedirectUri { return c.redirectUri }

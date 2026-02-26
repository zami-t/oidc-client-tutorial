package dto

// MeInput is the input for the me usecase.
type MeInput struct {
	SessionId string
}

// MeOutput is the output from the me usecase.
type MeOutput struct {
	Subject string
	Issuer  string
	Email   string
	Name    string
	Picture string
}

package dto

// LoginInput is the input for the login usecase.
type LoginInput struct {
	Idp      string
	ReturnTo string
}

// LoginOutput is the output from the login usecase.
type LoginOutput struct {
	RedirectUrl string
}

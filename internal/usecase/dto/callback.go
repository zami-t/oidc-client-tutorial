package dto

import "errors"

// CallbackParams holds raw query parameters from the callback HTTP request.
type CallbackParams struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// CallbackInput is the validated input for the callback usecase.
type CallbackInput struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// NewCallbackInput validates params and constructs a CallbackInput.
// state is always required. Either code (success) or error (OP error) must be present.
func NewCallbackInput(p CallbackParams) (CallbackInput, error) {
	if p.State == "" {
		return CallbackInput{}, errors.New("state parameter missing")
	}
	if p.Code == "" && p.Error == "" {
		return CallbackInput{}, errors.New("either code or error parameter is required")
	}
	return CallbackInput{
		Code:             p.Code,
		State:            p.State,
		Error:            p.Error,
		ErrorDescription: p.ErrorDescription,
	}, nil
}

// CallbackOutput is the output from the callback usecase.
type CallbackOutput struct {
	ReturnTo  string
	SessionId string
}

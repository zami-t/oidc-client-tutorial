package dto

// CallbackInput is the input for the callback usecase.
type CallbackInput struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

// CallbackOutput is the output from the callback usecase.
type CallbackOutput struct {
	ReturnTo  string
	SessionId string
}

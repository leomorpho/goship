package emailsubscriptions

import (
	"errors"
	"fmt"
)

type ErrAlreadySubscribed struct {
	EmailList string
	Err       error
}

func (e *ErrAlreadySubscribed) Error() string {
	return fmt.Sprintf("email address is already subscribed to email list %s, error is: %v", e.EmailList, e.Err)
}

var ErrInvalidEmailConfirmationCode = errors.New("email confirmation code is invalid")
var ErrEmailSyntaxInvalid = errors.New("email address syntax is invalid")
var ErrEmailAddressInvalidCatchAll = errors.New("invalid email address")
var ErrListNotAllowed = errors.New("subscription list is not allowed by catalog")
var ErrListInactive = errors.New("subscription list is inactive")
var ErrInvalidUnsubscribeToken = errors.New("unsubscribe token is invalid")

type ErrEmailVerificationFailed struct {
	Err error
}

func (e *ErrEmailVerificationFailed) Error() string {
	return fmt.Sprintf("verify email address failed, error is: %v", e.Err)
}

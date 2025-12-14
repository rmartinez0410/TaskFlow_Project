package validator

import (
	"regexp"
)

var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

type Validator struct {
	Errors map[string]string
}

func New() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) Check(ok bool, key, msg string) {
	if !ok {
		v.Errors[key] = msg
	}
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

func (v *Validator) AddError(key, msg string) {
	if _, ok := v.Errors[key]; !ok {
		v.Errors[key] = msg
	}
}

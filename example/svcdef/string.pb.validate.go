// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: string.proto

package svcdef

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = ptypes.DynamicAny{}
)

// Validate checks the field values on StringMsg with the rules defined in the
// proto definition for this message. If any rules are violated, an error is returned.
func (m *StringMsg) Validate() error {
	if m == nil {
		return nil
	}

	if l := utf8.RuneCountInString(m.GetStr()); l < 5 || l > 100 {
		return StringMsgValidationError{
			field:  "Str",
			reason: "value length must be between 5 and 100 runes, inclusive",
		}
	}

	return nil
}

// StringMsgValidationError is the validation error returned by
// StringMsg.Validate if the designated constraints aren't met.
type StringMsgValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e StringMsgValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e StringMsgValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e StringMsgValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e StringMsgValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e StringMsgValidationError) ErrorName() string { return "StringMsgValidationError" }

// Error satisfies the builtin error interface
func (e StringMsgValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sStringMsg.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = StringMsgValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = StringMsgValidationError{}
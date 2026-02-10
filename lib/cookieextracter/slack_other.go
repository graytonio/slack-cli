//go:build !darwin

package cookieextracter

import "errors"

type SlackCredentials struct {
	Cookie    string `mapstructure:"cookie"`
	UserToken string `mapstructure:"token"`
}

var (
	ErrSlackCookieNotFound = errors.New("could not find slack cookie")
	ErrSlackTokenNotFound  = errors.New("could not find slack user token")
	ErrNotSupported        = errors.New("credential extraction is not supported on this platform")
)

func GetSlackCredentials(workspace string) (*SlackCredentials, error) {
	return nil, ErrNotSupported
}

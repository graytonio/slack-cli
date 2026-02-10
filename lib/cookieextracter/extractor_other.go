//go:build !darwin

package cookieextracter

import "errors"

type BrowserCookieConfig struct {
	CookiePath  string
	LevelDBPath string
	Iterations  int
	Password    string
}

var ErrCouldNotFindCookieFile = errors.New("could not find cookie file: not supported on this platform")

func GetDarwinConfig() (*BrowserCookieConfig, error) {
	return nil, ErrCouldNotFindCookieFile
}

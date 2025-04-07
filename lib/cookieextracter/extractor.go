package cookieextracter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/keybase/go-keychain"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/crypto/pbkdf2"
)

var home, _ = os.UserHomeDir()
var darwinCookiePaths = []string{path.Join(home, "Library/Application Support/Slack/Cookies"), path.Join(home, "Library/Containers/com.tinyspeck.slackmacgap/Data/Library/Application Support/Slack/Cookies")}

type BrowserCookieConfig struct {
	CookiePath string
	LevelDBPath string
	Iterations int
	Password string
}

var (
	ErrCouldNotFindCookieFile = errors.New("could not find cookie file")
)

func GetDarwinConfig() (*BrowserCookieConfig, error) {
	config := BrowserCookieConfig{
		Iterations: 1003,
		LevelDBPath: path.Join(home, "Library/Application Support/Slack/Local Storage/leveldb"),
	}

	for _, f := range darwinCookiePaths {
		logrus.WithField("path", f).Debug("looking for cookie path")
		if _, err := os.Open(f); err != nil {
			logrus.WithError(err).Debug("could not find cookie path")
			continue
		}

		config.CookiePath = f
		break
	}

	if config.CookiePath == "" {
		return nil, ErrCouldNotFindCookieFile
	}

	logrus.Debug("fetching slack keys")
	password, err := keychain.GetGenericPassword("Slack Safe Storage", "Slack Key", "", "")
	if err != nil {
		return nil, err
	}

	config.Password = string(password)

	return &config, nil
}

var (
	salt = []byte("saltysalt")
	keyLen = 16
)

func generateHostKeys(hostname string) (keys []string) {
	if hostname == "localhost" {
		return []string{hostname}
	}

	labels := strings.Split(hostname, ".")
	for i := 2; i<len(labels) + 1; i++ {
		domain := strings.Join(labels[len(labels)-i:], ".")
		keys = append(keys, domain, "." + domain)
	}

	return keys
}

func DecryptCookie(encryptedValue []byte, key []byte, initVector []byte) (string, error) {
	if len(encryptedValue) == 0 {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	cbc := cipher.NewCBCDecrypter(block, initVector)
	plainText := make([]byte, len(encryptedValue))
	cbc.CryptBlocks(plainText, encryptedValue[3:])
	return string(plainText), nil
}

type LocalData struct {
	Teams map[string]TokenData `json:"teams"`
}

type TokenData struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Token string `json:"token"`
	Domain string `json:"domain"`
}

func GetSlackUserTokens(path string) (*LocalData, error) {
	logrus.WithField("path", path).Debug("opening local config db")
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var data = LocalData{}
	iter := db.NewIterator(nil, nil)
	logrus.Debug("iterating local config db")
	for iter.Next() {
		key := iter.Key()
		if strings.Contains(string(key), "localConfig_v2") {
			logrus.WithField("data", string(iter.Value())).Debug("found local config key")
			err := json.Unmarshal(iter.Value()[1:], &data)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	iter.Release()

	return &data, nil
}

type CookieData struct {
	HostKey string
	Path string
	Name string
	Value string
	EncryptedValue string
}

func getDBVersion(db *sql.DB) (int, error) {
	sql := "select value from meta where key = 'version';"
	rows, err := db.Query(sql)
	if err != nil {
	  return -1, err
	}

	db_version := 0
	for rows.Next() {
		if err := rows.Scan(&db_version); err != nil {
			return -1, err
		}
	}

	return db_version, nil
}

func GetSlackCookies(config *BrowserCookieConfig) ([]CookieData, error) {
	df := pbkdf2.Key([]byte(config.Password), salt, config.Iterations, keyLen, sha1.New)

	logrus.WithField("cookies_path", config.CookiePath).Debug("opening cookies sqlite db")
	db, err := sql.Open("sqlite3", config.CookiePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	db_version, err := getDBVersion(db)
	if err != nil {
	  return nil, err
	}

	data := []CookieData{}
	sql := "select host_key, path, name, encrypted_value from cookies where host_key like ?"
	
	for _, host_key := range generateHostKeys("slack.com") {
		logrus.WithField("host_key", host_key).Debug("querying for keys")
		rows, err := db.Query(sql, host_key)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			c := CookieData{}
			if err := rows.Scan(&c.HostKey, &c.Path, &c.Name, &c.EncryptedValue); err != nil {
				return nil, err
			}
			logrus.WithFields(logrus.Fields{
				"data": fmt.Sprintf("%+v", c),
			}).Debug("parsed row")

			logrus.WithField("encrypted_data", c.EncryptedValue).Debug("decrypting cookie")
			value, err := DecryptCookie([]byte(c.EncryptedValue), df, []byte("                "))
			if err != nil {
				return nil, err
			}
			logrus.WithField("cookie", value).Debug("successfully decrypted cookie")

			// Cookies in database version 24 and later include a SHA256
        	// hash of the domain to the start of the encrypted value.
        	// https://github.com/chromium/chromium/blob/280265158d778772c48206ffaea788c1030b9aaa/net/extras/sqlite/sqlite_persistent_cookie_store.cc#L223-L224
			if db_version >= 24 {
				value = value[32:]
			}

			c.Value = value
			data = append(data, c)
		}
	}

	return data, nil
}
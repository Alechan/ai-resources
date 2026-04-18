package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1" //nolint:gosec
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite" // register sqlite driver
)

// CookieProvider returns HTTP cookies for use in requests.
type CookieProvider interface {
	Cookies() ([]*http.Cookie, error)
}

// ChromeCookieProvider reads and decrypts cookies from Chrome's SQLite cookies database.
type ChromeCookieProvider struct {
	path string
}

// NewChromeCookieProvider creates a provider for the given cookies file path.
// A leading "~" is expanded to the user's home directory.
func NewChromeCookieProvider(path string) *ChromeCookieProvider {
	return &ChromeCookieProvider{path: expandHome(path)}
}

// Path returns the resolved cookies file path.
func (p *ChromeCookieProvider) Path() string { return p.path }

// Cookies opens the Chrome cookies database and returns decrypted DataDog cookies.
func (p *ChromeCookieProvider) Cookies() ([]*http.Cookie, error) {
	// Chrome holds a shared lock on the file while running; copy to a temp location.
	tmpDir, err := os.MkdirTemp("", "ddctl-cookies-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dst := filepath.Join(tmpDir, "Cookies")
	if err := copyFile(p.path, dst); err != nil {
		return nil, fmt.Errorf("copy cookies file: %w", err)
	}

	db, err := sql.Open("sqlite", dst)
	if err != nil {
		return nil, fmt.Errorf("open cookies db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, encrypted_value FROM cookies WHERE host_key LIKE '%datadoghq.com'")
	if err != nil {
		return nil, fmt.Errorf("query cookies: %w", err)
	}
	defer rows.Close()

	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	var cookies []*http.Cookie
	for rows.Next() {
		var name string
		var encVal []byte
		if err := rows.Scan(&name, &encVal); err != nil {
			continue
		}
		value, err := decryptChromeValue(key, encVal)
		if err != nil {
			continue
		}
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cookies: %w", err)
	}
	return cookies, nil
}

// deriveKey shells out to the macOS keychain to get the Chrome Safe Storage password,
// then derives an AES-128 key via PBKDF2.
func deriveKey() ([]byte, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", "Chrome Safe Storage", "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("keychain access: %w", err)
	}
	password := strings.TrimSpace(string(out))
	//nolint:gosec // SHA1 is mandated by Chrome's key derivation spec
	key := pbkdf2.Key([]byte(password), []byte("saltysalt"), 1003, 16, sha1.New)
	return key, nil
}

// decryptChromeValue decrypts a Chrome cookie value encrypted with AES-128-CBC.
// Chrome prepends "v10" or "v11" to the ciphertext on macOS.
func decryptChromeValue(key, encVal []byte) (string, error) {
	if len(encVal) < 3 {
		return string(encVal), nil
	}
	prefix := string(encVal[:3])
	if prefix != "v10" && prefix != "v11" {
		return string(encVal), nil
	}
	ciphertext := encVal[3:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = ' '
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	plaintext = pkcs7Unpad(plaintext)
	return string(plaintext), nil
}

func pkcs7Unpad(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > aes.BlockSize || pad > len(b) {
		return b
	}
	return b[:len(b)-pad]
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

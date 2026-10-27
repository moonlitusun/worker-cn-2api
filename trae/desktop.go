package trae

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Codec constants from the installed client's byteCrypto.js; these are public
// format salts, not account secrets. Only the current tc/AES format is accepted.
var codecSaltA = []byte{82, 9, 106, 213, 48, 54, 165, 56, 191, 64, 163, 158, 129, 243, 215, 251, 124, 227, 57, 130, 155, 47, 255, 135, 52, 142, 67, 68, 196, 222, 233, 203, 84, 123, 148, 50, 166, 194, 35, 61, 238, 76, 149, 11, 66, 250, 195, 78, 8, 46, 161, 102, 40, 217, 36, 178, 118, 91, 162, 73, 109, 139, 209, 37}
var codecSaltB = []byte{31, 221, 168, 51, 136, 7, 199, 49, 177, 18, 16, 89, 39, 128, 236, 95, 96, 81, 127, 169, 25, 181, 74, 13, 45, 229, 122, 159, 147, 201, 156, 239, 160, 224, 59, 77, 174, 42, 245, 176, 200, 235, 187, 60, 131, 83, 153, 97, 23, 43, 4, 126, 186, 119, 214, 38, 225, 105, 20, 99, 85, 33, 12, 125}

func decodeDesktop(encoded string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(b) < 54 || string(b[:6]) != "tc\x05\x10\x00\x00" || (len(b)-38)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("unsupported Trae desktop credential format")
	}
	salt := make([]byte, 64)
	for i := range salt {
		salt[i] = codecSaltA[i] ^ codecSaltB[i]
	}
	seedHash := sha512.Sum512(b[6:38])
	material := sha512.Sum512(append(seedHash[:], salt...))
	block, err := aes.NewCipher(material[:16])
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(b)-38)
	cipher.NewCBCDecrypter(block, material[16:32]).CryptBlocks(plain, b[38:])
	pad := int(plain[len(plain)-1])
	if pad < 1 || pad > aes.BlockSize || len(plain) < pad {
		return nil, fmt.Errorf("invalid Trae credential padding")
	}
	for _, v := range plain[len(plain)-pad:] {
		if int(v) != pad {
			return nil, fmt.Errorf("invalid Trae credential padding")
		}
	}
	plain = plain[:len(plain)-pad]
	if len(plain) < 64 {
		return nil, fmt.Errorf("invalid Trae credential payload")
	}
	hash := sha512.Sum512(plain[64:])
	if subtle.ConstantTimeCompare(hash[:], plain[:64]) != 1 {
		return nil, fmt.Errorf("Trae credential integrity check failed")
	}
	return plain[64:], nil
}

func detectDesktop() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	base := filepath.Join(home, ".config")
	if runtime.GOOS == "darwin" {
		base = filepath.Join(home, "Library", "Application Support")
	}
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
		if base == "" {
			return Config{}, fmt.Errorf("APPDATA is unset")
		}
	}
	for _, app := range []string{"TRAE SOLO CN", "Trae-CN"} {
		file := filepath.Join(base, app, "User", "globalStorage", "storage.json")
		data, err := os.ReadFile(file)
		if err != nil || len(data) > 10<<20 {
			continue
		}
		var storage map[string]json.RawMessage
		if json.Unmarshal(data, &storage) != nil {
			continue
		}
		var encoded string
		if json.Unmarshal(storage["iCubeAuthInfo://icube.cloudide"], &encoded) != nil {
			continue
		}
		plain, err := decodeDesktop(encoded)
		if err != nil {
			continue
		}
		var auth struct {
			Token        string `json:"token"`
			UserID       string `json:"userId"`
			RefreshToken string `json:"refreshToken"`
			ExpiresAt    string `json:"expiredAt"`
		}
		if json.Unmarshal(plain, &auth) != nil || auth.Token == "" {
			continue
		}
		clientID := "ono9krqynydwx5"
		if app == "TRAE SOLO CN" {
			clientID = "en1oxy7wnw8j9n"
		}
		return Config{Token: auth.Token, UserID: auth.UserID, RefreshToken: auth.RefreshToken, ExpiresAt: auth.ExpiresAt, ClientID: clientID}, nil
	}
	return Config{}, fmt.Errorf("no supported Trae CN desktop login found")
}

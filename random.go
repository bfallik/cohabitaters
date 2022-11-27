package cohabitaters

import (
	"crypto/rand"
	"encoding/base64"
)

func RandBase64() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

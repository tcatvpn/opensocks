package cipher

import (
	"crypto/sha256"
	"math/rand"
	"strings"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

var nonce = make([]byte, chacha20poly1305.NonceSizeX)
var chars = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖabcdefghijklmnopqrstuvwxyzåäö0123456789")

func CreateHash(key string) []byte {
	sha := sha256.Sum256([]byte(key))
	ret := make([]byte, 32)
	copy(sha[:32], ret[:32])
	return ret
}

func Encrypt(data *[]byte, key []byte) {
	aead, _ := chacha20poly1305.NewX(key)
	ciphertext := aead.Seal(nil, nonce, *data, nil)
	data = &ciphertext
}

func Decrypt(data *[]byte, key []byte) {
	aead, _ := chacha20poly1305.NewX(key)
	plaintext, _ := aead.Open(nil, nonce, *data, nil)
	data = &plaintext
}

func Random() string {
	rand.Seed(time.Now().UnixNano())
	length := 8 + rand.Intn(8)
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

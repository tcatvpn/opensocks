package cipher

import (
	"crypto/sha256"
	"math/rand"
	"strings"
	"time"
)

var _chars = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖabcdefghijklmnopqrstuvwxyzåäö0123456789")
var _key = []byte("SpUsXuZw4z6B9EbGdKgNjQnTqVsYv2x5")

func GenerateKey(key string) {
	sha := sha256.Sum256([]byte(key))
	buff := make([]byte, 32)
	copy(sha[:32], buff[:32])
	_key = buff
}

func XOR(src []byte) []byte {
	_klen := len(_key)
	for i := 0; i < len(src); i++ {
		src[i] ^= _key[i%_klen]
	}
	return src
}

func Random() string {
	rand.Seed(time.Now().UnixNano())
	length := 8 + rand.Intn(8)
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(_chars[rand.Intn(len(_chars))])
	}
	return b.String()
}

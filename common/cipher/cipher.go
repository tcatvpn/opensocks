package cipher

import (
	"crypto/rc4"
	"crypto/sha256"
	"log"
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
	c, err := rc4.NewCipher(_key)
	if err != nil {
		log.Fatalln(err)
	}
	dst := make([]byte, len(src))
	c.XORKeyStream(dst, src)
	return dst
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

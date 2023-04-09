package main

import (
    "math/rand"
	"crypto/aes"
	"encoding/hex"
)

var l = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func rnd(n int) string {
    s := make([]rune, n)
    for i := range s {
        s[i] = l[rand.Intn(len(l))]
    }
    return string(s)
}

func encrypt(k string, m string) string {
	c, _ := aes.NewCipher([]byte(k))
	
	msg := make([]byte, len(m))
	c.Encrypt(msg, []byte(m))
	return hex.EncodeToString(msg)
}

func decrypt(k string, m string) string {
	txt, _ := hex.DecodeString(m)
	c, _ := aes.NewCipher([]byte(k))
	
	msg := make([]byte, len(txt))
	c.Decrypt(msg, []byte(txt))

	return string(msg[:])
}

func main() {
	for {
		key := rnd(32)
		decrypt(key, encrypt(key, key))
	}
}

package main

import (
    "math/rand"
	"crypto/aes"
	"encoding/hex"
	"runtime"
	"os"
    "strconv"
	"fmt"
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
	var data = make([]byte, 1)
    memPercentageStr := os.Getenv("MEMORY_PERCENTAGE")
    if memPercentageStr != "" {
		memPercentage, err := strconv.ParseFloat(memPercentageStr, 64)
		if err != nil {
			fmt.Println("Invalid memory percentage:", memPercentageStr)
			os.Exit(1)
		}
	
		mem := runtime.MemStats{}
		runtime.ReadMemStats(&mem)
		totalMem := mem.Sys
	
		memToAllocate := uint64(float64(totalMem) * memPercentage / 100)
	
		data = make([]byte, memToAllocate)
		data[0] = 1
	}
	for {
		key := rnd(32)
		decrypt(key, encrypt(key, key))
	}
}

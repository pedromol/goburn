package main

import (
	"bufio"
	"crypto/aes"

	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
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

func readMemoryTotal() (int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		text := strings.ReplaceAll(line[:len(line)-2], " ", "")
		keyValue := strings.Split(text, ":")

		switch keyValue[0] {
		case "MemTotal":
			value, err := strconv.Atoi(keyValue[1])
			if err != nil {
				return 0, err
			}
			return value, nil
		}
	}
	return 0, errors.New("unable to read total memory size")
}

func main() {
	var memToAllocate uint64
	var data = make([]byte, 0)

	memPercentageStr := os.Getenv("MEMORY_PERCENTAGE")
	if memPercentageStr != "" {
		memPercentage, err := strconv.ParseFloat(memPercentageStr, 64)
		if err != nil {
			fmt.Println("Invalid memory percentage:", err)
			os.Exit(1)
		}

		totalMem, err := readMemoryTotal()
		if err != nil {
			fmt.Println("Failed to read total memory:", err)
			os.Exit(1)
		}

		memToAllocate = uint64(float64(totalMem) * memPercentage / 100)
		data = make([]byte, memToAllocate*1000)

		fmt.Printf("Allocated %.2f MB of memory (%.2f%% of total memory)\n", float64(memToAllocate)/1000, memPercentage)
		fmt.Printf("Total memory: %.2f MB\n", float64(totalMem)/1000)

	}
	for {
		for i := range data {
			data[i] = byte([]byte(rnd(1))[0])
			key := rnd(32)
			decrypt(key, encrypt(key, key))
		}
	}
}

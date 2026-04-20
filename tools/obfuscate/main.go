// Command obfuscate emits go-build ldflag fragments that bake a
// XOR-obfuscated third-party provider key into the binary.
//
// Usage:
//
//	go run ./tools/obfuscate <obfVar> <xorVar> <plaintext>
//
// Prints a pair of -X ldflag fragments on stdout that the
// Dockerfile/Makefile/CI splice into go build:
//
//	-X github.com/beacon-stack/prism/internal/config.<obfVar>=<b64>
//	-X github.com/beacon-stack/prism/internal/config.<xorVar>=<b64>
//
// Empty plaintext yields empty ldflag values so unset env vars flow
// through without breaking the build. See pilot/tools/obfuscate for
// the originating pattern.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

const modulePath = "github.com/beacon-stack/prism/internal/config"

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: %s <obfVar> <xorVar> <plaintext>\n", os.Args[0])
		os.Exit(2)
	}
	obfVar := os.Args[1]
	xorVar := os.Args[2]
	plain := []byte(os.Args[3])

	obf, xor, err := obfuscate(plain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "obfuscate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("-X %s.%s=%s -X %s.%s=%s", modulePath, obfVar, obf, modulePath, xorVar, xor)
}

func obfuscate(plain []byte) (ciphertextB64, keyB64 string, err error) {
	if len(plain) == 0 {
		return "", "", nil
	}
	key := make([]byte, len(plain))
	if _, err := rand.Read(key); err != nil {
		return "", "", err
	}
	ct := make([]byte, len(plain))
	for i := range plain {
		ct[i] = plain[i] ^ key[i]
	}
	return base64.StdEncoding.EncodeToString(ct), base64.StdEncoding.EncodeToString(key), nil
}

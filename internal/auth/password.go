package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MiB in KiB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	encoded := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

func CheckPassword(encoded, password string) bool {
	salt, expected, params, err := decodeArgon2Hash(encoded)
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(expected, actual) == 1
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func decodeArgon2Hash(encoded string) (salt, hash []byte, params argon2Params, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, params, errors.New("invalid argon2 hash")
	}
	if parts[2] != "v=19" {
		return nil, nil, params, errors.New("unsupported argon2 version")
	}

	for _, field := range strings.Split(parts[3], ",") {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			return nil, nil, params, errors.New("invalid argon2 params")
		}
		val, convErr := strconv.ParseUint(kv[1], 10, 32)
		if convErr != nil {
			return nil, nil, params, errors.New("invalid argon2 params")
		}
		switch kv[0] {
		case "m":
			params.memory = uint32(val)
		case "t":
			params.time = uint32(val)
		case "p":
			params.threads = uint8(val)
		}
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, params, err
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, params, err
	}
	return salt, hash, params, nil
}

package agent

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

const registrationKeyBytes = 32

func discoveryIdentity(key string) (string, string, error) {
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) != registrationKeyBytes {
		return "", "", errors.New(errRegistrationKeyInvalid)
	}
	nodeHash := sha256.Sum256(append([]byte("openflare:discovery:node:v1:"), decoded...))
	tokenHash := sha256.Sum256(append([]byte("openflare:discovery:token:v1:"), decoded...))
	return "node-" + base64.RawURLEncoding.EncodeToString(nodeHash[:]), hex.EncodeToString(tokenHash[:]), nil
}

package node

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ledgerwatch/erigon/common"
)

var (
	errFileNotFound = errors.New("file not found")
	ErrStateToken   = errors.New("state JWT token")
)

type JWTAuth struct {
	SignedTime time.Time
	JWTSecret  *[32]byte
}

func ReadJWTAuthSecret(path string) (*[32]byte, error) {
	if path == "" {
		return nil, errFileNotFound
	}
	var secret [32]byte
	if data, err := os.ReadFile(path); err == nil {
		jwtSecret := common.FromHex(strings.TrimSpace(string(data)))
		if len(jwtSecret) != 32 {
			return nil, fmt.Errorf("invalid jwt secret in path %s, not 32 hex-formatted bytes", path)
		}
		copy(secret[:], jwtSecret)
		return &secret, nil
	} else {
		return nil, fmt.Errorf("failed to read file: %s", err.Error())
	}
}

func (j *JWTAuth) NewJWTAuth(client RPC, jwtSecret *[32]byte) error {
	currentTime := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": &jwt.NumericDate{Time: currentTime},
	})
	s, err := token.SignedString(jwtSecret[:])
	if err != nil {
		return fmt.Errorf("failed to create JWT token: %w", err)
	}
	j.SignedTime = currentTime
	j.JWTSecret = jwtSecret
	client.SetHeader("Authorization", "Bearer "+s)
	return nil
}

func (j *JWTAuth) RefreshJWTAuth(client RPC) error {
	if time.Since(j.SignedTime) > time.Second*30 {
		return j.NewJWTAuth(client, j.JWTSecret)
	}
	return nil
}

package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadJWTAuthSecret(t *testing.T) {
	_, err := ReadJWTAuthSecret("")
	require.Equal(t, err, errFileNotFound)
	jwt, err := ReadJWTAuthSecret("./testdata/jwt_secret.txt")
	require.NoError(t, err)
	require.Equal(t, [32]byte{}, *jwt)
}

func TestNewJWTAuth(t *testing.T) {
	jwt := JWTAuth{}
	mockClient := &MockRPC{}
	jwtSecret := [32]byte{1}
	err := jwt.NewJWTAuth(mockClient, &jwtSecret)
	require.NoError(t, err)
	require.Equal(t, *jwt.JWTSecret, jwtSecret)
}

func TestRefreshJWTAuth(t *testing.T) {
	jwt := JWTAuth{}
	mockClient := &MockRPC{}
	oldSignedTime := time.Now().Add(-time.Second * 31)
	jwt.SignedTime = oldSignedTime
	jwt.JWTSecret = &[32]byte{1}
	err := jwt.RefreshJWTAuth(mockClient)
	require.NoError(t, err)
	require.NotEqual(t, jwt.SignedTime, oldSignedTime)
}

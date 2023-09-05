package node

import (
	"testing"
	"time"

	"github.com/ledgerwatch/erigon/rpc"
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
	client := &BackendRPC{
		client: &rpc.Client{},
	}
	jwtSecret := [32]byte{1}
	err := jwt.NewJWTAuth(client, &jwtSecret)
	require.NoError(t, err)
	require.Equal(t, *jwt.JWTSecret, jwtSecret)
}

func TestRefreshJWTAuth(t *testing.T) {
	jwt := JWTAuth{}
	client := &BackendRPC{
		client: &rpc.Client{},
	}
	oldSignedTime := time.Now().Add(-time.Second * 31)
	jwt.SignedTime = oldSignedTime
	jwt.JWTSecret = &[32]byte{1}
	err := jwt.RefreshJWTAuth(client)
	require.NoError(t, err)
	require.NotEqual(t, jwt.SignedTime, oldSignedTime)
}

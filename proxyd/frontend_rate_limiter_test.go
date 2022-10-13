package proxyd

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestFrontendRateLimiter(t *testing.T) {
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("127.0.0.1:%s", redisServer.Port()),
	})

	max := 2
	lims := []struct {
		name string
		frl  FrontendRateLimiter
	}{
		{"memory", NewMemoryFrontendRateLimit(2*time.Second, max)},
		{"redis", NewRedisFrontendRateLimiter(redisClient, 2*time.Second, max, "")},
	}

	for _, cfg := range lims {
		frl := cfg.frl
		ctx := context.Background()
		t.Run(cfg.name, func(t *testing.T) {
			for i := 0; i < 4; i++ {
				ok, err := frl.Take(ctx, "foo")
				require.NoError(t, err)
				require.Equal(t, i < max, ok)
				ok, err = frl.Take(ctx, "bar")
				require.NoError(t, err)
				require.Equal(t, i < max, ok)
			}
			time.Sleep(2 * time.Second)
			for i := 0; i < 4; i++ {
				ok, _ := frl.Take(ctx, "foo")
				require.Equal(t, i < max, ok)
				ok, _ = frl.Take(ctx, "bar")
				require.Equal(t, i < max, ok)
			}
		})
	}
}

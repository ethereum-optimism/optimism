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

	lims := []struct {
		name string
		frl  FrontendRateLimiter
	}{
		{"memory", NewMemoryFrontendRateLimit(2 * time.Second)},
		{"redis", NewRedisFrontendRateLimiter(redisClient, 2*time.Second)},
	}

	max := 2
	for _, cfg := range lims {
		frl := cfg.frl
		ctx := context.Background()
		t.Run(cfg.name, func(t *testing.T) {
			for i := 0; i < 4; i++ {
				ok, err := frl.Take(ctx, "foo", max)
				require.NoError(t, err)
				require.Equal(t, i < max, ok)
				ok, err = frl.Take(ctx, "bar", max)
				require.NoError(t, err)
				require.Equal(t, i < max, ok)
			}
			time.Sleep(2 * time.Second)
			for i := 0; i < 4; i++ {
				ok, _ := frl.Take(ctx, "foo", max)
				require.Equal(t, i < max, ok)
				ok, _ = frl.Take(ctx, "bar", max)
				require.Equal(t, i < max, ok)
			}
		})
	}
}

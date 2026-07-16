package presets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFetchGaugeSelectsLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `# TYPE op_dispute_mon_games gauge
op_dispute_mon_games{game_type="super-cannon-kona"} 2
op_dispute_mon_games{game_type="super-permissioned"} 1
`)
	}))
	defer server.Close()

	value, payload, err := fetchGauge(
		context.Background(),
		server.Client(),
		server.URL,
		"op_dispute_mon_games",
		map[string]string{"game_type": "super-permissioned"},
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), value)
	require.Contains(t, payload, `game_type="super-permissioned"`)
}

func TestWaitForGaugeRetriesUntilExpectedValue(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		value := 0
		if requests.Add(1) >= 3 {
			value = 1
		}
		fmt.Fprintf(w, "# TYPE op_dispute_mon_games gauge\nop_dispute_mon_games{game_type=\"super-permissioned\"} %d\n", value)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := waitForGauge(
		ctx,
		server.Client(),
		server.URL,
		"op_dispute_mon_games",
		map[string]string{"game_type": "super-permissioned"},
		1,
		10*time.Millisecond,
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, requests.Load(), int32(3))
}

func TestWaitForGaugeTimeoutIncludesLastObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "# TYPE op_dispute_mon_games gauge\nop_dispute_mon_games{game_type=\"super-permissioned\"} 0\n")
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := waitForGauge(
		ctx,
		server.Client(),
		server.URL,
		"op_dispute_mon_games",
		map[string]string{"game_type": "super-permissioned"},
		1,
		10*time.Millisecond,
	)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "expected 1") && strings.Contains(err.Error(), "observed 0"), err)
	require.Contains(t, err.Error(), "op_dispute_mon_games")
}

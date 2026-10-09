package main

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/stretchr/testify/require"
)

func sampleGames() []gameRecord {
	return []gameRecord{
		{
			Index: 2188, Game: "0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c", GameType: 0,
			Timestamp: 1780752851, Created: "2026-06-06T09:34:11-04:00", L2BlockNumber: 47253642,
			OutputRoot: "0x62dc7ddcee7f846d0b12d74cdf08ec851c883c201240edc41a3281e44ec299e8",
			ClaimCount: ptr.New(uint64(41)), Status: "In Progress",
		},
		{
			Index: 2172, Game: "0xc0B7Ea85D376F61ED820b1F74b05161acf3Dee6a", GameType: 1,
			Timestamp: 1780400000, Created: "2026-06-02T09:30:59-04:00", L2BlockNumber: 46907480,
			OutputRoot: "0xf5bfdaca6f0dda93ef406c0b74ce70ee54e630c028c26337fa044cff2e47f1f1",
			ClaimCount: ptr.New(uint64(1)), Status: "Defender Won",
		},
		{
			Index: 2190, Game: "0x1234567890AbcdEF1234567890aBcdef12345678", GameType: 10,
			Timestamp: 1780760000, Created: "2026-06-06T11:33:20-04:00", L2BlockNumber: 1780759000,
			OutputRoot: "0x0aa2c3ad4c3c4b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3c4b5a",
			Status:     "In Progress",
		},
	}
}

func TestRenderGamesJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderGamesJSON(&buf, sampleGames()))

	var got struct {
		Games []gameRecord `json:"games"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Games, 3)
	require.Equal(t, uint64(2188), got.Games[0].Index)
	require.Equal(t, uint64(47253642), got.Games[0].L2BlockNumber)
	require.Equal(t, ptr.New(uint64(41)), got.Games[0].ClaimCount)
	require.Equal(t, "In Progress", got.Games[0].Status)
	require.Equal(t, uint32(1), got.Games[1].GameType)
	require.Equal(t, "Defender Won", got.Games[1].Status)

	var raw struct {
		Games []map[string]any `json:"games"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	require.Contains(t, raw.Games[0], "claimCount")
	require.NotContains(t, raw.Games[2], "claimCount")
}

func TestRenderGamesText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderGamesText(&buf, sampleGames()))
	out := buf.String()
	require.Contains(t, out, "Idx ")
	require.Contains(t, out, "Output Root")
	require.Contains(t, out, "0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c")
	require.Contains(t, out, "In Progress")
	require.Contains(t, out, "Defender Won")

	// Columns: idx, game, type, created date, created time, L2 block, output root, claims, status.
	claimsColumn := func(game string) string {
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, game) {
				return strings.Fields(line)[7]
			}
		}
		t.Fatalf("no row for game %v", game)
		return ""
	}
	require.Equal(t, "41", claimsColumn("0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c"))
	require.Equal(t, "-", claimsColumn("0x1234567890AbcdEF1234567890aBcdef12345678"))
}

func TestCompareClaimCountsPutsMissingCountsLast(t *testing.T) {
	counts := []*uint64{nil, ptr.New(uint64(3)), nil, ptr.New(uint64(1)), ptr.New(uint64(2))}
	sorted := func(desc bool) []*uint64 {
		out := slices.Clone(counts)
		slices.SortFunc(out, func(a, b *uint64) int { return compareClaimCounts(a, b, desc) })
		return out
	}
	require.Equal(t, []*uint64{ptr.New(uint64(1)), ptr.New(uint64(2)), ptr.New(uint64(3)), nil, nil}, sorted(false))
	require.Equal(t, []*uint64{ptr.New(uint64(3)), ptr.New(uint64(2)), ptr.New(uint64(1)), nil, nil}, sorted(true))
}

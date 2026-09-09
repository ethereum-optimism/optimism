package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleGames() []gameRecord {
	return []gameRecord{
		{
			Index: 2188, Game: "0xf63aF5d56AA0aD2331FAcFFb87BF23BA1136880c", GameType: 0,
			Timestamp: 1780752851, Created: "2026-06-06T09:34:11-04:00", L2BlockNumber: 47253642,
			OutputRoot: "0x62dc7ddcee7f846d0b12d74cdf08ec851c883c201240edc41a3281e44ec299e8",
			ClaimCount: 41, Status: "In Progress",
		},
		{
			Index: 2172, Game: "0xc0B7Ea85D376F61ED820b1F74b05161acf3Dee6a", GameType: 1,
			Timestamp: 1780400000, Created: "2026-06-02T09:30:59-04:00", L2BlockNumber: 46907480,
			OutputRoot: "0xf5bfdaca6f0dda93ef406c0b74ce70ee54e630c028c26337fa044cff2e47f1f1",
			ClaimCount: 1, Status: "Defender Won",
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
	require.Len(t, got.Games, 2)
	require.Equal(t, uint64(2188), got.Games[0].Index)
	require.Equal(t, uint64(47253642), got.Games[0].L2BlockNumber)
	require.Equal(t, uint64(41), got.Games[0].ClaimCount)
	require.Equal(t, "In Progress", got.Games[0].Status)
	require.Equal(t, uint32(1), got.Games[1].GameType)
	require.Equal(t, "Defender Won", got.Games[1].Status)
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
}

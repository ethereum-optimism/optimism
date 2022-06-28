package l1

import (
	"net/http"
	"strconv"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/server"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
)

func (s *Service) GetIndexerStatus(w http.ResponseWriter, r *http.Request) {
	highestBlock, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var synced float64
	if s.latestHeader != 0 {
		synced = float64(highestBlock.Number) / float64(s.latestHeader)
	}

	status := &IndexerStatus{
		Synced:  synced,
		Highest: *highestBlock,
	}

	server.RespondWithJSON(w, http.StatusOK, status)
}

func (s *Service) GetDeposits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil && limitStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if limit == 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil && offsetStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page := db.PaginationParam{
		Limit:  uint64(limit),
		Offset: uint64(offset),
	}

	deposits, err := s.cfg.DB.GetDepositsByAddress(common.HexToAddress(vars["address"]), page)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondWithJSON(w, http.StatusOK, deposits)
}

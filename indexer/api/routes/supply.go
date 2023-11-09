package routes

import (
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/models"
)

// SupplyView ... Handles /api/v0/supply GET requests
func (h Routes) SupplyView(w http.ResponseWriter, r *http.Request) {

	depositSum, err := h.view.L1BridgeDepositSum()
	if err != nil {
		http.Error(w, "internal server error reading deposits", http.StatusInternalServerError)
		h.logger.Error("unable to read deposits from DB", "err", err.Error())
		return
	}

	withdrawalSum, err := h.view.L2BridgeWithdrawalSum()
	if err != nil {
		http.Error(w, "internal server error reading withdrawals", http.StatusInternalServerError)
		h.logger.Error("unable to read withdrawals from DB", "err", err.Error())
		return
	}

	view := models.BridgeSupplyView{
		L1DepositSum:    depositSum,
		L2WithdrawalSum: withdrawalSum,
	}

	err = jsonResponse(w, view, http.StatusOK)
	if err != nil {
		h.logger.Error("error writing response", "err", err)
	}
}

package catalyst

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
)

var (
	requiredProtocolDeltaGauge    = metrics.NewRegisteredGauge("superchain/required/delta", nil)
	recommendedProtocolDeltaGauge = metrics.NewRegisteredGauge("superchain/recommended/delta", nil)
)

type SuperchainSignal struct {
	Recommended params.ProtocolVersion `json:"recommended"`
	Required    params.ProtocolVersion `json:"required"`
}

func (api *ConsensusAPI) SignalSuperchainV1(signal *SuperchainSignal) (params.ProtocolVersion, error) {
	if signal == nil {
		log.Info("Received empty superchain version signal", "local", params.OPStackSupport)
		return params.OPStackSupport, nil
	}
	// update metrics and log any warnings/info
	requiredProtocolDeltaGauge.Update(int64(params.OPStackSupport.Compare(signal.Required)))
	recommendedProtocolDeltaGauge.Update(int64(params.OPStackSupport.Compare(signal.Recommended)))
	logger := log.New("local", params.OPStackSupport, "required", signal.Required, "recommended", signal.Recommended)
	LogProtocolVersionSupport(logger, params.OPStackSupport, signal.Recommended, "recommended")
	LogProtocolVersionSupport(logger, params.OPStackSupport, signal.Required, "required")

	if err := api.eth.HandleRequiredProtocolVersion(signal.Required); err != nil {
		log.Error("Failed to handle required protocol version", "err", err, "required", signal.Required)
		return params.OPStackSupport, err
	}

	return params.OPStackSupport, nil
}

func LogProtocolVersionSupport(logger log.Logger, local, other params.ProtocolVersion, name string) {
	switch local.Compare(other) {
	case params.AheadMajor:
		logger.Info(fmt.Sprintf("Ahead with major %s protocol version change", name))
	case params.AheadMinor, params.AheadPatch, params.AheadPrerelease:
		logger.Debug(fmt.Sprintf("Ahead with compatible %s protocol version change", name))
	case params.Matching:
		logger.Debug(fmt.Sprintf("Latest %s protocol version is supported", name))
	case params.OutdatedMajor:
		logger.Error(fmt.Sprintf("Outdated with major %s protocol change", name))
	case params.OutdatedMinor:
		logger.Warn(fmt.Sprintf("Outdated with minor backward-compatible %s protocol change", name))
	case params.OutdatedPatch:
		logger.Info(fmt.Sprintf("Outdated with support backward-compatible %s protocol change", name))
	case params.OutdatedPrerelease:
		logger.Debug(fmt.Sprintf("New %s protocol pre-release is available", name))
	case params.DiffBuild:
		logger.Debug(fmt.Sprintf("Ignoring %s protocolversion signal, local build is different", name))
	case params.DiffVersionType:
		logger.Warn(fmt.Sprintf("Failed to recognize %s protocol version signal version-type", name))
	case params.EmptyVersion:
		logger.Debug(fmt.Sprintf("No %s protocol version available to check", name))
	case params.InvalidVersion:
		logger.Warn(fmt.Sprintf("Invalid protocol version comparison with %s", name))
	}
}

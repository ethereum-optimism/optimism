package utils

import (
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type CrossDomainMessengerSentMessage struct {
	*bindings.CrossDomainMessengerSentMessage
	Value       *big.Int
	MessageHash common.Hash
}

func ParseCrossDomainMessage(sentMessageReceipt *types.Receipt) (CrossDomainMessengerSentMessage, error) {
	abi, err := bindings.CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return CrossDomainMessengerSentMessage{}, err
	}

	sentMessageEventAbi := abi.Events["SentMessage"]
	messenger, err := bindings.NewCrossDomainMessenger(common.Address{}, nil)
	if err != nil {
		return CrossDomainMessengerSentMessage{}, err
	}

	for i, log := range sentMessageReceipt.Logs {
		if len(log.Topics) > 0 && log.Topics[0] == sentMessageEventAbi.ID {
			sentMessage, err := messenger.ParseSentMessage(*log)
			if err != nil {
				return CrossDomainMessengerSentMessage{}, err
			}
			sentMessageExtension, err := messenger.ParseSentMessageExtension1(*sentMessageReceipt.Logs[i+1])
			if err != nil {
				return CrossDomainMessengerSentMessage{}, err
			}

			msg := crossdomain.NewCrossDomainMessage(
				sentMessage.MessageNonce,
				sentMessage.Sender,
				sentMessage.Target,
				sentMessageExtension.Value,
				sentMessage.GasLimit,
				sentMessage.Message,
			)

			msgHash, err := msg.Hash()
			if err != nil {
				return CrossDomainMessengerSentMessage{}, err
			}

			return CrossDomainMessengerSentMessage{sentMessage, sentMessageExtension.Value, msgHash}, nil
		}
	}

	return CrossDomainMessengerSentMessage{}, errors.New("missing SentMessage receipts")
}

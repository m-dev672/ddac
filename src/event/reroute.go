package event

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mdev672/ddac/config"
	"github.com/mdev672/ddac/contract"
	"github.com/mdev672/ddac/sqlWrapper"
)

func reroute(destination [16]byte) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		return err
	}

	var destinationHash common.Hash
	copy(destinationHash[:], destination[:])

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   nil,
		Addresses: []common.Address{dispatcherAddr},
		Topics: [][]common.Hash{
			{dispatcherABI.Events["FlightPlanSubmitted"].ID},
			{destinationHash},
		},
	}

	config, err := config.Load()
	if err != nil {
		return err
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return err
	}
	defer client.Close()

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		return err
	}

	for _, vLog := range logs {
		var event struct {
			Destination [16]byte
			Nonce       *big.Int
			Query       string
			Operator    common.Address
		}

		err := dispatcherABI.UnpackIntoInterface(&event, "FlightPlanSubmitted", vLog.Data)
		if err != nil {
			return err
		} else {
			write, parsedQueries, err := sqlWrapper.Parse(event.Query)
			if err != nil {
				return err
			}

			if checkPermission(write, destination, event.Operator) {
				sqlWrapper.Exec(destination, parsedQueries)
			} else {
				return err
			}
		}
	}

	return nil
}

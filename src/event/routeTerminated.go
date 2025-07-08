package event

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/mdev672/ddac/contract"
)

func CheckRouteTerminatedEvent(client *ethclient.Client, airportCode uuid.UUID, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		fmt.Printf("(General)>> Error: %s\n", err)
	}

	var originHash common.Hash
	copy(originHash[:], airportCode[:])

	query := ethereum.FilterQuery{
		Addresses: []common.Address{dispatcherAddr},
		Topics: [][]common.Hash{
			{dispatcherABI.Events["RouteTerminated"].ID},
			{originHash},
		},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("(General)>> Error: %s\n", err)
	}

	fmt.Println("(General)>> Your airport has been opened (1/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println("(General)>> Your airport has been closed (1/2).")
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Origin      [16]byte
				Destination [16]byte
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "RouteTerminated", vLog.Data)
			if err != nil {
				fmt.Printf("(General)>> Error: %s\n", err)
			} else {
				fmt.Printf("(%x)>> The route has been terminated.", event.Destination)

				close(routes[event.Destination].Instance)
				routes[event.Destination].Client.Close()
			}
		}
	}
}

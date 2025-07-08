package event

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/mdev672/ddac/config"
	"github.com/mdev672/ddac/contract"
)

type Route struct {
	Client            *ethclient.Client
	Instance          chan struct{}
	CanWriteAccount   []common.Address
	DefaultPermission string
}

var routes = make(map[[16]byte]Route)

func CheckNewRouteLaunchedEvent(client *ethclient.Client, airportCode uuid.UUID, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		fmt.Printf("(General)>> Error: %s\n", err)
	}

	var originHash common.Hash
	copy(originHash[:], airportCode[:])

	query := ethereum.FilterQuery{
		Addresses: []common.Address{dispatcherAddr},
		Topics: [][]common.Hash{
			{dispatcherABI.Events["NewRouteLaunched"].ID},
			{originHash},
		},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("(General)>> Error: %s\n", err)
	}

	fmt.Println("(General)>> Your airport has been opened (2/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println("(General)>> Your airport has been closed (2/2).")
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Origin                  [16]byte
				Destination             [16]byte
				CanWriteAccount         []common.Address
				DefaultPermission       string
				Reroute                 bool
				EstablishedOriginIPAddr [4]byte
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "NewRouteLaunched", vLog.Data)
			if err != nil {
				fmt.Printf("(%x)>> Error: %s\n", event.Destination, err)
			} else {
				config, err := config.Load()
				if err != nil {
					fmt.Printf("(%x)>> Error: %s\n", event.Destination, err)
				}

				client, err := ethclient.Dial(config.RPCEndpoint)
				if err != nil {
					fmt.Printf("(%x)>> Error: %s\n", event.Destination, err)
				}

				routes[event.Destination] = Route{
					client,
					make(chan struct{}),
					event.CanWriteAccount,
					event.DefaultPermission,
				}

				if event.Reroute {
					err := reroute(event.Destination)
					if err != nil {
						fmt.Printf("(%x)>> Error: %s\n", event.Destination, err)
					}
				}

				go CheckFlightPlanSubmittedEvent(airportCode, event.Destination)
			}
		}
	}
}

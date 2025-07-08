package event

import (
	"context"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/mdev672/ddac/contract"
	"github.com/mdev672/ddac/sqlWrapper"

	_ "github.com/mattn/go-sqlite3"
)

func sendHash(airportCode [16]byte, destination [16]byte, nonce *big.Int, hash [32]byte) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("reportPos", airportCode, destination, nonce, hash)

	err = contract.SendTx(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return err
}

func checkPermission(write bool, destination [16]byte, operator common.Address) bool {
	if write && slices.Contains(routes[destination].CanWriteAccount, operator) {
		return true
	} else if routes[destination].DefaultPermission == "write" {
		return true
	} else if !write {
		return true
	}

	return false
}

func CheckFlightPlanSubmittedEvent(airportCode, destination [16]byte) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		fmt.Printf("(%x)>> Error: %s\n", destination, err)
	}

	var destinationHash common.Hash
	copy(destinationHash[:], destination[:])

	query := ethereum.FilterQuery{
		Addresses: []common.Address{dispatcherAddr},
		Topics: [][]common.Hash{
			{dispatcherABI.Events["FlightPlanSubmitted"].ID},
			{destinationHash},
		},
	}

	logs := make(chan types.Log)
	sub, err := routes[destination].Client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Printf("(%x)>> Error: %s\n", destination, err)
	}

	fmt.Printf("(%x)>> New route has been launched.\n", destination)

	for {
		select {
		case <-routes[destination].Instance:
			fmt.Printf("(%x)>> The route has been terminated.\n", destination)
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Destination [16]byte
				Nonce       *big.Int
				Query       string
				Operator    common.Address
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "FlightPlanSubmitted", vLog.Data)
			if err != nil {
				fmt.Printf("(%x)>> Error: %s\n", destination, err)
			} else {
				write, parsedQueries, err := sqlWrapper.Parse(event.Query)
				if err != nil {
					fmt.Printf("(%x)>> Error: %s\n", destination, err)
				}

				if checkPermission(write, destination, event.Operator) {
					hash := sqlWrapper.Exec(destination, parsedQueries)
					sendHash(airportCode, destination, event.Nonce, hash)
				} else {
					fmt.Printf("(%x)>> Error: No proper permissions.\n", destination)
				}
			}
		}
	}
}

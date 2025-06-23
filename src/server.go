package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/rqlite/sql"
)

var airportCode [16]byte

type Config struct {
	RPCEndpoint           string `json:"rpcEndpoint"`
	DispatcherAddr        string `json:"dispatcherAddr"`
	DispatcherABIFilePath string `json:"dispatcherABIFilePath"`
	PrivateKey            string `json:"privateKey"`
}

func loadConfig() (Config, error) {
	var config Config

	configBytes, err := os.ReadFile("config.json")
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configBytes, &config)

	return config, nil
}

func saveConfig(config Config) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	err = os.WriteFile("config.json", configBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func sendTransaction(contractAddr common.Address, txData []byte, amount *big.Int) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}
	senderAddr := crypto.PubkeyToAddress(privateKey.PublicKey)

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		From: senderAddr,
		Data: txData,
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return err
	}
	defer client.Close()

	estimateGas, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return err
	}
	gasLimit := uint64(float64(estimateGas) * 1.3)

	nonce, err := client.PendingNonceAt(context.Background(), senderAddr)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return err
	}

	tx := types.NewTransaction(
		nonce,
		contractAddr,
		amount,
		gasLimit,
		gasPrice,
		txData,
	)

	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	return nil
}

func dispatcher() (common.Address, abi.ABI, error) {
	var dispatcherAddr common.Address
	var dispatcherABI abi.ABI

	config, err := loadConfig()
	if err != nil {
		return dispatcherAddr, dispatcherABI, err
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return dispatcherAddr, dispatcherABI, err
	}
	defer client.Close()

	dispatcherAddr = common.HexToAddress(config.DispatcherAddr)

	abiBytes, err := os.ReadFile(config.DispatcherABIFilePath)
	if err != nil {
		return dispatcherAddr, dispatcherABI, err
	}

	dispatcherABI, err = abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return dispatcherAddr, dispatcherABI, err
	}

	return dispatcherAddr, dispatcherABI, nil
}

func constructAirport() error {
	airportCode = uuid.New()

	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("constructAirport", airportCode)

	err = sendTransaction(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return err
}

func closeAirport() error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("closeAirport", airportCode)

	err = sendTransaction(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return nil
}

func destroyAirport() error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("destroyAirport", airportCode)

	err = sendTransaction(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return nil
}

func parseQuery(query string) (bool, string, error) {
	write := false
	var newQuery []string
	p := sql.NewParser(strings.NewReader(query))
	for {
		stmt, err := p.ParseStatement()
		if err == io.EOF {
			break
		}
		if err != nil {
			return write, strings.Join(newQuery, "; ") + ";", err
		}

		switch stmt.(type) {
		case *sql.InsertStatement, *sql.UpdateStatement, *sql.DeleteStatement:
			newQuery = append(newQuery, stmt.String()+" RETURNING *")
			write = true
		case *sql.SelectStatement:
			newQuery = append(newQuery, stmt.String())
		default:
			newQuery = append(newQuery, stmt.String())
			write = true
		}
	}

	return write, strings.Join(newQuery, "; ") + ";", nil
}

type Route struct {
	Client            *ethclient.Client
	Instance          chan struct{}
	CanWriteAccount   []common.Address
	DefaultPermission string
}

var routes = make(map[[16]byte]Route)

func checkNewFlightPlanEvent(destination [16]byte) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		fmt.Printf(">> Error: %s\n", err)
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
		fmt.Printf(">> Error: %s\n", err)
	}

	fmt.Println(">> New route has been launched.")
	for {
		select {
		case <-routes[destination].Instance:
			fmt.Println(">> The route has been terminated.")
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Destination [16]byte
				Query       string
				Operator    common.Address
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "FlightPlanSubmitted", vLog.Data)
			if err == nil {
				write, query, err := parseQuery(event.Query)
				if err != nil {
					fmt.Printf(">> Error: %s\n", err)
				}

				fmt.Printf(">> %v\n", routes[destination].CanWriteAccount)
				fmt.Printf(">> %s\n", event.Operator)

				if write && slices.Contains(routes[destination].CanWriteAccount, event.Operator) {
					fmt.Printf(">> Write Query: %s\n", query)
					// SQLを実行
					// RETURNINGのハッシュを計算
					// ハッシュの送信
				} else if !write {
					fmt.Printf(">> Read Query: %s\n", query)
				} else {
					fmt.Println(">> Permission error.")
				}
			}
		}
	}
}

func checkRouteTerminatedEvent(client *ethclient.Client, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		fmt.Printf(">> Error: %s\n", err)
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
		fmt.Printf(">> Error: %s\n", err)
	}

	fmt.Println(">> Your airport has been opened (1/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println(">> Your airport has been closed (1/2).")
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Origin      [16]byte
				Destination [16]byte
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "RouteTerminated", vLog.Data)
			if err == nil {
				fmt.Println(">> The route has been terminated.")

				close(routes[event.Destination].Instance)
				routes[event.Destination].Client.Close()

				go checkNewFlightPlanEvent(event.Destination)
			}
		}
	}
}

func checkNewRouteEvent(client *ethclient.Client, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		fmt.Printf(">> Error: %s\n", err)
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
		fmt.Printf(">> Error: %s\n", err)
	}

	fmt.Println(">> Your airport has been opened (2/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println(">> Your airport has been closed (2/2).")
			return nil
		case <-sub.Err():
			continue
		case vLog := <-logs:
			var event struct {
				Origin            [16]byte
				Destination       [16]byte
				CanWriteAccount   []common.Address
				DefaultPermission string
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "NewRouteLaunched", vLog.Data)
			if err == nil {
				fmt.Println(">> New route has been launched.")
				config, err := loadConfig()
				if err != nil {
					fmt.Printf(">> Error: %s\n", err)
				}

				client, err := ethclient.Dial(config.RPCEndpoint)
				if err != nil {
					fmt.Printf(">> Error: %s\n", err)
				}

				routes[event.Destination] = Route{
					client,
					make(chan struct{}),
					event.CanWriteAccount,
					event.DefaultPermission,
				}

				go checkNewFlightPlanEvent(event.Destination)
			}
		}
	}
}

func main() {
	if airportCode == [16]byte{} {
		err := constructAirport()
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(">> Hello! President!")
	fmt.Println(">> Here is airport console (Press Ctrl+C to exit).")
	fmt.Printf(">> Your airport code is %x.\n", airportCode)

	airportInstance := make(chan struct{})

	config, err := loadConfig()
	if err != nil {
		panic(err)
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		panic(err)
	}

	go checkNewRouteEvent(client, airportInstance)
	go checkRouteTerminatedEvent(client, airportInstance)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(airportInstance)
	client.Close()

	err = closeAirport()
	if err != nil {
		panic(err)
	}

	// マイグレートを実行

	err = destroyAirport()
	if err != nil {
		panic(err)
	}

	fmt.Println(">> Bye!")
}

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"database/sql"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	rql "github.com/rqlite/sql"
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

	nonce, err := client.NonceAt(context.Background(), senderAddr, nil)
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

type Query struct {
	Returning bool
	Query     string
}

func parseQuery(query string) (bool, []Query, error) {
	write := false
	var splittedQuery []Query
	p := rql.NewParser(strings.NewReader(query))
	for {
		stmt, err := p.ParseStatement()
		if err == io.EOF {
			break
		}
		if err != nil {
			return write, splittedQuery, err
		}

		switch stmt.(type) {
		case *rql.InsertStatement, *rql.UpdateStatement, *rql.DeleteStatement:
			splittedQuery = append(splittedQuery, Query{true, stmt.String() + " RETURNING *"})
			write = true
		case *rql.SelectStatement:
			splittedQuery = append(splittedQuery, Query{true, stmt.String()})
		default:
			splittedQuery = append(splittedQuery, Query{false, stmt.String()})
			write = true
		}
	}

	return write, splittedQuery, nil
}

type Route struct {
	Client            *ethclient.Client
	Instance          chan struct{}
	CanWriteAccount   []common.Address
	DefaultPermission string
}

var routes = make(map[[16]byte]Route)

func checkFlightPlanSubmittedEvent(destination [16]byte) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
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
	db, err := sql.Open("sqlite3", fmt.Sprintf("database/%x.db", destination))
	if err != nil {
		fmt.Printf("(%x)>> Error: %s\n", destination, err)
	}
	defer db.Close()
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
				Query       string
				Operator    common.Address
			}

			err := dispatcherABI.UnpackIntoInterface(&event, "FlightPlanSubmitted", vLog.Data)
			if err == nil {
				write, query, err := parseQuery(event.Query)
				if err != nil {
					fmt.Printf("(%x)>> Error: %s\n", destination, err)
				}

				if (write && (slices.Contains(routes[destination].CanWriteAccount, event.Operator) || routes[destination].DefaultPermission == "write")) || !write {
					for _, q := range query {
						fmt.Printf("(%x)>> Query: %s\n", destination, q.Query)
						if q.Returning {
							rows, err := db.Query(q.Query)
							if err != nil {
								fmt.Printf("(%x)>> Error: %s\n", destination, err)
								continue
							}
							defer rows.Close()

							cols, err := rows.Columns()
							if err != nil {
								fmt.Printf("(%x)>> Error: %s\n", destination, err)
								continue
							}

							result := make([]sql.RawBytes, len(cols))
							resultAddr := make([]interface{}, len(cols))
							for i := range result {
								resultAddr[i] = &result[i]
							}

							for rows.Next() {
								if err := rows.Scan(resultAddr...); err != nil {
									fmt.Printf("(%x)>> Error: %s\n", destination, err)
									continue
								}
							}

							hasher := sha256.New()
							for _, bytes := range result {
								if bytes == nil {
									hasher.Write([]byte{0xDE, 0xAD, 0xBE, 0xEF})
								} else {
									hasher.Write(bytes)
								}
								hasher.Write([]byte{0})
							}

							var hash [32]byte
							copy(hash[:], hasher.Sum(nil)[:])
							fmt.Printf("(%x)>> Hash: %x\n", destination, hash)
						} else {
							_, err := db.Exec(q.Query)
							if err != nil {
								fmt.Printf("(%x)>> Error: %s\n", destination, err)
							}
						}
					}
					// ハッシュの送信
				} else {
					fmt.Printf("(%x)>> Error: No proper permissions.\n", destination)
				}
			}
		}
	}
}

func checkRouteTerminatedEvent(client *ethclient.Client, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		fmt.Printf("(N/A)>> Error: %s\n", err)
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
		fmt.Printf("(N/A)>> Error: %s\n", err)
	}

	fmt.Println("(N/A)>> Your airport has been opened (1/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println("(N/A)>> Your airport has been closed (1/2).")
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
				fmt.Printf("(%x)>> The route has been terminated.", event.Destination)

				close(routes[event.Destination].Instance)
				routes[event.Destination].Client.Close()
			}
		}
	}
}

func checkNewRouteLaunchedEvent(client *ethclient.Client, airportInstance chan struct{}) error {
	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		fmt.Printf("(N/A)>> Error: %s\n", err)
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
		fmt.Printf("(N/A)>> Error: %s\n", err)
	}

	fmt.Println("(N/A)>> Your airport has been opened (2/2).")
	for {
		select {
		case <-airportInstance:
			fmt.Println("(N/A)>> Your airport has been closed (2/2).")
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
				config, err := loadConfig()
				if err != nil {
					fmt.Printf("(N/A)>> Error: %s\n", err)
				}

				client, err := ethclient.Dial(config.RPCEndpoint)
				if err != nil {
					fmt.Printf("(N/A)>> Error: %s\n", err)
				}

				routes[event.Destination] = Route{
					client,
					make(chan struct{}),
					event.CanWriteAccount,
					event.DefaultPermission,
				}

				go checkFlightPlanSubmittedEvent(event.Destination)
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

	fmt.Println("(N/A)>> Hello! President!")
	fmt.Println("(N/A)>> Here is airport console (Press Ctrl+C to exit).")
	fmt.Printf("(N/A)>> Your airport code is %x.\n", airportCode)

	airportInstance := make(chan struct{})

	config, err := loadConfig()
	if err != nil {
		panic(err)
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		panic(err)
	}

	go checkNewRouteLaunchedEvent(client, airportInstance)
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

	fmt.Println("(N/A)>> Bye!")
}

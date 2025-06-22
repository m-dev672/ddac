package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/rqlite/sql"
)

type Config struct {
	AirportCode           string `json:"airportCode"`
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
	senderAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: txData,
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return err
	}
	defer client.Close()

	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		return err
	}

	nonce, err := client.PendingNonceAt(context.Background(), senderAddress)
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
	config, err := loadConfig()
	if err != nil {
		return err
	}

	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		return err
	}

	config.AirportCode = uuid.New().String()
	txData, err := dispatcherABI.Pack("constructAirport", config.AirportCode)

	err = sendTransaction(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	err = saveConfig(config)
	if err != nil {
		return err
	}

	return nil
}

func destroyAirport() error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	dispatcherAddr, dispatcherABI, err := dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("destroyAirport", config.AirportCode)

	err = sendTransaction(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	config.AirportCode = ""
	err = saveConfig(config)
	if err != nil {
		return err
	}

	return nil
}

func parseQuery(query []string) ([]string, error) {
	newQuery := []string{}
	for _, q := range query {
		parser := sql.NewParser(strings.NewReader(q))
		stmt, err := parser.ParseStatement()
		if err != nil {
			return newQuery, err
		}

		switch stmt.(type) {
		case *sql.InsertStatement, *sql.UpdateStatement, *sql.DeleteStatement:
			newQuery = append(newQuery, strings.TrimRight(strings.TrimSpace(q), ";")+" RETURNING *;")
		}
	}

	return newQuery, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}

	if config.AirportCode == "" {
		err = constructAirport()
		if err != nil {
			panic(err)
		}

		config, err = loadConfig()
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("%s", config.AirportCode)

	err = destroyAirport()
	if err != nil {
		panic(err)
	}
}

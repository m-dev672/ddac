package contract

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/mdev672/ddac/config"
)

func Dispatcher() (common.Address, abi.ABI, error) {
	var dispatcherAddr common.Address
	var dispatcherABI abi.ABI

	config, err := config.Load()
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

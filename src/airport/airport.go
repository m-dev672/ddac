package airport

import (
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/mdev672/ddac/config"
	"github.com/mdev672/ddac/contract"
)

func Construct() (uuid.UUID, error) {
	airportCode := uuid.New()

	config, err := config.Load()
	if err != nil {
		return airportCode, err
	}

	var ipFixedBytes [4]byte
	ipBytes := net.ParseIP(config.IPAddr).To4()
	if ipBytes == nil {
		fmt.Printf("(General)>> Warning: \"%s\" is invalid IPv4 address.\n", config.IPAddr)
	} else {
		copy(ipFixedBytes[:], ipBytes)
	}

	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		return airportCode, err
	}

	txData, err := dispatcherABI.Pack("constructAirport", airportCode, ipFixedBytes)

	err = contract.SendTx(dispatcherAddr, txData, nil)
	if err != nil {
		return airportCode, err
	}

	return airportCode, err
}

func Close(airportCode uuid.UUID) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("closeAirport", airportCode)

	err = contract.SendTx(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return nil
}

func Destroy(airportCode uuid.UUID) error {
	dispatcherAddr, dispatcherABI, err := contract.Dispatcher()
	if err != nil {
		return err
	}

	txData, err := dispatcherABI.Pack("destroyAirport", airportCode)

	err = contract.SendTx(dispatcherAddr, txData, nil)
	if err != nil {
		return err
	}

	return nil
}

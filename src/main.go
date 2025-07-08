package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mdev672/ddac/airport"
	"github.com/mdev672/ddac/config"
	"github.com/mdev672/ddac/event"
)

func main() {
	var airportCode [16]byte
	var err error

	airportCode, err = airport.Construct()
	if err != nil {
		panic(err)
	}

	fmt.Println("(General)>> Hello! President!")
	fmt.Println("(General)>> Here is airport console (Press Ctrl+C to exit).")
	fmt.Printf("(General)>> Your airport code is %x.\n", airportCode)

	airportInstance := make(chan struct{})

	config, err := config.Load()
	if err != nil {
		panic(err)
	}

	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		panic(err)
	}

	go event.CheckNewRouteLaunchedEvent(client, airportCode, airportInstance)
	go event.CheckRouteTerminatedEvent(client, airportCode, airportInstance)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	close(airportInstance)
	client.Close()

	err = airport.Close(airportCode)
	if err != nil {
		panic(err)
	}

	// マイグレートを実行

	err = airport.Destroy(airportCode)
	if err != nil {
		panic(err)
	}

	fmt.Println("(General)>> Bye!")
}

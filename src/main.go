package main

import (
    "context"
    "fmt"
    "github.com/ethereum/go-ethereum/ethclient"
)

func main() {
    const url = "https://arbitrum-sepolia.drpc.org"  // url string
    
    rpcClient,err := ethclient.Dial(url)
    
    if err != nil {
        panic(err)
    }
    
    blockNumber, err := rpcClient.BlockNumber(context.Background())
    
    if err != nil {
        panic(err)
    }
    
    fmt.Println(blockNumber)
}
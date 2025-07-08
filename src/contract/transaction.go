package contract

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mdev672/ddac/config"
)

func SendTx(contractAddr common.Address, txData []byte, amount *big.Int) error {
	config, err := config.Load()
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

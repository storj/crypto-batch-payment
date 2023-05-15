package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/csv"
	"encoding/hex"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	//spenderKey, spenderKey := makeAccount()
	ownerKey, ownerAddress := makeAccount()

	writeFile("owner-key", []byte(ownerKey+"\n"), 0600)
	writeFile("owner-address", []byte(ownerAddress+"\n"), 0644)

	accountsBytes := new(bytes.Buffer)
	accountsCSV := csv.NewWriter(accountsBytes)
	for i := 0; i < 1000; i++ {
		key, address := makeAccount()
		accountsCSV.Write([]string{strconv.Itoa(i + 1), address, key})
	}
	accountsCSV.Flush()

	writeFile("accounts.csv", accountsBytes.Bytes(), 0600)
}

func writeFile(path string, data []byte, mode os.FileMode) {
	check(os.WriteFile(path, data, mode))
}

func makeAccount() (string, string) {
	rawKey := generateKey()

	key := hex.EncodeToString(crypto.FromECDSA(rawKey))
	address := crypto.PubkeyToAddress(rawKey.PublicKey).Hex()
	return key, address
}

func generateKey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	check(err)
	return key
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

package config

import (
	"bufio"
	"crypto/ecdsa"
	"errors"
	"io/fs"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/zeebo/errs"
)

var (
	zeroAddress common.Address
)

func loadSpenderKey(path string) (*ecdsa.PrivateKey, common.Address, error) {
	return loadETHKey(path, "spender_key_path")
}

func loadETHKey(path, which string) (*ecdsa.PrivateKey, common.Address, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, common.Address{}, errs.New("%s: %s not found\n", which, path)
		}
		return nil, common.Address{}, errs.New("unable to stat %s key: %v\n", which, err)
	}

	if (fi.Mode() & 0177) != 0 {
		return nil, common.Address{}, errs.New("%s mode %#o is too permissive (set to 0600)\n", path, fi.Mode())
	}

	key, err := crypto.LoadECDSA(path)
	if err != nil {
		return nil, common.Address{}, errs.New("unable to load %s key: %v\n", which, err)
	}
	return key, crypto.PubkeyToAddress(key.PublicKey), nil
}

func loadFirstLine(p string) (_ string, err error) {
	f, err := os.Open(p)
	if err != nil {
		return "", errs.Wrap(err)
	}
	defer func() {
		err = errs.Combine(err, f.Close())
	}()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", errs.Wrap(scanner.Err())
	}
	return scanner.Text(), nil
}

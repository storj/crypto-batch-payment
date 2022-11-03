package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

const cleanupAbi = false
const compileToken = false

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	if cleanupAbi {
		defer func() {
			err := combine(
				deleteGlob("*.abi"),
				deleteGlob("*.bin"),
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete: %v\n", err)
			}
		}()
	}

	if compileToken {
		pwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working dir: %v", err)
		}

		err = run("docker", "run", "--rm",
			"-v", pwd+":/s", "ethereum/solc:0.4.8",
			"--optimize", "--abi", "--bin",
			"-o", "/s", "/s/storj.sol")
		if err != nil {
			return fmt.Errorf("failed to build storj.sol: %v", err)
		}
	}

	err := abigen()
	if err != nil {
		return fmt.Errorf("failed to generate abi: %v", err)
	}

	return nil
}

func abigen() error {
	abi, err := os.ReadFile("CentrallyIssuedToken.abi")
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}
	bin, err := os.ReadFile("CentrallyIssuedToken.bin")
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	code, err := bind.Bind(
		[]string{"Token"},
		[]string{string(abi)},
		[]string{string(bin)},
		nil,
		"contract",
		bind.LangGo,
		nil,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to generate abi: %w", err)
	}

	err = os.WriteFile("token.go", []byte(code), 0600)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deleteGlob(pattern string) error {
	var errs errorlist
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, match := range matches {
		errs.Add(os.Remove(match))
	}
	return errs.Err()
}

type errorlist []error

func combine(errs ...error) error     { return errorlist(errs).Err() }
func (errs *errorlist) Add(err error) { *errs = append(*errs, err) }

func (errs errorlist) Err() error {
	nonZero := errs[:0]
	for _, err := range errs {
		if err == nil {
			continue
		}
		nonZero = append(nonZero, err)
	}

	if len(nonZero) == 0 {
		return nil
	}
	if len(nonZero) == 1 {
		return nonZero[0]
	}
	return fmt.Errorf("%v", nonZero)
}

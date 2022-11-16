package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"

	"storj.io/crypto-batch-payment/pkg/payouts"
)

type auditConfig struct {
	*rootConfig

	PayoutsCSV    string
	ReceiptsCSV   string
	PayerType     string
	ReceiptsForce bool
}

func newAuditCommand(rootConfig *rootConfig) *cobra.Command {
	config := &auditConfig{
		rootConfig: rootConfig,
	}
	cmd := &cobra.Command{
		Use:   "audit CSVPATH",
		Short: "Audits payouts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.PayoutsCSV = args[0]
			return checkCmd(doAudit(config))
		},
	}
	cmd.Flags().StringVarP(
		&config.ReceiptsCSV,
		"receipts", "r",
		"",
		"File to receive payout receipts CSV",
	)
	cmd.Flags().BoolVarP(
		&config.ReceiptsForce,
		"force-receipts", "f",
		false,
		"Force receipts",
	)
	cmd.Flags().StringVarP(
		&config.PayerType,
		"type", "",
		string(payouts.Eth),
		"Type of the payment (eth,sim)")
	return cmd
}

func doAudit(config *auditConfig) error {
	var sink auditSink

	var bad bool

	payerType, err := payouts.PayerTypeFromString(config.PayerType)
	if err != nil {
		return err
	}

	chainID, err := strconv.Atoi(config.ChainID)
	if err != nil {
		return err
	}

	fmt.Printf("Auditing %q...\n", config.PayoutsCSV)
	stats, err := payouts.Audit(config.Ctx, config.DataDir, config.PayoutsCSV, payerType, config.NodeAddress, chainID, sink, config.ReceiptsCSV, config.ReceiptsForce)
	if err != nil {
		return err
	}
	fmt.Println("Audit complete.")
	fmt.Printf("Total.......................: %d\n", stats.Total)
	fmt.Printf("Confirmed...................: %d\n", stats.Confirmed)
	fmt.Printf("False Confirmed.............: %d\n", stats.FalseConfirmed)
	fmt.Printf("Overpaid. ..................: %d\n", stats.Overpaid)
	fmt.Printf("Unstarted...................: %d\n", stats.Unstarted)
	fmt.Printf("Pending.....................: %d\n", stats.Pending)
	fmt.Printf("Failed......................: %d\n", stats.Failed)
	fmt.Printf("Dropped.....................: %d\n", stats.Dropped)
	fmt.Printf("Unknown.....................: %d\n", stats.Unknown)
	fmt.Printf("Mismatched..................: %d\n", stats.Mismatched)
	if stats.DoublePays > 0 {
		fmt.Println(aurora.Red(fmt.Sprintf("Double Pays.................: %d", stats.DoublePays)))
		fmt.Println(aurora.Red(fmt.Sprintf("Double Pay Amount (raw STORJ value): %s", stats.DoublePayStorj)))
		bad = true
	}

	if bad {
		fmt.Println()
		fmt.Println(aurora.Red("There were one or more problems with the payouts"))
	}
	return nil
}

type auditSink struct{}

func (auditSink) ReportStatus(format string, args ...interface{}) {
	fmt.Println(aurora.White(fmt.Sprintf(format, args...)))
}

func (auditSink) ReportWarn(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, aurora.Yellow(fmt.Sprintf(format, args...)))
}

func (auditSink) ReportError(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, aurora.Red(fmt.Sprintf(format, args...)))
}

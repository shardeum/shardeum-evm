//go:build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
)

// Markers for log sections
var (
	markerAnte      = regexp.MustCompile(`ANTE HANDLER`)
	markerTyped     = regexp.MustCompile(`CHAIN RECONSTRUCTED TYPED DATA`)
	markerHash      = regexp.MustCompile(`CHAIN CALCULATED HASH`)
	markerSignature = regexp.MustCompile(`FEE PAYER SIGNATURE`)
	markerVerify    = regexp.MustCompile(`ABOUT TO CALL secp256k1.VerifySignature`)
	markerResult    = regexp.MustCompile(`✅✅✅|❌❌❌`)
)

type TxLog struct {
	AnteLine      int
	TypedDataLine int
	HashLine      int
	SignatureLine int
	VerifyLine    int
	ResultLine    int
	Lines         []string
	Result        string
}

func main() {
	logPath := flag.String("log", "node.log", "path to node log file")
	latest := flag.Bool("latest", true, "only report on the most recent tx attempt")
	flag.Parse()

	logs, err := os.Open(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log: %v\n", err)
		os.Exit(1)
	}
	defer logs.Close()

	txAttempts := scanTxAttempts(logs)
	if len(txAttempts) == 0 {
		fmt.Println("No EIP-712 tx attempts found in logs.")
		return
	}

	if *latest {
		txAttempts = txAttempts[len(txAttempts)-1:]
	}

	for i, tx := range txAttempts {
		fmt.Printf("\n=== TX Attempt #%d ===\n", i+1)
		printTxReport(tx)
	}
}

func scanTxAttempts(logs *os.File) []TxLog {
	scanner := bufio.NewScanner(logs)
	var (
		lines []string
		txAttempts []TxLog
		current TxLog
		lineNum int
	)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		lineNum++

		if markerAnte.MatchString(line) {
			current = TxLog{AnteLine: lineNum, Lines: []string{line}}
		} else if markerTyped.MatchString(line) {
			current.TypedDataLine = lineNum
			current.Lines = append(current.Lines, line)
		} else if markerHash.MatchString(line) {
			current.HashLine = lineNum
			current.Lines = append(current.Lines, line)
		} else if markerSignature.MatchString(line) {
			current.SignatureLine = lineNum
			current.Lines = append(current.Lines, line)
		} else if markerVerify.MatchString(line) {
			current.VerifyLine = lineNum
			current.Lines = append(current.Lines, line)
		} else if markerResult.MatchString(line) {
			current.ResultLine = lineNum
			current.Result = line
			current.Lines = append(current.Lines, line)
			txAttempts = append(txAttempts, current)
			current = TxLog{}
		}
	}
	return txAttempts
}

func printTxReport(tx TxLog) {
	fmt.Printf("ANTE HANDLER at line %d\n", tx.AnteLine)
	fmt.Printf("TYPED DATA at line %d\n", tx.TypedDataLine)
	fmt.Printf("HASH at line %d\n", tx.HashLine)
	fmt.Printf("SIGNATURE at line %d\n", tx.SignatureLine)
	fmt.Printf("VERIFY INPUTS at line %d\n", tx.VerifyLine)
	fmt.Printf("RESULT at line %d: %s\n", tx.ResultLine, tx.Result)
	fmt.Println("--- Log Snippet ---")
	for _, l := range tx.Lines {
		fmt.Println(l)
	}
	fmt.Println("-------------------")
}

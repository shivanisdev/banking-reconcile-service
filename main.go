package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"sync"
	"time"
)

type ReconcillationInput struct {
	SystemTransactionsFile string   `json:"system_transactions_file"`
	BankStatementsFiles    []string `json:"bank_statements_files"`
	StartDate              string   `json:"start_date"`
	EndDate                string   `json:"end_date"`
}

type SystemTransaction struct {
	TrxID           string  `json:"trxID"`
	Amount          float64 `json:"amount"`
	Type            string  `json:"type"` // DEBIT or CREDIT
	TransactionTime string  `json:"transactionTime"`
}

type BankStatement struct {
	UniqueIdentifier string  `json:"unique_identifier"`
	Amount           float64 `json:"amount"`
	Date             string  `json:"date"`
	FilePath         string  `json:"file_path"`
}

type ReconcillationResult struct {
	TotalTransactions               int                                      `json:"total_transactions"`
	TotalMatchedTransactions        int                                      `json:"total_matched_transactions"`
	SysUnMatchedDetail              SysUnMatchedTransDetails                 `json:"sys_unmatched_transactions_detail"`
	BankUnMatchedTransactionsDetail map[string]ThisBankUnMatchedTransDetails `json:"bank_unmatched_transactions_detail"`
	TotalDiscrepanciesAmount        float64                                  `json:"total_discrepancies_amount"` // sum of absolute differences in amount between matched transactions
}

type SysUnMatchedTransDetails struct {
	TotalSystemTransMissingBank int                 `json:"total_system_trans_missing_bank"`
	SystemTransMissingBankList  []SystemTransaction `json:"system_trans_missing_bank_list"`
}

type ThisBankUnMatchedTransDetails struct {
	ThisBankUnMatchedTransCount int             `json:"this_bank_unmatched_trans_count"`
	ThisBankUnMatchedTransList  []BankStatement `json:"this_bank_unmatched_trans_list"`
}

func main() {
	// This is the entry point of the reconcillation service.
	// The service will be implemented in the future.
	// For now, it is just a placeholder.
	inp := ReconcillationInput{
		SystemTransactionsFile: "sample_files/system_data.csv",
		BankStatementsFiles:    []string{"sample_files/bank_A.csv", "sample_files/bank_B.csv", "sample_files/bank_C.csv"},
		StartDate:              "2025-06-10",
		EndDate:                "2025-07-10",
	}

	reconcillationService(inp)
}

func reconcillationService(inp ReconcillationInput) error {
	// parse system transactions file
	parsedStart, err := time.Parse(time.DateOnly, inp.StartDate)
	if err != nil {
		return fmt.Errorf("error parsing given start date: %v ", err)
	}

	parsedEnd, err := time.Parse(time.DateOnly, inp.EndDate)
	if err != nil {
		return fmt.Errorf("error parsing given end date: %v ", err)
	}

	sysTxs, err := readSystemTransactions(inp.SystemTransactionsFile, parsedStart, parsedEnd)
	if err != nil {
		return fmt.Errorf("error readSystemTransactions %v ", err)
	}

	fmt.Printf("System Transactions in dateRange %+v \n", sysTxs)

	// parse bank statements files in parallel

	var wg sync.WaitGroup
	bankTxs := make(chan []BankStatement, len(inp.BankStatementsFiles))

	for _, path := range inp.BankStatementsFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			stmts, err := readBankStatements(path, parsedStart, parsedEnd)
			if err == nil {
				bankTxs <- stmts
			}
		}(path)
	}

	wg.Wait()
	close(bankTxs)

	// Collect all statements from channel into one slice
	var combinedBankTxs []BankStatement
	for bankList := range bankTxs {
		combinedBankTxs = append(combinedBankTxs, bankList...)
	}

	fmt.Printf("Combined Bank Transactions in dateRange from all files %+v \n", bankTxs)

	// till now we have already filterred for the required dates

	result, err := Reconcile(sysTxs, combinedBankTxs)
	if err != nil {
		return fmt.Errorf("error in reconcile %v", err)
	}

	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling result to JSON: %v", err)
	}

	fmt.Println("\n🧾 Final Reconciliation Result (JSON):")
	fmt.Println(string(jsonOutput))

	return nil
}

func readSystemTransactions(path string, start, end time.Time) ([]SystemTransaction, error) {
	fmt.Printf("readSystemTransactions file %v \n", path)
	fmt.Printf("readSystemTransactions start %v \n", start)
	fmt.Printf("readSystemTransactions end %v \n", end)
	// This function will read system transactions from a CSV and will filter for given dates

	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Error opening file %s , Error %v ", path, err))
	}
	defer file.Close()

	// read as stream
	reader := csv.NewReader(file)
	reader.Read() // skip header

	var sysTxn []SystemTransaction
	for {
		row, err := reader.Read()
		if err == io.EOF {
			fmt.Println("finish reading end")
			break
		} else if err != nil {
			return nil, err
		}

		amount, _ := strconv.ParseFloat(row[1], 64)

		// to have common format for system and bank statement dates
		timeVal, err := time.Parse(time.DateTime, row[3])
		if err != nil {
			fmt.Println("Error parsing system transaction time:", err)
			return nil, err
		}

		sysTxn = append(sysTxn, SystemTransaction{
			TrxID:           row[0],
			Amount:          amount,
			Type:            row[2],
			TransactionTime: timeVal.Format(time.DateOnly),
		})

	}

	return sysTxn, nil
}

func readBankStatements(path string, start, end time.Time) ([]BankStatement, error) {
	fmt.Printf("readSystemTransactions file %+v \n", path)
	fmt.Printf("readSystemTransactions start %v \n", start)
	fmt.Printf("readSystemTransactions end %v \n", end)
	// This function will read bank statements from CSV files. according to the given dates

	var bnkTxn []BankStatement

	file, err := os.Open(path)
	if err != nil {
		panic(fmt.Sprintf("Error opening file %s , Error %v ", path, err))
	}
	defer file.Close()

	// read as stream
	reader := csv.NewReader(file)
	reader.Read() // skip header

	for {
		row, err := reader.Read()
		if err == io.EOF {
			fmt.Println("finish reading end")
			break
		} else if err != nil {
			return nil, err
		}

		amount, _ := strconv.ParseFloat(row[1], 64)

		// to have common format for system and bank statement dates
		timeVal, err := time.Parse(time.DateOnly, row[2])
		if err != nil {
			fmt.Println("Error parsing system transaction time:", err)
			return nil, err
		}

		bnkTxn = append(bnkTxn, BankStatement{
			UniqueIdentifier: row[0],
			Amount:           amount,
			Date:             timeVal.Format(time.DateOnly),
			FilePath:         path,
		})

	}

	return bnkTxn, nil
}

// This function will reconcile system transactions with bank statements.
// It will compare the transactions and identify discrepancies.
func Reconcile(sysTxs []SystemTransaction, bankTxs []BankStatement) (ReconcillationResult, error) {
	var reconcillationResult ReconcillationResult
	reconcillationResult.TotalTransactions = len(sysTxs) + len(bankTxs)

	// - Iterate through system transactions
	alreadyMatch := make([]bool, len(bankTxs))
	for _, sysTx := range sysTxs {
		match := false
		for i, bankTx := range bankTxs {
			// skip bankTxn if already a match track by key
			if alreadyMatch[i] {
				continue
			}

			if sysTx.TransactionTime == bankTx.Date && math.Abs(sysTx.Amount-bankTx.Amount) < 0.50 {
				amtDiff := math.Abs(sysTx.Amount - bankTx.Amount)
				alreadyMatch[i] = true
				match = true
				reconcillationResult.TotalMatchedTransactions++

				// If amount difference is within discrepancy threshold, consider it discrepant
				if amtDiff > 0.01 && amtDiff < 5.0 {
					reconcillationResult.TotalDiscrepanciesAmount += amtDiff
				}

				break
			}

		}

		// sysTxn scaned against all bankTns match not found
		if !match {
			reconcillationResult.SysUnMatchedDetail.TotalSystemTransMissingBank++
			reconcillationResult.SysUnMatchedDetail.SystemTransMissingBankList = append(
				reconcillationResult.SysUnMatchedDetail.SystemTransMissingBankList,
				sysTx,
			)
		}
	}

	// All the bankTxn with alreadyMatch false are unmatched bnkTxns
	missingBankTans := make(map[string][]BankStatement)
	for i, alreadyMatchFlag := range alreadyMatch {
		if !alreadyMatchFlag {
			// this is missing in bankTxns slice
			pathKey := bankTxs[i] // missing bank txn for this path/file
			missingBankTans[pathKey.FilePath] = append(missingBankTans[pathKey.FilePath], bankTxs[i])

		}
	}

	fmt.Printf("tota bnk files %v \n", len(missingBankTans))

	reconcillationResult.BankUnMatchedTransactionsDetail = make(map[string]ThisBankUnMatchedTransDetails)
	for path, eachBankTnxs := range missingBankTans {
		reconcillationResult.BankUnMatchedTransactionsDetail[path] = ThisBankUnMatchedTransDetails{
			ThisBankUnMatchedTransCount: len(eachBankTnxs),
			ThisBankUnMatchedTransList:  eachBankTnxs,
		}
	}

	return reconcillationResult, nil
}

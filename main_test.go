package main

import (
	"testing"
)

func TestReconcile_MatchedAndDiscrepant(t *testing.T) {
	sysTxs := []SystemTransaction{
		{TrxID: "TXN001", Amount: 1500.50, Type: "DEBIT", TransactionTime: "2025-07-08"},
		{TrxID: "TXN002", Amount: 250.75, Type: "CREDIT", TransactionTime: "2025-07-10"},
		{TrxID: "TXN003", Amount: 100.00, Type: "CREDIT", TransactionTime: "2025-07-08"},
	}

	bankTxs := []BankStatement{
		{UniqueIdentifier: "BS001", Amount: 1500.20, Date: "2025-07-08", FilePath: "bank_A.csv"}, // discrepant
		{UniqueIdentifier: "BS002", Amount: 250.75, Date: "2025-07-10", FilePath: "bank_A.csv"},  // exact match
		{UniqueIdentifier: "BS003", Amount: 105.00, Date: "2025-07-08", FilePath: "bank_A.csv"},  // unmatched
	}

	expectedMatched := 2

	result, err := Reconcile(sysTxs, bankTxs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalMatchedTransactions != expectedMatched {
		t.Errorf("expected %d matched transactions, got %d", expectedMatched, result.TotalMatchedTransactions)
	}

	if result.SysUnMatchedDetail.TotalSystemTransMissingBank != 1 {
		t.Errorf("expected 1 unmatched system transaction, got %d", result.SysUnMatchedDetail.TotalSystemTransMissingBank)
	}

	if result.BankUnMatchedTransactionsDetail["bank_A.csv"].ThisBankUnMatchedTransCount != 1 {
		t.Errorf("expected 1 unmatched bank transaction for bank_A.csv, got %d", result.BankUnMatchedTransactionsDetail["bank_A.csv"].ThisBankUnMatchedTransCount)
	}
}

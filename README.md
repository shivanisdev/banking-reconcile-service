Reconciliation service that identifies unmatched and discrepant transactions between internal data
(system transactions) and external data (bank statements) for Amartha.

### Problem Statement:
This Service manages multiple bank accounts and requires a service to reconcile transactions occurring within their system against corresponding transactions reflected in bank statements. This process helps identify errors, discrepancies, and missing transactions.

### Data Model:

#### Transaction:
- trxID : Unique identifier for the transaction (string)
- amount : Transaction amount (decimal)
- type : Transaction type (enum: DEBIT, CREDIT)
- transactionTime : Date and time of the transaction (datetime)

Data Model in Golang 
```go
type Transaction struct {
    TrxID          string    `json:"trxID"`
    Amount         float64   `json:"amount"`
    Type           string    `json:"type"` // DEBIT or CREDIT
    TransactionTime time.Time `json:"transactionTime"`
}
```

#### Bank Statement:
- unique_identifier : Unique identifier for the transaction in the bank statement (string) (varies by bank, not necessarily equivalent to trxID)
- amount : Transaction amount (decimal) (can be negative for debits)
- date : Date of the transaction (date)


```go
type BankStatement struct {
    UniqueIdentifier string    `json:"unique_identifier"`
    Amount           float64   `json:"amount"`
    Date             time.Time `json:"date"`
}   
```

### Assumptions:
- Both system transactions and bank statements are provided as separate CSV files.
- Discrepancies only occur in amount.


### Functionality:

#### Input:
The service accepts the following input parameters:
- System transaction CSV file path
- Bank statement CSV file path (can handle multiple files from different banks)
- Start date for reconciliation timeframe (date)
- End date for reconciliation timeframe (date)

#### Point to Note:
The service performs the reconciliation process by comparing transactions within the specified timeframe across system and bank statement data.
> As (unique_identifier) not necessarily equivalent to trxID we will use a combination of amount and date to match transactions.
> Another point to note is System transaction time is in format datetime `YYYY-MM-DD HH:MM:SS` and Bank statement date is in format date `YYYY-MM-DD`.

#### Expected Output:
The service outputs a reconciliation summary containing:

- Total number of transactions processed
- Total number of matched transactions
- Total number of unmatched transactions
- Details of unmatched transactions:
  - System transaction details if missing in bank statement(s)
  - Bank statement details if missing in system transactions (grouped by bank)
- Total discrepancies (sum of absolute differences in amount between matched transactions)

```
{
  "total_transactions": 20,
  "total_matched_transactions": 6,
  "sys_unmatched_transactions_detail": {
    "total_system_trans_missing_bank": 2,
    "system_trans_missing_bank_list": [
      {
        "trxID": "TXN006",
        "amount": 1500.5,
        "type": "DEBIT",
        "transactionTime": "2025-06-04"
      },
      {
        "trxID": "TXN001",
        "amount": 1500.5,
        "type": "DEBIT",
        "transactionTime": "2025-07-12"
      }
    ]
  },
  "bank_unmatched_transactions_detail": {
    "sample_files/bank_A.csv": {
      "this_bank_unmatched_trans_count": 1,
      "this_bank_unmatched_trans_list": [
        {
          "unique_identifier": "BS008",
          "amount": 500,
          "date": "2025-07-12",
          "file_path": "sample_files/bank_A.csv"
        }
      ]
    }
  },
  "Total_discrepancies_amount": 0
}
```

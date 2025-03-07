package blnk

import (
	"fmt"
	"math"

	"github.com/jerry-enebeli/blnk/internal/notification"

	"github.com/jerry-enebeli/blnk/model"
)

// Max constants for normalization
const (
	maxChangeFrequency   = 10.0
	maxTransactionAmount = 100000
	maxBalance           = 500000
	maxCreditBalance     = 1000000
	maxDebitBalance      = 700000
)

func NewBalanceTracker() *model.BalanceTracker {
	return &model.BalanceTracker{
		Balances:    make(map[string]*model.Balance),
		Frequencies: make(map[string]int),
	}
}

func (l Blnk) checkBalanceMonitors(updatedBalance *model.Balance) {
	// Fetch monitors for this balance using datasource
	monitors, _ := l.datasource.GetBalanceMonitors(updatedBalance.BalanceID)
	// Check each monitor's condition
	for _, monitor := range monitors {
		if monitor.CheckCondition(updatedBalance) {
			fmt.Printf("Condition met for balance: %s\n", monitor.MonitorID)
			data := map[string]interface{}{
				"event": "balance.monitor",
				"data":  monitor,
			}
			notification.WebhookNotification(data)
		}
	}

}

func (l Blnk) CreateBalance(balance model.Balance) (model.Balance, error) {
	return l.datasource.CreateBalance(balance)
}

func (l Blnk) GetBalanceByID(id string, include []string) (*model.Balance, error) {
	return l.datasource.GetBalanceByID(id, include)
}

func (l Blnk) GetAllBalances() ([]model.Balance, error) {
	return l.datasource.GetAllBalances()
}

func (l Blnk) CreateMonitor(monitor model.BalanceMonitor) (model.BalanceMonitor, error) {
	return l.datasource.CreateMonitor(monitor)
}

func (l Blnk) GetMonitorByID(id string) (*model.BalanceMonitor, error) {
	return l.datasource.GetMonitorByID(id)
}

func (l Blnk) GetAllMonitors() ([]model.BalanceMonitor, error) {
	return l.datasource.GetAllMonitors()
}

func (l Blnk) GetBalanceMonitors(balanceId string) ([]model.BalanceMonitor, error) {
	return l.datasource.GetBalanceMonitors(balanceId)
}

func (l Blnk) UpdateMonitor(monitor *model.BalanceMonitor) error {
	return l.datasource.UpdateMonitor(monitor)
}

func (l Blnk) DeleteMonitor(id string) error {
	return l.datasource.DeleteMonitor(id)
}

// ApplyFraudScore updates the balance and computes the fraud score
func (l Blnk) ApplyFraudScore(newBalance *model.Balance, amount int64) float64 {
	bt := l.bt
	bt.Mutex.Lock()
	defer bt.Mutex.Unlock()

	oldBalance, exists := bt.Balances[newBalance.BalanceID]

	if exists && (oldBalance.CreditBalance != newBalance.CreditBalance || oldBalance.DebitBalance != newBalance.DebitBalance) {
		bt.Frequencies[newBalance.BalanceID]++
	}

	bt.Balances[newBalance.BalanceID] = newBalance

	changeFrequency := float64(bt.Frequencies[newBalance.BalanceID])
	transactionAmount := float64(amount)
	currentBalance := float64(newBalance.Balance)
	creditBalance := float64(newBalance.CreditBalance)
	debitBalance := float64(newBalance.DebitBalance)

	return ComputeFraudScore(changeFrequency, transactionAmount, currentBalance, creditBalance, debitBalance)
}

// Normalize function to scale values between 0 and 1
func normalize(value, min, max float64) float64 {
	return (value - min) / (max - min)
}

// ComputeFraudScore computes a fraud score based on various parameters
func ComputeFraudScore(changeFrequency, transactionAmount, currentBalance, creditBalance, debitBalance float64) float64 {
	normalizedChangeFrequency := normalize(changeFrequency, 0, maxChangeFrequency)
	normalizedTransactionAmount := normalize(transactionAmount, 0, maxTransactionAmount)
	normalizedCurrentBalance := normalize(currentBalance, 0, maxBalance)
	normalizedCreditBalance := normalize(creditBalance, 0, maxCreditBalance)
	normalizedDebitBalance := normalize(debitBalance, 0, maxDebitBalance)

	//ensure they sum up to 1
	weightChangeFrequency := 0.3
	weightTransactionAmount := 0.3
	weightCurrentBalance := 0.1
	weightCreditBalance := 0.1
	weightDebitBalance := 0.2

	fraudScore := normalizedChangeFrequency*weightChangeFrequency +
		normalizedTransactionAmount*weightTransactionAmount +
		normalizedCurrentBalance*weightCurrentBalance +
		normalizedCreditBalance*weightCreditBalance +
		normalizedDebitBalance*weightDebitBalance

	return math.Max(0, math.Min(fraudScore, 1))
}

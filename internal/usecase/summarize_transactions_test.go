package usecase

import (
	"strings"
	"testing"
	"time"

	"whatsup-bot/internal/domain"
)

func summaryTx(day int, typ domain.TransactionType, amount int64, cat domain.Category, desc string) *domain.Transaction {
	return &domain.Transaction{
		Type: typ, Amount: amount, Category: cat, Description: desc,
		TransactionDate: time.Date(2026, 8, day, 0, 0, 0, 0, time.UTC),
	}
}

func TestRenderSummary(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	t.Run("expense and income", func(t *testing.T) {
		got := renderSummary(start, end, []*domain.Transaction{
			summaryTx(1, domain.Outcome, 25000, domain.CategoryFood, "roti"),
			summaryTx(1, domain.Outcome, 27000, domain.CategoryFood, "kopi"),
			summaryTx(2, domain.Outcome, 50000, domain.CategoryUtilities, "pulsa"),
			summaryTx(2, domain.Income, 500000, domain.CategoryInvestment, "deposito"),
		})
		for _, want := range []string{
			"*SUMMARY* [1/8/26 - 31/8/26]",
			"- Expense: *IDR 102,000* from *3 transactions*",
			"- Income: *IDR 500,000* from *1 transaction*",
			"most expense on *Food*, and income from *Investment*, and biggest expense is *Pulsa IDR 50,000*, and biggest income is *Deposito IDR 500,000*",
			"*Detail of 4 transactions*",
			"- `1/8: -IDR 25,000 Roti`",
			"- `2/8: +IDR 500,000 Deposito`",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
	})

	t.Run("expense only", func(t *testing.T) {
		got := renderSummary(start, end, []*domain.Transaction{summaryTx(1, domain.Outcome, 25000, domain.CategoryFood, "roti")})
		for _, want := range []string{"- Income: *none*", "_With most expense on *Food*, and biggest expense is *Roti IDR 25,000*_"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
		if strings.Contains(got, "income from") {
			t.Errorf("unexpected income insight in:\n%s", got)
		}
	})

	t.Run("income only", func(t *testing.T) {
		got := renderSummary(start, end, []*domain.Transaction{summaryTx(3, domain.Income, 900000, domain.CategorySalary, "gaji")})
		for _, want := range []string{"- Expense: *none*", "_With income from *Salary*, and biggest income is *Gaji IDR 900,000*_"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
	})

	t.Run("nothing to count", func(t *testing.T) {
		want := "No transactions found for [1/8/26 - 31/8/26]."
		for _, txs := range [][]*domain.Transaction{nil, {summaryTx(1, domain.Outcome, 0, domain.CategoryFood, "donut")}} {
			if got := renderSummary(start, end, txs); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		}
	})
}

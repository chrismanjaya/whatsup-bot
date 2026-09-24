package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"whatsup-bot/internal/domain"
)

func TestListByUserBetween(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO users (jid, name, register_at) VALUES ('a@s.whatsapp.net', 'a', ?)`, time.Now()); err != nil {
		t.Fatal(err)
	}
	repo := NewTransactionRepo(db)
	jkt := time.FixedZone("WIB", 7*3600)
	for _, d := range []string{"2026-07-31", "2026-08-01", "2026-08-01", "2026-08-15", "2026-09-01"} {
		day, _ := time.ParseInLocation("2006-01-02", d, jkt)
		if err := repo.Save(ctx, &domain.Transaction{UserID: 1, Description: d, Type: domain.Outcome, Amount: 1, Category: domain.CategoryOther, TransactionDate: day, CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}

	from, _ := time.ParseInLocation("2006-01-02", "2026-08-01", jkt)
	to, _ := time.ParseInLocation("2006-01-02", "2026-09-01", jkt)
	txs, total, err := repo.ListByUserBetween(ctx, 1, from, to, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(txs) != 2 || txs[0].Description != "2026-08-01" || txs[1].Description != "2026-08-15" {
		t.Fatalf("total=%d txs=%v", total, txs)
	}

	one, total, _ := repo.ListByUserBetween(ctx, 1, from, from.AddDate(0, 0, 1), 10, 0)
	if total != 2 || len(one) != 2 {
		t.Fatalf("single day: total=%d len=%d", total, len(one))
	}
}

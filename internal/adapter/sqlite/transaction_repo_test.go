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

func TestLastCreatedAt(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO users (jid, name, register_at) VALUES ('a@s.whatsapp.net', 'a', ?), ('b@s.whatsapp.net', 'b', ?)`, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	repo := NewTransactionRepo(db)

	if _, ok, err := repo.LastCreatedAt(ctx, 1); err != nil || ok {
		t.Fatalf("no transactions: ok=%v err=%v", ok, err)
	}

	jkt := time.FixedZone("WIB", 7*3600)
	older := time.Date(2026, 10, 4, 23, 50, 0, 0, jkt)
	newer := time.Date(2026, 10, 5, 0, 10, 0, 0, time.UTC) // stored with a different offset
	for _, c := range []struct {
		user int64
		at   time.Time
	}{{1, older}, {1, newer}, {2, older.Add(-time.Hour)}} {
		if err := repo.Save(ctx, &domain.Transaction{UserID: c.user, Description: "x", Type: domain.Outcome, Amount: 1, Category: domain.CategoryOther, TransactionDate: c.at, CreatedAt: c.at}); err != nil {
			t.Fatal(err)
		}
	}

	got, ok, err := repo.LastCreatedAt(ctx, 1)
	if err != nil || !ok || !got.Equal(newer) {
		t.Fatalf("LastCreatedAt(1) = %v, %v, %v; want %v", got, ok, err, newer)
	}
	got, ok, _ = repo.LastCreatedAt(ctx, 2)
	if !ok || !got.Equal(older.Add(-time.Hour)) {
		t.Fatalf("LastCreatedAt(2) = %v; want user 2's own transaction", got)
	}
}

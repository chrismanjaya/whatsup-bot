package sqlite

import (
	"context"
	"database/sql"
	"time"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/utils"
)

// The date range is stored as "YYYY-MM-DD" text: it is a calendar range, not
// an instant, so no timezone needs to survive the round trip.
const queryPageDateFmt = "2006-01-02"

type QueryPageRepo struct {
	db  *sql.DB
	loc *time.Location
}

func NewQueryPageRepo(db *sql.DB) *QueryPageRepo {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.UTC
	}
	return &QueryPageRepo{db: db, loc: loc}
}

func (r *QueryPageRepo) Save(ctx context.Context, p *domain.QueryPage) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO query_pages (user_id, from_date, to_date, page, created_at) VALUES (?, ?, ?, ?, ?)`,
		p.UserID, p.From.Format(queryPageDateFmt), p.To.Format(queryPageDateFmt), p.Page, p.CreatedAt,
	)
	if err != nil {
		return utils.Wrap(err, "insert query page failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return utils.Wrap(err, "get last insert id failed")
	}
	p.ID = id
	return nil
}

func (r *QueryPageRepo) FindByWAMessageID(ctx context.Context, waMessageID string) (*domain.QueryPage, error) {
	var p domain.QueryPage
	var from, to string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, from_date, to_date, page, created_at FROM query_pages WHERE wa_message_id = ?`,
		waMessageID,
	).Scan(&p.ID, &p.UserID, &from, &to, &p.Page, &p.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.Wrap(err, "find query page by wa message id")
	}
	if p.From, err = time.ParseInLocation(queryPageDateFmt, from, r.loc); err != nil {
		return nil, utils.Wrap(err, "parse query page from date")
	}
	if p.To, err = time.ParseInLocation(queryPageDateFmt, to, r.loc); err != nil {
		return nil, utils.Wrap(err, "parse query page to date")
	}
	p.WAMessageID = waMessageID
	return &p, nil
}

func (r *QueryPageRepo) SetWAMessageID(ctx context.Context, id int64, waMessageID string) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE query_pages SET wa_message_id = ? WHERE id = ?`, waMessageID, id); err != nil {
		return utils.Wrap(err, "set query page wa message id failed")
	}
	return nil
}

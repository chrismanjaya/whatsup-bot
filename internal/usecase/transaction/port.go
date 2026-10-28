package transaction

import (
	"context"
	"time"
	"whatsupbot/internal/domain"
)

type (
	TransactionRepository interface {
		Create(ctx context.Context, transaction *domain.Transaction) error
		Update(ctx context.Context, transaction *domain.Transaction) error
		Delete(ctx context.Context, transactionID int64) error

		FindByGroup(ctx context.Context, groupID int64, since time.Time) ([]*domain.Transaction, error)
		FindByMessage(ctx context.Context, messageID string) (*domain.Transaction, error)
		FindByUser(ctx context.Context, userID int64, since time.Time) ([]*domain.Transaction, error)

		GetListByPeriod(ctx context.Context, userID int64, from, to time.Time, transactionType domain.TransactionType, limit, offset int) ([]*domain.Transaction, int, error)
		GetLastTransactionTime(ctx context.Context, userID int64) (*time.Time, error)

		SetMessage(ctx context.Context, transactionID int64, messageID string) error
	}

	UserRepository interface {
		Create(ctx context.Context, u *domain.User) error
		FindByID(ctx context.Context, userID int64) (*domain.User, error)
		FindByJID(ctx context.Context, jid string) (*domain.User, error)
	}

	GroupRepository interface {
		Create(ctx context.Context, group *domain.Group) error
		FindByID(ctx context.Context, groupID int64) (*domain.Group, error)
		FindByJID(ctx context.Context, jid string) (*domain.Group, error)

		AddMember(ctx context.Context, groupID, userID int64) error
		GetListMember(ctx context.Context, groupID int64) ([]*domain.User, error)
	}
)

type GeminiClient interface {
	ParseRecordMessage(ctx context.Context, rawMessage string) (*domain.ParsedMessage, error)
	ParseAmendMessage(ctx context.Context, rawMessage string, transaction *domain.Transaction) (*domain.ParsedMessage, error)
}

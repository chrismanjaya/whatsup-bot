// Package message holds every user-facing string the bot sends, so wording
// can be changed in one place. It has no dependencies; usecases and adapters
// import it the same way they import internal/constant.
package message

// Error replies, picked by errors.Is() classification in the whatsapp router.
const (
	ErrGeneric            = "Something went wrong. Please try again."
	ErrAlreadyRegistered  = "You're already registered."
	ErrNotRegistered      = "Please register first: register <name> <email>"
	ErrInvalidRequest     = "I can only help track income and expenses."
	ErrServiceUnavailable = "The service is a bit busy right now, please try sending that again in a moment."

	// ErrWithCode wraps one of the above with the numeric error code (%d) so
	// a user can report it without exposing any internal detail.
	ErrWithCode = "%s (Error code: %d)"
)

// Registration.
const (
	RegisterUsage = "Usage: register <name> <email>"
	Welcome       = "Welcome, %s! You can now log your income/expenses." // %s: name
)

// Amend (reply to a confirmation).
const (
	NotTransactionOwner = "Sorry, you can only update or delete your own transactions."
	AmendUnclear        = "I couldn't tell what to change. Reply with the new amount or say \"delete\" to remove it."
)

// Transaction listing and paging.
const (
	QueryDateUnclear = "I couldn't tell which date or period you mean?"
	QueryNoResults   = "No transactions found for %s." // %s: date or period

	PageHeader          = "*PAGE"
	PageSummaryTemplate = PageHeader + ` [[page]]/[[total_pages]]*
- Period: *[[period]]*
- Showing: *[[from_n]]-[[to_n]] of [[total]]* ([[remaining]] more)

_*Reply to this message to see more_`

	PageNotOwner   = "Sorry, you can only page through your own transactions."
	PageUnclear    = "I couldn't tell which page you want?"
	PageOutOfRange = "Page %d doesn't exist, there are %d page(s) in total." // requested, total pages
)

// Split.
const (
	SplitNoMembers = "No members found for this group."
	SplitHeader    = "This month's household expenses:\n"
	SplitMember    = "%s: Rp%d\n"                                // name, spent
	SplitTotal     = "Total: Rp%d (Rp%d each for equal split)\n" // total, share
	SplitOwes      = "%s owes Rp%d to balance out\n"             // name, owed
)

// Transaction confirmations. The [[placeholder]] tokens are filled in by
// strings.NewReplacer in the usecase.
const (
	TypeExpense = "EXPENSE"
	TypeIncome  = "INCOME"
	Currency    = "IDR"

	TransactionReplyTemplate = `*[[transaction_type_str]]*
- Desc: *[[transaction_description]]*
- Category: *[[transaction_category]]*
- Amount: *[[transaction_amount_formatted]]*
- Date: *[[transaction_date_formatted]]*

_*Reply to this message to update or delete this transaction_`

	AmountPromptHeader   = "*AMOUNT NEEDED*"
	AmountPromptTemplate = AmountPromptHeader + `
- Desc: *[[transaction_description]]*
- Category: *[[transaction_category]]*
- Date: *[[transaction_date_formatted]]*

How much was it?`

	DeletedHeader        = "*DELETED*"
	DeletedReplyTemplate = DeletedHeader + `
- Desc: *[[transaction_description]]*
- Amount: *[[transaction_amount_formatted]]*`
)

// BotReplyPrefixes are the first words of every reply the bot sends. The
// whatsapp router ignores incoming messages starting with one of these, as a
// fallback self-echo guard behind IsFromMe. Add an entry whenever a new
// reply template's first line changes.
var BotReplyPrefixes = []string{
	"*" + TypeExpense + "*",
	"*" + TypeIncome + "*",
	AmountPromptHeader,
	DeletedHeader,
	PageHeader,
	"No transactions found",
	"Welcome",
	ErrAlreadyRegistered,
}

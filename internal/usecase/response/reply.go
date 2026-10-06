// Package response holds the output types shared by every usecase. It
// imports no other usecase package, so any of them can use it without an
// import cycle.
package response

import "whatsup-bot/internal/domain"

// Reply is one outgoing WhatsApp message. Tx or Page, when set, tell the
// sender to remember the sent message's ID against that row so a later reply
// to it can be resolved.
type Reply struct {
	Text string
	// Typing makes the handler show "typing..." for a moment before
	// sending this reply, so Frankie's chatty messages (like a quip)
	// arrive like a person typing them. Data messages leave it false.
	Typing bool
	Tx     *domain.Transaction
	Page   *domain.QueryPage
}

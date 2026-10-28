package persona

import "whatsupbot/internal/constant"

type MessagePool struct {
	Messages []string
	Emoji    string
}

var (
	ErrorMessagePools = map[int]MessagePool{
		// common errors
		constant.ErrInvalidRequest.Code: {
			Emoji: "🧪",
			Messages: []string{
				"Hmm... Frankie only understands money, bos. Try something like: buy coffee 5K today",
				"*scratches stitched head* ...Income? Expense? Tell me one of those. Example: buy coffee 5K today",
				"I was stitched together for one job: tracking your money. Try: buy coffee 5K today",
				"Hi... bos! That one went over my bolts. Send me a transaction, like: buy coffee 5K today",
				"Frankie's brain is small and full of numbers. Give me an expense or income, like: pbuy coffee 5K today",
				"Not sure what to do with that. I log income and expenses, e.g. buy coffee 5K today",
			},
		},
		constant.ErrInvalidGroup.Code: {
			Emoji: "👀",
			Messages: []string{
				"Wait, bos... that transaction lives in a different chat. Change it from the chat where it was logged.",
				"*flips through the big book* ...wrong chapter! That entry belongs to another chat, edit it there.",
				"Sorry, bos! That one was recorded in a different chat. Frankie keeps each chat's pages separate.",
				"Frankie stitched that entry into another chat's ledger. Head back there to update or delete it.",
				"Group money stays in its group, personal money stays personal. Change that transaction where it was recorded.",
				"My lab, my rules: every chat has its own pages. That transaction isn't from this one.",
			},
		},
		constant.ErrInvalidUser.Code: {
			Emoji: "🌩️",
			Messages: []string{
				"Hold on, bos... that one isn't yours. Frankie only lets the person who logged it change it.",
				"*sniffs the page* ...wrong handwriting! Only whoever recorded this transaction can update or delete it.",
				"Sorry, bos! That entry belongs to someone else's ledger. You can only change your own transactions.",
				"Frankie guards every page of the big book. That transaction isn't yours to edit, bos.",
				"Nice try, but that's not your entry. Ask the one who logged it, or reply to one of your own confirmations.",
				"My lab, my rules: only the owner of a transaction can fix or delete it. That one isn't yours.",
			},
		},
		constant.ErrUserNotRegister.Code: {
			Emoji: "🕯️",
			Messages: []string{
				"Who... are you? We haven't met yet. Introduce yourself first: *register <name> <email>*",
				"*sniffs* ...a stranger! Frankie only works for registered bosses. Type: *register <name> <email>*",
				"Hi! I'd love to help, but I don't know you yet. Register first: *register <name> <email>*",
				"My lab, my rules: no name, no ledger. Type: *register <name> <email>*",
				"Frankie doesn't recognize this face. Let's fix that: *register <name> <email>*",
			},
		},
		constant.ErrAlreadyRegister.Code: {
			Emoji: "🧷",
			Messages: []string{
				"We've met before, bos. Frankie never forgets a face. Just send me your transactions.",
				"You already brought me to life. Once is enough! Send me a transaction instead.",
				"Hi! You're already registered, no need to stitch me together twice.",
				"Frankie remembers you already. Go ahead and log something, like: buy coffee 5K today",
			},
		},
		constant.ErrServiceUnavailable.Code: {
			Emoji: "🧠",
			Messages: []string{
				"The lab's lightning is weak right now. Please send that again in a moment.",
				"Frankie's brain is buzzing too hard... give me a moment, then send that again.",
				"Too many sparks in the lab right now. Try sending that again in a bit.",
				"*bolts flickering* ...the machine needs a short rest. Please send that again soon.",
			},
		},
		constant.ErrInternal.Code: {
			Emoji: "🥱",
			Messages: []string{
				"Uh oh... something came unstitched in the lab. Please try again.",
				"*sparks fly* ...that didn't work, bos. Please try again.",
				"Frankie tripped over a cable. Please try that again.",
				"Something broke in the lab. Please try again in a moment.",
			},
		},
		// report errors
		constant.ErrReportNotFound.Code: {
			Emoji: "😭",
			Messages: []string{
				"Every jar on the lab shelf is empty for those dates. No coins came in, no coins went out.",
				"*taps the counting machine* ...it says zero. Frankie checked twice. No transactions there.",
				"Sorry, bos! Frankie wanted to stitch a report together, but there are no pieces to sew.",
				"Quiet days, huh? Not a single entry in that range. Frankie's bolts are almost disappointed.",
				"Frankie sniffed every page of those dates. Nothing but old ink smell. No transactions found.",
				"A report with no numbers is just a sad piece of paper, bos. Nothing was logged in that period.",
			},
		},
		constant.ErrUnclearReportPeriod.Code: {
			Emoji: "🎃",
			Messages: []string{
				"*squints at the calendar on the lab wall* ...which days, bos? Try more specific date",
				"Frankie's calendar gears are spinning but not clicking. Tell me the period, and try be more specific date",
				"A report for... when? Frankie needs a date to dig through. Try more specific date",
				"Sorry, bos! Frankie is ready to count, but doesn't know where to start. Try more specific date",
				"*scratches a bolt* ...yesterday? Last month? Frankie can't guess. Say something like this month or yesterday",
				"Frankie only reads calendars, not minds. Give me a period, for example: this week or today",
				"The time machine in the lab needs a destination, bos. Try more specific date",
				"Frankie got lost somewhere between Monday and forever. Pick a period, like: August or September",
			},
		},
		constant.ErrInvalidReportPeriodRange.Code: {
			Emoji: "😮‍💨",
			Messages: []string{
				"*wheezes under a mountain of pages* ... too much, bos! Frankie can only summarize one month at a time.",
				"That range would blow every fuse in the lab. Keep it to one month, max.",
				"Sorry, bos! Frankie's arms are stitched on, not endless. One month per report, please.",
				"*bolts overheating* ...that's way more days than Frankie can carry. Pick a period of a month or less.",
			},
		},

		// transaction errors
		constant.ErrTransactionNotFound.Code: {
			Emoji: "🙈",
			Messages: []string{
				"Hmm... Frankie can't find that transaction in the big book anymore. Maybe it was already deleted?",
				"*digs through the lab drawers* ...nothing! That entry is gone, bos. Send it again as a new transaction if you need it.",
				"Sorry, bos! That transaction isn't in the ledger anymore, so there's nothing to change.",
				"Frankie looked everywhere, but that entry has vanished. If it was deleted, just log it again.",
				"That page was torn out of the big book already. Send a new transaction instead, like: buy coffee 5K today",
				"No trace of that one in the lab, bos. It may have been deleted earlier.",
			},
		},
		constant.ErrUnclearTransactionDate.Code: {
			Emoji: "😵‍💫",
			Messages: []string{
				"*stares at the lab calendar* ...Frankie can't tell which day that happened, bos.",
				"Frankie caught the amount but dropped the date somewhere between the test tubes. Which day was it?",
				"Sorry, bos! When did that happen? Frankie needs a day to pin it on.",
				"*bolts buzzing* ...that date doesn't fit any square on Frankie's calendar.",
				"Frankie wants to file this on the right page, but which page? Tell me the day.",
				"Today? Last week? The day the lab exploded? Frankie isn't sure which day you mean.",
				"Frankie's stitched brain can't place that day on the calendar. Say the date again, bos?",
				"The date slipped through Frankie's fingers, all six of them. Which day was that?",
			},
		},
		constant.ErrInvalidTransactionType.Code: {
			Emoji: "🫨",
			Messages: []string{
				"*tilts stitched head* ...money in or money out, bos? Frankie can't tell which one this is.",
				"Frankie has two jars on the shelf: income and expense. This one doesn't fit either.",
				"Sorry, bos! Did you earn that or spend it? Frankie's brain went fuzzy on that part.",
				"*bolts sparking* ...Frankie got the money, but not which way it went. Coming in or going out?",
				"Frankie's scale has two sides, and this coin is balancing right in the middle. Income or expense?",
				"The lab ledger has a plus column and a minus column. Frankie doesn't know where to write this one.",
				"Hmm, was that money arriving or leaving, bos? Frankie won't guess with your coins.",
				"Frankie sniffed the coins and still can't tell. Is it something you got, or something you paid?",
			},
		},

		// integration errors
		constant.ErrPrivateChatOnly.Code: {
			Emoji: "🔐",
			Messages: []string{
				"Psst, bos... that key is secret! Ask me for *mcp* in our private chat, not in a group.",
				"*hides the key behind back* ...not here! Send *mcp* to me in a private chat.",
				"My lab, my rules: secret keys are only handed over in private. Send *mcp* to me directly.",
			},
		},
	}
)

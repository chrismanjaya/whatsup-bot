package message

// Frankie is the bot's persona: a friendly creature stitched together in a
// lab, whose one job is keeping track of the user's money. The pools below
// give each reply several in-character variants, so the bot doesn't sound
// scripted. internal/persona picks one at random (never the same one twice
// in a row for a user) and fills the [[placeholder]] tokens.
//
// Rules for writing variants:
//   - Keep any instruction the user must follow (e.g. "register <name>
//     <email>") identical in every variant: only the personality changes.
//   - Don't start a variant with a BotReplyPrefixes entry.
//   - Don't use *, _, ~ or ` inside quips (they break WhatsApp formatting).
//   - No emoji in error variants: the router appends the error's code emoji
//     (ErrorEmoji) at the end, and it must be the only one.

// Lang is the language a reply is written in.
type Lang string

const (
	LangEN Lang = "en"
	LangID Lang = "id"
)

// ReplyLang is the language every Frankie reply is written in, to match the
// English templates. Set it to "" to reply in the language the user wrote
// in instead (the Indonesian pools below are kept for that).
const ReplyLang Lang = LangEN

// PoolKey names one pool of reply variants.
type PoolKey string

const (
	PoolInvalidRequest     PoolKey = "invalid_request"     // 400
	PoolNotRegistered      PoolKey = "not_registered"      // 404
	PoolAlreadyRegistered  PoolKey = "already_registered"  // 409
	PoolServiceUnavailable PoolKey = "service_unavailable" // 503
	PoolGeneric            PoolKey = "generic"             // 500 and anything unclassified
	PoolReplyToQuip        PoolKey = "reply_to_quip"       // 400: user replied to a quip instead of the confirmation
)

// QuipPrefix starts every quip message. It marks the message as Frankie's
// chatter (not a record), and lets the router recognize a reply to a quip
// from the quoted text alone, with nothing stored.
const QuipPrefix = "💬 "

// Placeholders filled in by persona.
const (
	PlaceholderGreeting = "[[greeting]]"
)

// Greetings by time of day (Asia/Jakarta), used for [[greeting]].
var Greetings = map[Lang]map[string]string{
	LangEN: {"night": "Still up", "morning": "Morning", "afternoon": "Afternoon", "evening": "Evening"},
	LangID: {"night": "Masih begadang", "morning": "Pagi", "afternoon": "Siang", "evening": "Malam"},
}

// ErrorEmoji maps an error code to the emoji appended to the error reply.
// It identifies the error for debugging without showing a technical code:
// look up the sender and time in the logs for the full error. Keep them
// visually distinct, and don't use them anywhere else in the pools.
var ErrorEmoji = map[int]string{
	400: "🧪",  // not a transaction
	404: "🕯️", // not registered
	409: "🧷",  // already registered
	503: "🌩️", // Gemini busy
	500: "🧠",  // internal error
}

// ErrorEmojiDefault is used for a code with no entry in ErrorEmoji.
const ErrorEmojiDefault = "🧠"

// ErrorPools holds the variants for each error reply. The code emoji is
// appended by the router (see ErrWithCode), so it's not part of the text.
var ErrorPools = map[PoolKey]map[Lang][]string{
	PoolInvalidRequest: {
		LangEN: {
			"Hmm... Frankie only understands money, bos. Try something like: makan siang 25rb",
			"*scratches stitched head* ...Income? Expense? Tell me one of those. Example: gaji 10jt",
			"I was stitched together for one job: tracking your money. Try: kopi 20rb",
			"[[greeting]], bos! That one went over my bolts. Send me a transaction, like: bensin 50rb",
			"Frankie's brain is small and full of numbers. Give me an expense or income, like: parkir 5rb",
			"Not sure what to do with that. I log income and expenses, e.g. beli buku 80rb",
		},
		LangID: {
			"Hmm... Frankie cuma ngerti soal uang, bos. Coba kayak gini: makan siang 25rb",
			"*garuk kepala jahitan* ...Pemasukan? Pengeluaran? Kasih tau salah satunya. Contoh: gaji 10jt",
			"Frankie dijahit cuma buat satu tugas: catat uangmu. Coba: kopi 20rb",
			"[[greeting]], bos! Yang itu Frankie nggak nangkep. Kirim transaksi aja, misalnya: bensin 50rb",
			"Otak Frankie kecil dan isinya angka semua. Kasih pengeluaran atau pemasukan, misalnya: parkir 5rb",
			"Frankie bingung mau diapain. Frankie cuma catat pemasukan dan pengeluaran, misalnya: beli buku 80rb",
		},
	},
	PoolNotRegistered: {
		LangEN: {
			"Who... are you? We haven't met yet. Introduce yourself first: register <name> <email>",
			"*sniffs* ...a stranger! Frankie only works for registered bosses. Type: register <name> <email>",
			"[[greeting]]! I'd love to help, but I don't know you yet. Register first: register <name> <email>",
			"My lab, my rules: no name, no ledger. Type: register <name> <email>",
			"Frankie doesn't recognize this face. Let's fix that: register <name> <email>",
		},
		LangID: {
			"Kamu... siapa? Kita belum kenalan. Kenalan dulu yuk: register <name> <email>",
			"*endus-endus* ...orang asing! Frankie cuma kerja buat bos yang udah daftar. Ketik: register <name> <email>",
			"[[greeting]]! Frankie mau bantu, tapi belum kenal kamu. Daftar dulu: register <name> <email>",
			"Lab Frankie, aturan Frankie: belum daftar, belum bisa dicatat. Ketik: register <name> <email>",
			"Frankie belum kenal wajah ini. Ayo kenalan: register <name> <email>",
		},
	},
	PoolAlreadyRegistered: {
		LangEN: {
			"We've met before, bos. Frankie never forgets a face. Just send me your transactions.",
			"You already brought me to life. Once is enough! Send me a transaction instead.",
			"[[greeting]]! You're already registered, no need to stitch me together twice.",
			"Frankie remembers you already. Go ahead and log something, like: makan siang 25rb",
		},
		LangID: {
			"Kita udah kenalan, bos. Frankie nggak pernah lupa wajah. Langsung kirim transaksimu aja.",
			"Kamu udah menghidupkan Frankie. Sekali aja cukup! Kirim transaksi aja ya.",
			"[[greeting]]! Kamu udah terdaftar, nggak perlu jahit Frankie dua kali.",
			"Frankie udah inget kamu kok. Langsung catat aja, misalnya: makan siang 25rb",
		},
	},
	PoolServiceUnavailable: {
		LangEN: {
			"The lab's lightning is weak right now. Please send that again in a moment.",
			"Frankie's brain is buzzing too hard... give me a moment, then send that again.",
			"Too many sparks in the lab right now. Try sending that again in a bit.",
			"*bolts flickering* ...the machine needs a short rest. Please send that again soon.",
		},
		LangID: {
			"Petir di lab lagi lemah nih. Kirim ulang sebentar lagi ya.",
			"Otak Frankie lagi kepenuhan... tunggu sebentar, terus kirim ulang ya.",
			"Lab lagi sibuk banget. Coba kirim ulang sebentar lagi.",
			"*baut kedip-kedip* ...mesinnya butuh istirahat sebentar. Kirim ulang nanti ya.",
		},
	},
	PoolReplyToQuip: {
		LangEN: {
			"That was just Frankie chatting, bos. To change a transaction, reply to its confirmation message instead.",
			"Frankie's small talk can't be edited. Reply to the confirmation message to update or delete the transaction.",
			"*scratches stitched head* ...that was only a comment. Reply to the message with the transaction details instead.",
			"Ah, that's just Frankie talking. To fix a transaction, reply to its confirmation message.",
		},
		LangID: {
			"Itu cuma Frankie ngobrol, bos. Untuk ubah transaksi, balas pesan konfirmasinya ya.",
			"Obrolan Frankie nggak bisa diubah. Balas pesan konfirmasi untuk ubah atau hapus transaksi.",
			"*garuk kepala jahitan* ...itu cuma komentar. Balas pesan yang berisi detail transaksinya ya.",
			"Itu cuma Frankie ngomong. Untuk benerin transaksi, balas pesan konfirmasinya.",
		},
	},
	PoolGeneric: {
		LangEN: {
			"Uh oh... something came unstitched in the lab. Please try again.",
			"*sparks fly* ...that didn't work, bos. Please try again.",
			"Frankie tripped over a cable. Please try that again.",
			"Something broke in the lab. Please try again in a moment.",
		},
		LangID: {
			"Waduh... ada jahitan yang lepas di lab. Coba lagi ya.",
			"*percikan api* ...gagal, bos. Coba lagi ya.",
			"Frankie kesandung kabel. Coba lagi ya.",
			"Ada yang rusak di lab. Coba lagi sebentar lagi ya.",
		},
	},
}

// How often a new transaction gets a quip. Income is rarer and good news,
// so it always gets one; everyday expenses only sometimes, so it doesn't
// get repetitive.
const (
	QuipIncomeChancePct  = 100
	QuipExpenseChancePct = 35
)

// Quip fallbacks: used when a transaction gets a quip but Gemini didn't
// return one (or the transaction was completed via a reply, which has no
// Gemini quip). Keyed by category; expenses in a category with no pool use
// QuipExpenseDefault, income uses QuipIncome. Quips are always English, to
// match the English confirmation template they sit in.
var QuipByCategory = map[string][]string{
	"food":          {"Smells good from here... save me a bite, bos?", "Eating well keeps the stitches strong 🍽️", "Frankie is hungry now too. Thanks a lot."},
	"groceries":     {"Cooking tonight? Frankie will guard the leftovers.", "A stocked fridge is a happy lab 🥬"},
	"transport":     {"Feeding the metal beast. Vroom ⚡", "Safe trip, bos. Frankie stays and guards the lab."},
	"utilities":     {"Electricity paid! Frankie needs that to stay alive ⚡", "Bills paid, the lab lights stay on."},
	"entertainment": {"Have fun, bos. Even creatures need a break.", "Frankie approves of fun, in moderation 🎬"},
	"health":        {"Take care of yourself, bos. Frankie knows a thing about patching up.", "Health first. Stitches second."},
	"medicine":      {"Get well soon, bos 🩹", "Frankie hopes you feel better soon."},
	"investment":    {"Planting coins for the future. Smart, bos 🌱", "Money that works while you sleep. Frankie approves."},
}

var QuipExpenseDefault = []string{"Logged and stitched into the ledger 🔩", "Frankie wrote it down. Nothing escapes this notebook.", "Noted, bos ⚡"}

var QuipIncome = []string{"IT'S ALIVE! Your wallet breathes again 💰", "Money in! Frankie is doing a happy stomp.", "The lab hums with joy. Nice one, bos ⚡"}

// Package persona gives the bot Frankie's voice without spending AI tokens:
// it picks a random variant from the pools in internal/message, avoids
// repeating the last one a user saw, and fills in context like the time of
// day. It depends only on the standard library and internal/message, so
// both usecases and adapters can use it.
package persona

import (
	"math/rand/v2"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"whatsup-bot/internal/message"
)

// maxQuipRunes caps a quip's length, since it can come from the LLM.
const maxQuipRunes = 120

// Picker chooses reply variants. It's safe for concurrent use.
type Picker struct {
	mu   sync.Mutex
	rng  *rand.Rand
	now  func() time.Time
	last map[string]int          // "<userJID>|<pool id>" -> index last shown
	lang map[string]message.Lang // userJID -> language of their last message with words
}

// New returns a Picker with a random seed and the real clock.
func New() *Picker {
	return NewWithSource(rand.NewPCG(rand.Uint64(), rand.Uint64()), time.Now)
}

// NewWithSource returns a Picker with a fixed random source and clock, for
// deterministic tests.
func NewWithSource(src rand.Source, now func() time.Time) *Picker {
	return &Picker{rng: rand.New(src), now: now, last: map[string]int{}, lang: map[string]message.Lang{}}
}

// Lang returns the language to reply to userJID in. It detects it from text
// (see DetectLang) and remembers it, so a message with no words to go on,
// like a bare amount "509589", gets the user's last language instead of
// defaulting to English.
func (p *Picker) Lang(userJID, text string) message.Lang {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !hasWords(text) {
		if l, ok := p.lang[userJID]; ok {
			return l
		}
		return message.LangEN
	}
	l := DetectLang(text)
	p.lang[userJID] = l
	return l
}

// hasWords reports whether text has at least one word of 2+ letters, i.e.
// something DetectLang can go on (amount suffixes like "25rb" count).
func hasWords(text string) bool {
	run := 0
	for _, r := range strings.ToLower(text) {
		if r >= 'a' && r <= 'z' {
			run++
			if run >= 2 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

// Error returns a variant of the error reply for key, in lang, for user.
func (p *Picker) Error(key message.PoolKey, userJID string, lang message.Lang) string {
	return p.pick("err:"+string(key), userJID, variants(message.ErrorPools[key], lang), lang)
}

// Chance reports true pct% of the time.
func (p *Picker) Chance(pct int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rng.IntN(100) < pct
}

// FallbackQuip returns a pooled (English) quip for a transaction: by
// category for expenses (or a generic expense quip if the category has no
// pool), and the income pool for income.
func (p *Picker) FallbackQuip(category string, isIncome bool, userJID string) string {
	if isIncome {
		return p.pick("quip:income", userJID, message.QuipIncome, message.LangEN)
	}
	if pool, ok := message.QuipByCategory[category]; ok {
		return p.pick("quip:"+category, userJID, pool, message.LangEN)
	}
	return p.pick("quip:expense", userJID, message.QuipExpenseDefault, message.LangEN)
}

// pick returns a random variant, never the same index twice in a row for
// the same user and pool (when the pool has more than one variant).
func (p *Picker) pick(poolID, userJID string, opts []string, lang message.Lang) string {
	if len(opts) == 0 {
		return ""
	}

	p.mu.Lock()
	k := userJID + "|" + poolID
	i := 0
	if len(opts) > 1 {
		if last, seen := p.last[k]; seen && last < len(opts) {
			// Pick from the other n-1 variants.
			i = p.rng.IntN(len(opts) - 1)
			if i >= last {
				i++
			}
		} else {
			i = p.rng.IntN(len(opts))
		}
	}
	p.last[k] = i
	now := p.now()
	p.mu.Unlock()

	return strings.ReplaceAll(opts[i], message.PlaceholderGreeting, Greeting(now, lang))
}

// variants returns the pool's variants in lang, falling back to English.
func variants(pool map[message.Lang][]string, lang message.Lang) []string {
	if v := pool[lang]; len(v) > 0 {
		return v
	}
	return pool[message.LangEN]
}

// Greeting returns a time-of-day greeting for t, in Asia/Jakarta time.
func Greeting(t time.Time, lang message.Lang) string {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		t = t.In(loc)
	}
	part := "evening"
	switch h := t.Hour(); {
	case h < 4:
		part = "night"
	case h < 11:
		part = "morning"
	case h < 18:
		part = "afternoon"
	}
	greetings := message.Greetings[lang]
	if greetings == nil {
		greetings = message.Greetings[message.LangEN]
	}
	return greetings[part]
}

// indonesianWords are common words that mark a message as Indonesian.
var indonesianWords = map[string]bool{
	"beli": true, "bayar": true, "makan": true, "minum": true, "jajan": true, "belanja": true,
	"gaji": true, "uang": true, "duit": true, "transaksi": true, "laporan": true, "rekap": true,
	"ringkasan": true, "kemarin": true, "hari": true, "ini": true, "bulan": true, "minggu": true,
	"tanggal": true, "lalu": true, "rb": true, "ribu": true, "jt": true, "juta": true,
	"dan": true, "di": true, "ke": true, "dari": true, "untuk": true, "buat": true,
	"yang": true, "sudah": true, "udah": true, "aku": true, "saya": true, "gue": true,
	"tolong": true, "dong": true, "nih": true, "ya": true, "apa": true, "siapa": true,
	"daftar": true, "hapus": true, "ubah": true, "jadi": true, "tadi": true, "siang": true,
	"pagi": true, "malam": true, "sore": true, "bensin": true, "parkir": true, "pulsa": true,
	"halo": true, "hai": true, "permisi": true, "makasih": true, "terima": true, "kasih": true,
	"gimana": true, "bisa": true, "mau": true, "nggak": true, "gak": true, "tidak": true,
	"bunga": true, "tabungan": true, "setor": true, "tarik": true, "tunai": true, "cicilan": true,
}

// DetectLang guesses whether text is Indonesian or English with a simple
// keyword check, so error replies can match the user's language without a
// model call. Defaults to English.
func DetectLang(text string) message.Lang {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z')
	})
	for _, w := range fields {
		if indonesianWords[w] {
			return message.LangID
		}
	}
	// Amounts like "25rb" or "1jt" are split into "rb"/"jt" above, so they
	// count too.
	return message.LangEN
}

// amountWords are Indonesian amount words that are normal inside an English
// quip ("Nasi goreng for 5 ribu?!"), so IsEnglish ignores them.
var amountWords = map[string]bool{"rb": true, "ribu": true, "jt": true, "juta": true, "k": true}

// IsEnglish reports whether a quip reads as English: it has no Indonesian
// words other than amounts. Food, brand and place names aren't in the
// word list, so they're fine.
func IsEnglish(text string) bool {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z')
	})
	for _, w := range fields {
		if indonesianWords[w] && !amountWords[w] {
			return false
		}
	}
	return true
}

// SanitizeQuip makes an LLM-written quip safe to show: one line, no
// WhatsApp formatting characters, and capped in length. It returns "" if
// nothing usable is left.
func SanitizeQuip(q string) string {
	q = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return ' '
		case '*', '_', '~', '`':
			return -1
		}
		return r
	}, q)
	q = strings.Join(strings.Fields(q), " ")
	if utf8.RuneCountInString(q) > maxQuipRunes {
		r := []rune(q)[:maxQuipRunes]
		cut := string(r)
		if sp := strings.LastIndex(cut, " "); sp > maxQuipRunes/2 {
			cut = cut[:sp]
		}
		q = strings.TrimRight(cut, " ,.;:-") + "…"
	}
	return q
}

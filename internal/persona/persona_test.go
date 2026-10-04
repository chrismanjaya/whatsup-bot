package persona

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"whatsup-bot/internal/message"
)

var allLangs = []message.Lang{message.LangEN, message.LangID}

func fixedPicker(seed uint64, now time.Time) *Picker {
	return NewWithSource(rand.NewPCG(seed, seed), func() time.Time { return now })
}

func TestErrorPoolsComplete(t *testing.T) {
	keys := []message.PoolKey{
		message.PoolInvalidRequest, message.PoolNotRegistered, message.PoolAlreadyRegistered,
		message.PoolServiceUnavailable, message.PoolGeneric,
	}
	for _, k := range keys {
		for _, l := range allLangs {
			if n := len(message.ErrorPools[k][l]); n < 3 {
				t.Errorf("pool %s/%s has %d variants, want at least 3", k, l, n)
			}
		}
	}
}

func TestQuipPoolsComplete(t *testing.T) {
	for _, l := range allLangs {
		if len(message.QuipIncome[l]) == 0 || len(message.QuipExpenseDefault[l]) == 0 {
			t.Errorf("default quip pools missing language %s", l)
		}
		for cat, pool := range message.QuipByCategory {
			if len(pool[l]) == 0 {
				t.Errorf("quip pool %q missing language %s", cat, l)
			}
		}
	}
}

func TestNotRegisteredKeepsInstruction(t *testing.T) {
	for _, l := range allLangs {
		for _, v := range message.ErrorPools[message.PoolNotRegistered][l] {
			if !strings.Contains(v, "register <name> <email>") {
				t.Errorf("not-registered variant lost the instruction: %q", v)
			}
		}
	}
}

func TestNoVariantLooksLikeBotReply(t *testing.T) {
	for k, pool := range message.ErrorPools {
		for _, vs := range pool {
			for _, v := range vs {
				for _, p := range message.BotReplyPrefixes {
					if strings.HasPrefix(v, p) {
						t.Errorf("pool %s variant %q starts with bot reply prefix %q", k, v, p)
					}
				}
			}
		}
	}
}

func TestPickNeverRepeatsBackToBack(t *testing.T) {
	p := fixedPicker(42, time.Now())
	for _, l := range allLangs {
		prev := ""
		for i := 0; i < 200; i++ {
			got := p.Error(message.PoolInvalidRequest, "user-1", l)
			if got == prev {
				t.Fatalf("variant repeated back-to-back at step %d: %q", i, got)
			}
			prev = got
		}
	}
}

func TestPickUsesEveryVariant(t *testing.T) {
	p := fixedPicker(7, time.Now())
	pool := message.ErrorPools[message.PoolGeneric][message.LangEN]
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		seen[p.Error(message.PoolGeneric, "u", message.LangEN)] = true
	}
	if len(seen) != len(pool) {
		t.Errorf("saw %d distinct variants, want %d", len(seen), len(pool))
	}
}

func TestPickFillsPlaceholders(t *testing.T) {
	p := fixedPicker(1, time.Now())
	for k := range message.ErrorPools {
		for _, l := range allLangs {
			for i := 0; i < 50; i++ {
				if got := p.Error(k, "u", l); strings.Contains(got, "[[") {
					t.Fatalf("unfilled placeholder in %s/%s: %q", k, l, got)
				}
			}
		}
	}
}

func TestGreeting(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skip("no tzdata")
	}
	cases := []struct {
		hour int
		lang message.Lang
		want string
	}{
		{2, message.LangEN, "Still up"},
		{7, message.LangID, "Pagi"},
		{13, message.LangEN, "Afternoon"},
		{20, message.LangID, "Malam"},
	}
	for _, c := range cases {
		got := Greeting(time.Date(2026, 10, 4, c.hour, 0, 0, 0, jkt), c.lang)
		if got != c.want {
			t.Errorf("Greeting(%02d:00, %s) = %q, want %q", c.hour, c.lang, got, c.want)
		}
	}
}

func TestDetectLang(t *testing.T) {
	cases := map[string]message.Lang{
		"beli nasi goreng 5000":     message.LangID,
		"makan siang 25rb":          message.LangID,
		"transaksi kemarin":         message.LangID,
		"kopi 20rb":                 message.LangID,
		"spent 50k on lunch":        message.LangEN,
		"what's the weather today?": message.LangEN,
		"hello":                     message.LangEN,
		"":                          message.LangEN,
	}
	for in, want := range cases {
		if got := DetectLang(in); got != want {
			t.Errorf("DetectLang(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestChance(t *testing.T) {
	p := fixedPicker(3, time.Now())
	hits := 0
	const n = 10000
	for i := 0; i < n; i++ {
		if p.Chance(35) {
			hits++
		}
	}
	if pct := hits * 100 / n; pct < 32 || pct > 38 {
		t.Errorf("Chance(35) hit %d%%, want about 35%%", pct)
	}
}

func TestSanitizeQuip(t *testing.T) {
	cases := map[string]string{
		"  Nasi goreng 5 ribu?! *mau*  ": "Nasi goreng 5 ribu?! mau",
		"line one\nline two":             "line one line two",
		"_italic_ ~strike~ `code`":       "italic strike code",
		"   ":                            "",
	}
	for in, want := range cases {
		if got := SanitizeQuip(in); got != want {
			t.Errorf("SanitizeQuip(%q) = %q, want %q", in, got, want)
		}
	}

	long := SanitizeQuip(strings.Repeat("frankie ", 40))
	if n := len([]rune(long)); n > maxQuipRunes+1 || !strings.HasSuffix(long, "…") {
		t.Errorf("long quip not capped: %d runes, %q", n, long)
	}
}

func TestFallbackQuip(t *testing.T) {
	p := fixedPicker(9, time.Now())
	if q := p.FallbackQuip("salary", true, "u", message.LangEN); !contains(message.QuipIncome[message.LangEN], q) {
		t.Errorf("income quip %q not from income pool", q)
	}
	if q := p.FallbackQuip("food", false, "u", message.LangID); !contains(message.QuipByCategory["food"][message.LangID], q) {
		t.Errorf("food quip %q not from food pool", q)
	}
	if q := p.FallbackQuip("tax", false, "u", message.LangEN); !contains(message.QuipExpenseDefault[message.LangEN], q) {
		t.Errorf("tax quip %q not from default expense pool", q)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

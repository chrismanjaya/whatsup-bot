package whatsapp

import (
	"time"
	"unicode/utf8"
)

func typingDuration(text string, typingSince time.Time) time.Duration {
	d := typingBase + time.Duration(utf8.RuneCountInString(text))*typingPerRune
	if d > typingMax {
		return typingMax
	}

	remainingDuration := d - time.Since(typingSince)
	if remainingDuration < 0 {
		return 0
	}

	return remainingDuration
}

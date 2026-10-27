package util

import "time"

var Jakarta = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*3600)
}()

// SameDay reports whether a and b fall on the same calendar day in Jakarta.
func IsSameDay(a, b time.Time) bool {
	ay, am, ad := a.In(Jakarta).Date()
	by, bm, bd := b.In(Jakarta).Date()
	return ay == by && am == bm && ad == bd
}

func GetStartOfMonth() time.Time {
	return time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
}

func GetTodayLocalTimeString(layout string) string {
	return time.Now().In(Jakarta).Format(layout)
}

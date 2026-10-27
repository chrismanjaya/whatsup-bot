package util

import "testing"

func TestContainsAnyWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		want     bool
	}{
		{name: "matches whole word", text: "list this week", keywords: []string{"list"}, want: true},
		{name: "ignores keyword inside a word", text: "bayar listrik 200rb", keywords: []string{"list"}, want: false},
		{name: "ignores case", text: "Laporan bulan ini", keywords: []string{"laporan"}, want: true},
		{name: "splits on formatting characters", text: "*EXPENSE*\n- Desc: *kopi*", keywords: []string{"EXPENSE"}, want: true},
		{name: "splits on digits boundary punctuation", text: "*PAGE 1/3*", keywords: []string{"page"}, want: true},
		{name: "single-letter keyword needs its own word", text: "sebelumnya", keywords: []string{"n", "b"}, want: false},
		{name: "single-letter keyword alone", text: "n", keywords: []string{"n"}, want: true},
		{name: "empty text", text: "", keywords: []string{"list"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAnyWord(tt.text, tt.keywords); got != tt.want {
				t.Fatalf("ContainsAnyWord(%q, %v) = %v, want %v", tt.text, tt.keywords, got, tt.want)
			}
		})
	}
}

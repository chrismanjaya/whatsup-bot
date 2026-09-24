package usecase

import "testing"

func TestParsePageCommand(t *testing.T) {
	cases := []struct {
		in      string
		current int
		want    int
		ok      bool
	}{
		{"next", 2, 3, true},
		{"Lanjut", 2, 3, true},
		{"prev", 2, 1, true},
		{"sebelumnya", 1, 0, true},
		{"page 10", 1, 10, true},
		{"halaman 2", 5, 2, true},
		{"3", 1, 3, true},
		{"page-4", 1, 4, true},
		{"hapus", 1, 0, false},
		{"page x", 1, 0, false},
	}
	for _, c := range cases {
		got, ok := parsePageCommand(c.in, c.current)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("parsePageCommand(%q, %d) = %d, %v; want %d, %v", c.in, c.current, got, ok, c.want, c.ok)
		}
	}
}

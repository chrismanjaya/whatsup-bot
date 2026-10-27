package prediction

import (
	"context"
	"testing"
)

func TestPredictAction(t *testing.T) {
	tests := []struct {
		name       string
		rawMessage string
		quotedText string
		isGroup    bool
		isReply    bool
		want       Action
	}{
		{name: "records electricity bill (listrik is not list)", rawMessage: "bayar listrik 200rb", want: ACTION_RECORD},
		{name: "records report-like word inside another word", rawMessage: "beli reporter kit 50k", want: ACTION_RECORD},
		{name: "detail on list keyword", rawMessage: "list this week", want: ACTION_DETAIL},
		{name: "detail on transaksi", rawMessage: "transaksi kemarin", want: ACTION_DETAIL},
		{name: "summary on laporan", rawMessage: "laporan bulan ini", want: ACTION_SUMMARY},
		{name: "registration", rawMessage: "register Budi budi@mail.com", want: ACTION_REGISTRATION},
		{name: "split only in groups", rawMessage: "split", isGroup: true, want: ACTION_SPLIT},
		{name: "split outside group records", rawMessage: "split", want: ACTION_RECORD},
		{name: "reply to page message", quotedText: "*PAGE 1/3*\n- Period: ...", isReply: true, want: ACTION_DETAIL_PAGE},
		{name: "reply to transaction confirmation", quotedText: "*EXPENSE*\n- Desc: *kopi*", isReply: true, want: ACTION_AMEND},
		{name: "reply to unrelated message", quotedText: "see you tomorrow", isReply: true, want: ACTION_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PredictAction(context.Background(), tt.rawMessage, tt.quotedText, tt.isGroup, tt.isReply)
			if got != tt.want {
				t.Fatalf("PredictAction(%q, %q) = %v, want %v", tt.rawMessage, tt.quotedText, got, tt.want)
			}
		})
	}
}

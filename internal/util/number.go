package util

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type IntOrInt64 interface {
	int | int64
}

func FormatNumber[T IntOrInt64](num T, locale language.Tag) string {
	printer := message.NewPrinter(locale)
	return printer.Sprintf("%d", num)
}

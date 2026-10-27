package util

import (
	"math"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var defaultCurrency = currency.IDR

func FormatCurrency(amountInt64 int64, currencyCode string, locale language.Tag) string {
	cur, err := currency.ParseISO(currencyCode)
	if err != nil {
		LogError("Parse currency failed", err)
		cur = defaultCurrency
	}

	scale, _ := currency.Cash.Rounding(cur)
	divisor := math.Pow(10, float64(scale))
	valueFloat := float64(amountInt64) / divisor

	decFormatter := number.Decimal(valueFloat, number.Scale(scale))
	printer := message.NewPrinter(locale)
	return printer.Sprintf("%v%v", currency.Symbol(cur), decFormatter)
}

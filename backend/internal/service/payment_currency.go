package service

import (
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func paymentProviderConfigCurrency(providerKey string, cfg map[string]string) string {
	switch strings.TrimSpace(providerKey) {
	case payment.TypeStripe, payment.TypeAirwallex:
		currency, err := payment.NormalizePaymentCurrency(cfg["currency"])
		if err == nil {
			return currency
		}
	}
	return payment.DefaultPaymentCurrency
}

func PaymentOrderCurrency(order *dbent.PaymentOrder) string {
	if order == nil {
		return payment.DefaultPaymentCurrency
	}
	// New orders persist both values. Prefer the provider snapshot for legacy
	// rows because settlement_currency was added later and defaulted to CNY;
	// fall back to the persisted settlement currency when no usable snapshot
	// exists.
	if snapshot := psOrderProviderSnapshot(order); snapshot != nil {
		if currency, err := payment.NormalizePaymentCurrency(snapshot.Currency); err == nil {
			return currency
		}
	}
	if currency, err := payment.NormalizePaymentCurrency(order.SettlementCurrency); err == nil {
		return currency
	}
	return payment.DefaultPaymentCurrency
}

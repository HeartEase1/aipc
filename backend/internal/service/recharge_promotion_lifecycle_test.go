//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type rechargeCloseTestProvider struct {
	paymentOrderLifecycleQueryProvider
	closeErr error
}

func (p *rechargeCloseTestProvider) CancelPayment(ctx context.Context, ref string) error {
	_ = p.paymentOrderLifecycleQueryProvider.CancelPayment(ctx, ref)
	return p.closeErr
}

func TestRechargeCancellationPreservesProviderCloseForGracePeriod(t *testing.T) {
	for _, tc := range []struct {
		name, orderType string
		cancel          bool
		closeErr        error
		wantSaved       bool
	}{
		{"balance closed", payment.OrderTypeBalance, true, nil, true},
		{"close uncertain", payment.OrderTypeBalance, true, errors.New("network timeout"), false},
		{"read only reconciliation", payment.OrderTypeBalance, false, nil, false},
		{"subscription unchanged", payment.OrderTypeSubscription, true, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			prov := &rechargeCloseTestProvider{paymentOrderLifecycleQueryProvider: paymentOrderLifecycleQueryProvider{resp: &payment.QueryOrderResponse{Status: payment.ProviderStatusPending}}, closeErr: tc.closeErr}
			registry := payment.NewRegistry()
			registry.Register(prov)
			svc := &PaymentService{entClient: newPaymentConfigServiceTestClient(t), registry: registry, providersLoaded: true, sqlDB: db}
			if tc.wantSaved {
				mock.ExpectExec("UPDATE recharge_promotion_claims SET closed_confirmed_at=NOW\\(\\).*status='reserved' AND closed_confirmed_at IS NULL").WithArgs(int64(8)).WillReturnResult(sqlmock.NewResult(0, 1))
			}
			outcome := svc.checkPaidWithOptions(context.Background(), &dbent.PaymentOrder{ID: 8, PaymentType: payment.TypeAlipay, OutTradeNo: "order-8", OrderType: tc.orderType}, checkPaidOptions{cancelIfUnpaid: tc.cancel})
			require.Empty(t, outcome)
			if tc.cancel {
				require.Equal(t, 1, prov.cancelCalls)
			} else {
				require.Zero(t, prov.cancelCalls)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

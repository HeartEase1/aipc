package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSubscriptionAction(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "extend", " EXTEND "} {
		action, err := normalizeSubscriptionAction(input)
		require.NoError(t, err)
		require.Equal(t, payment.SubscriptionActionExtend, action)
	}
	action, err := normalizeSubscriptionAction(" Restart ")
	require.NoError(t, err)
	require.Equal(t, payment.SubscriptionActionRestart, action)

	_, err = normalizeSubscriptionAction("reset-someone-else")
	require.Error(t, err)
}

func TestValidateOrderInputRejectsRestartForBalanceOrder(t *testing.T) {
	t.Parallel()
	svc := &PaymentService{}

	_, err := svc.validateOrderInput(context.Background(), CreateOrderRequest{
		OrderType:          payment.OrderTypeBalance,
		SubscriptionAction: payment.SubscriptionActionRestart,
	}, &PaymentConfig{})

	require.Error(t, err)
}

func TestValidateSubOrderRestartRequiresLimitedActiveOwnedSubscription(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(73).
		SetName("Restart plan").
		SetPrice(20).
		SetValidityDays(30).
		Save(ctx)
	require.NoError(t, err)

	dailyLimit := 10.0
	groupRepo := &subscriptionGroupRepoStub{group: &Group{
		ID: 73, Status: payment.EntityStatusActive,
		SubscriptionType: SubscriptionTypeSubscription, DailyLimitUSD: &dailyLimit,
	}}
	subRepo := newSubscriptionUserSubRepoStub()
	subRepo.seed(&UserSubscription{
		ID: 79, UserID: 83, GroupID: 73,
		Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	subscriptionSvc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	svc := &PaymentService{
		configService:   &PaymentConfigService{entClient: client},
		groupRepo:       groupRepo,
		subscriptionSvc: subscriptionSvc,
	}

	_, err = svc.validateSubOrder(ctx, CreateOrderRequest{
		UserID: 83, PlanID: plan.ID, SubscriptionAction: payment.SubscriptionActionRestart,
	})
	require.NoError(t, err)

	_, err = svc.validateSubOrder(ctx, CreateOrderRequest{
		UserID: 84, PlanID: plan.ID, SubscriptionAction: payment.SubscriptionActionRestart,
	})
	require.Error(t, err, "a user must not restart another user's subscription")

	groupRepo.group.DailyLimitUSD = nil
	_, err = svc.validateSubOrder(ctx, CreateOrderRequest{
		UserID: 83, PlanID: plan.ID, SubscriptionAction: payment.SubscriptionActionRestart,
	})
	require.Error(t, err, "unlimited plans must not expose immediate reset")
}

func TestPaymentOrderAllowsOnlyOneActiveRestartPerUserGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("restart-order-%d@example.com", time.Now().UnixNano())).
		SetPasswordHash("hash").
		SetUsername("restart-order-user").
		Save(ctx)
	require.NoError(t, err)

	createOrder := func(outTradeNo, action string) error {
		_, err := client.PaymentOrder.Create().
			SetUserID(user.ID).
			SetUserEmail(user.Email).
			SetUserName(user.Username).
			SetAmount(20).
			SetPayAmount(20).
			SetRechargeCode("PAY-" + outTradeNo).
			SetOutTradeNo(outTradeNo).
			SetPaymentType(payment.TypeAlipay).
			SetPaymentTradeNo("").
			SetOrderType(payment.OrderTypeSubscription).
			SetSubscriptionAction(action).
			SetSubscriptionGroupID(73).
			SetSubscriptionDays(30).
			SetStatus(OrderStatusPending).
			SetExpiresAt(time.Now().Add(30 * time.Minute)).
			SetClientIP("127.0.0.1").
			SetSrcHost("app.example.com").
			Save(ctx)
		return err
	}

	require.NoError(t, createOrder("restart-order-1", payment.SubscriptionActionRestart))
	err = createOrder("restart-order-2", payment.SubscriptionActionRestart)
	require.Error(t, err)
	require.True(t, dbent.IsConstraintError(err))
	require.NoError(t, createOrder("extend-order-1", payment.SubscriptionActionExtend))
}

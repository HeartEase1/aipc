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

func (s *subscriptionUserSubRepoStub) GetActiveByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub, err := s.GetByUserIDAndGroupID(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}
	if sub.Status != SubscriptionStatusActive || !sub.ExpiresAt.After(time.Now()) {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
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

func TestPaidSubscriptionRestartResetsUsageWithoutExtendingOldTermOrRepeatingOnCallback(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	user, err := client.User.Create().
		SetEmail(fmt.Sprintf("restart-paid-%d@example.com", time.Now().UnixNano())).
		SetPasswordHash("hash").SetUsername("restart-paid-user").Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).
		SetAmount(80).SetPayAmount(80).SetRechargeCode("PAY-RESTART").
		SetOutTradeNo(fmt.Sprintf("restart-paid-%d", time.Now().UnixNano())).
		SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo("paid-trade").
		SetOrderType(payment.OrderTypeSubscription).
		SetSubscriptionAction(payment.SubscriptionActionRestart).
		SetSubscriptionGroupID(7).SetSubscriptionDays(30).
		SetStatus(OrderStatusPaid).SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").SetSrcHost("app.example.com").Save(ctx)
	require.NoError(t, err)

	oldExpiry := time.Now().Add(20 * 24 * time.Hour)
	dailyStart := time.Now().Add(-3 * time.Hour)
	subRepo := newSubscriptionUserSubRepoStub()
	subRepo.seed(&UserSubscription{
		ID: 79, UserID: order.UserID, GroupID: *order.SubscriptionGroupID,
		StartsAt: time.Now().Add(-10 * 24 * time.Hour), ExpiresAt: oldExpiry,
		Status: SubscriptionStatusActive, DailyUsageUSD: 8, WeeklyUsageUSD: 12,
		MonthlyUsageUSD: 20, DailyWindowStart: &dailyStart,
	})
	groupRepo := &subscriptionGroupRepoStub{group: &Group{
		ID: *order.SubscriptionGroupID, Status: payment.EntityStatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
	}}
	svc := &PaymentService{
		entClient: client, groupRepo: groupRepo,
		subscriptionSvc: NewSubscriptionService(groupRepo, subRepo, nil, nil, nil),
	}

	require.NoError(t, svc.ExecuteSubscriptionFulfillment(ctx, order.ID))
	reset, err := subRepo.GetByUserIDAndGroupID(ctx, order.UserID, *order.SubscriptionGroupID)
	require.NoError(t, err)
	require.Equal(t, 0.0, reset.DailyUsageUSD)
	require.Equal(t, 0.0, reset.WeeklyUsageUSD)
	require.Equal(t, 0.0, reset.MonthlyUsageUSD)
	require.WithinDuration(t, time.Now(), reset.StartsAt, time.Minute)
	require.WithinDuration(t, reset.StartsAt.AddDate(0, 0, *order.SubscriptionDays), reset.ExpiresAt, time.Second)
	require.True(t, reset.ExpiresAt.After(oldExpiry))

	require.NoError(t, svc.HandlePaymentNotification(ctx, &payment.PaymentNotification{
		OrderID: order.OutTradeNo, TradeNo: "duplicate-callback", Amount: order.PayAmount,
		Status: payment.NotificationStatusSuccess,
	}, payment.TypeAlipay))
	again, err := subRepo.GetByUserIDAndGroupID(ctx, order.UserID, *order.SubscriptionGroupID)
	require.NoError(t, err)
	require.Equal(t, reset.ExpiresAt, again.ExpiresAt, "replayed payment must not restart a second time")
}

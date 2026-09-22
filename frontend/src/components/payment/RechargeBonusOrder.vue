<template>
 <div v-if="bonus" class="rounded-lg border border-amber-200 p-3 text-sm dark:border-amber-900">
  <p class="font-semibold">{{ bonus.title }} · 充值赠送</p>
  <dl class="mt-2 grid grid-cols-2 gap-2">
   <dt>充值面额</dt><dd>{{ order.currency }} {{ order.original_amount }}</dd>
   <dt>折扣减免</dt><dd>{{ order.currency }} {{ order.discount_amount || 0 }}</dd>
   <dt>实际支付</dt><dd>{{ order.currency }} {{ order.pay_amount }}</dd>
   <dt>换算倍率</dt><dd>1 : {{ bonus.multiplier }}</dd>
   <dt>赠送比例</dt><dd>{{ bonus.percent }}%</dd>
   <dt>本金到账余额</dt><dd>${{ order.amount.toFixed(2) }}</dd>
   <dt>{{ bonus.confirmed ? (bonus.credited ? '赠送到账余额' : '待发放赠送余额') : '预计赠送余额' }}</dt><dd>${{ gift.toFixed(2) }}</dd>
   <dt>{{ bonus.confirmed ? '合计到账余额' : '预计合计到账余额' }}</dt><dd>${{ (order.amount + (bonus.confirmed && !bonus.credited ? 0 : gift)).toFixed(2) }}</dd>
   <dt>已收回赠送余额</dt><dd>${{ Number(bonus.refunded || 0).toFixed(2) }}</dd>
  </dl>
  <p class="mt-2 text-xs text-gray-500">{{ bonus.reason || '首笔资格以支付成功确认为准。获赠订单整笔不参与邀请返利。' }}</p>
 </div>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import type {PaymentOrder} from '@/types/payment'
const props=defineProps<{order:PaymentOrder}>()
const bonus=computed(()=>props.order.pricing?.bonus)
const gift=computed(()=>bonus.value?.confirmed ? (bonus.value.awarded ? Number(bonus.value.expected) : 0) : Number(bonus.value?.preview ?? bonus.value?.expected ?? 0))
</script>

<template>
  <div v-if="source" class="mt-1 flex flex-wrap items-center gap-1 text-xs">
    <span class="rounded border border-primary-200 bg-primary-50 px-1.5 py-0.5 text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300">{{ t(`balanceMarketing.sources.${source}`) }}</span>
    <span v-if="order.discount_amount && order.discount_amount > 0" class="text-gray-500 dark:text-gray-400">{{ t('balanceMarketing.discount') }}: {{ currencySymbol(order.currency) }}{{ order.discount_amount.toFixed(2) }}</span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types/payment'
import { currencySymbol } from './currency'
const props = defineProps<{ order: PaymentOrder }>()
const { t } = useI18n()
const source = computed(() => props.order.order_type === 'balance' && ['first_recharge', 'campaign', 'membership'].includes(props.order.discount_source || '') ? props.order.discount_source : '')
</script>

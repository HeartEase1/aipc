<template>
  <div class="mt-4 space-y-3 border-t border-gray-200 pt-3 text-sm dark:border-dark-600">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span class="font-semibold text-primary-700 dark:text-primary-300">{{ summary.current_tier || t('payment.membership.defaultTier') }}</span>
      <button class="btn btn-secondary !px-3 !py-1 text-xs" @click="showRules = true"><Icon name="infoCircle" size="sm" />{{ t('balanceMarketing.rules') }}</button>
    </div>
    <p>{{ t('balanceMarketing.rollingPaid') }}: <strong>{{ amountOnly(summary.current_amount) }}</strong></p>
    <template v-if="summary.enabled">
      <div class="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600" role="progressbar" :aria-valuenow="Number(summary.progress_percent)" aria-valuemin="0" aria-valuemax="100" :aria-label="t('balanceMarketing.progress')">
        <div class="h-full rounded-full bg-primary-500" :style="{ width: `${Math.min(100, Math.max(0, Number(summary.progress_percent)))}%` }" />
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ summary.next_tier ? t('balanceMarketing.nextTier', { amount: amountOnly(summary.amount_to_next || '0'), tier: summary.next_tier }) : t('payment.membership.maxTier') }}</p>
      <p>{{ t('balanceMarketing.memberBenefit', { percent: Number(summary.current_discount_percent) }) }}</p>
    </template>
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(summary.first_recharge_eligible ? 'balanceMarketing.firstEligible' : 'balanceMarketing.firstUnavailable') }}</p>
    <BaseDialog :show="showRules" :title="t('balanceMarketing.rules')" size="md" @close="showRules = false">
      <div class="space-y-4 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <p>{{ t('balanceMarketing.ruleWindow', { currency: summary.settlement_currency, hours: summary.rules?.window_hours || 720 }) }}</p>
        <p>{{ t('balanceMarketing.ruleExclusions') }}</p>
        <p>{{ t('balanceMarketing.rulePriority') }}</p>
        <p>{{ t('balanceMarketing.ruleFirst') }}</p>
        <p>{{ t('balanceMarketing.ruleCredit') }}</p>
        <p>{{ t('balanceMarketing.ruleAffiliate', { percent: summary.rules?.affiliate_commission_rate || '0' }) }}</p>
        <div v-if="summary.rules?.tiers.length" class="overflow-x-auto">
          <table class="w-full text-left text-sm"><thead><tr><th class="p-2">{{ t('balanceMarketing.tier') }}</th><th class="p-2">{{ t('balanceMarketing.threshold') }}</th><th class="p-2">{{ t('balanceMarketing.reduction') }}</th></tr></thead>
            <tbody><tr v-for="tier in summary.rules.tiers" :key="tier.name" class="border-t dark:border-dark-600"><td class="p-2">{{ tier.name }}</td><td class="p-2">{{ money(tier.threshold_amount) }}</td><td class="p-2">{{ Number(tier.discount_percent) }}%</td></tr></tbody>
          </table>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MembershipSummary } from '@/api/payment'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ summary: MembershipSummary }>()
const { t } = useI18n()
const showRules = ref(false)
const amountOnly = (value: string) => Number(value).toFixed(2)
const money = (value: string) => `${props.summary.settlement_currency} ${amountOnly(value)}`
</script>

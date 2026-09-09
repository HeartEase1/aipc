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
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(summary.first_recharge_status ? `balanceMarketing.firstStates.${summary.first_recharge_status}` : summary.first_recharge_eligible ? 'balanceMarketing.firstEligible' : 'balanceMarketing.firstUnavailable') }}</p>
    <BaseDialog :show="showRules" :title="t('balanceMarketing.rules')" size="md" @close="showRules = false">
      <div class="space-y-4 text-sm leading-6 text-gray-700 dark:text-gray-300">
        <section class="space-y-3">
          <h3 class="font-semibold text-gray-900 dark:text-gray-100">{{ t('balanceMarketing.sources.first_recharge') }}</h3>
          <div v-for="offer in summary.rules?.first_recharge_offers || []" :key="offer.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
            <p class="break-words font-medium">{{ offer.name }}</p>
            <p class="font-semibold text-primary-700 dark:text-primary-300">{{ t('balanceMarketing.firstOfferDiscount', { percent: Number(offer.discount_percent), fold: Number(((100 - Number(offer.discount_percent)) / 10).toFixed(5)) }) }}</p>
            <dl class="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs">
              <dt>{{ t('balanceMarketing.minimum') }}</dt><dd>{{ offer.min_amount ? money(offer.min_amount) : t('balanceMarketing.unlimited') }}</dd>
              <dt>{{ t('balanceMarketing.maximum') }}</dt><dd>{{ offer.max_amount ? money(offer.max_amount) : t('balanceMarketing.unlimited') }}</dd>
              <dt>{{ t('balanceMarketing.cap') }}</dt><dd>{{ offer.max_discount_amount ? money(offer.max_discount_amount) : t('balanceMarketing.unlimited') }}</dd>
              <dt>{{ t('balanceMarketing.start') }}</dt><dd>{{ offer.starts_at ? offerTime(offer.starts_at) : t('balanceMarketing.unlimited') }}</dd>
              <dt>{{ t('balanceMarketing.end') }}</dt><dd>{{ offer.ends_at ? offerTime(offer.ends_at) : t('balanceMarketing.unlimited') }}</dd>
            </dl>
          </div>
          <p v-if="!summary.rules?.first_recharge_offers?.length" class="text-xs text-gray-500 dark:text-gray-400">{{ t('balanceMarketing.noFirstOffer') }}</p>
          <p>{{ t('balanceMarketing.ruleFirst') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('balanceMarketing.offerQuoteHint') }}</p>
        </section>
        <p>{{ t('balanceMarketing.ruleWindow', { currency: summary.settlement_currency, hours: summary.rules?.window_hours || 720 }) }}</p>
        <p>{{ t('balanceMarketing.ruleExclusions') }}</p>
        <p>{{ t('balanceMarketing.rulePriority') }}</p>
        <p>{{ t('balanceMarketing.ruleCredit') }}</p>
        <p>{{ t('balanceMarketing.ruleAffiliate') }}</p>
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
const { t, locale } = useI18n()
const showRules = ref(false)
const amountOnly = (value: string) => Number(value).toFixed(2)
const money = (value: string) => `${props.summary.settlement_currency} ${amountOnly(value)}`
const offerTime = (value: string) => new Date(value).toLocaleString(locale.value, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', timeZoneName: 'short' })
</script>

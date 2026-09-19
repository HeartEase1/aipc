<template>
  <section
    v-if="isPercentageGrant"
    data-testid="benefit-calculation-details"
    class="border-y border-amber-100 py-4 dark:border-amber-900/40"
  >
    <div class="mb-2 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
      <Icon name="calculator" size="sm" class="text-amber-600 dark:text-amber-400" />
      <span>{{ t('benefits.calculation.title') }}</span>
    </div>

    <dl class="divide-y divide-gray-100 text-sm dark:divide-dark-700">
      <div
        v-for="line in calculationLines"
        :key="line.key"
        class="grid gap-1.5 py-2.5 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-4"
      >
        <dt class="font-medium text-gray-700 dark:text-gray-300">
          {{ t(line.labelKey) }}
        </dt>
        <dd
          class="flex flex-wrap items-baseline gap-1.5 tabular-nums text-gray-600 dark:text-gray-300 sm:justify-end"
        >
          <span>${{ line.base }}</span>
          <span class="text-gray-400">×</span>
          <span>{{ line.percentage }}</span>
          <span class="text-gray-400">=</span>
          <strong class="text-base font-semibold text-emerald-600 dark:text-emerald-400">
            +${{ line.calculatedAmount }}
          </strong>
        </dd>
      </div>
    </dl>

    <dl class="mt-2 border-t border-gray-100 pt-3 text-sm dark:border-dark-700">
      <template v-if="hasRuleAdjustment">
        <div class="flex items-center justify-between gap-4 py-1">
          <dt class="text-gray-500 dark:text-gray-400">
            {{ t('benefits.calculation.calculatedAmount') }}
          </dt>
          <dd class="font-medium tabular-nums text-gray-700 dark:text-gray-300">
            ${{ calculatedTotal }}
          </dd>
        </div>
        <div class="flex items-center justify-between gap-4 py-1">
          <dt class="text-gray-500 dark:text-gray-400">
            {{ t('benefits.calculation.ruleAdjustment') }}
          </dt>
          <dd class="font-medium tabular-nums text-gray-700 dark:text-gray-300">
            {{ formattedAdjustment }}
          </dd>
        </div>
      </template>

      <div
        class="mt-1 flex items-center justify-between gap-4 rounded-lg bg-emerald-50 px-3 py-2.5 dark:bg-emerald-900/20"
      >
        <dt class="font-medium text-emerald-800 dark:text-emerald-200">
          {{ t('benefits.calculation.actualAmount') }}
        </dt>
        <dd class="text-lg font-semibold tabular-nums text-emerald-700 dark:text-emerald-300">
          +${{ actualAmount }}
        </dd>
      </div>
    </dl>

    <div class="mt-4 border-t border-gray-100 pt-3 dark:border-dark-700">
      <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
        {{ t('benefits.calculation.window') }}
      </p>
      <dl class="mt-2 grid grid-cols-1 gap-2 text-sm sm:grid-cols-2 sm:gap-4">
        <div>
          <dt class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('benefits.calculation.windowStart') }}
          </dt>
          <dd class="mt-0.5 font-medium tabular-nums text-gray-800 dark:text-gray-200">
            {{ formattedWindowStart }}
          </dd>
        </div>
        <div>
          <dt class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('benefits.calculation.windowEnd') }}
          </dt>
          <dd class="mt-0.5 font-medium tabular-nums text-gray-800 dark:text-gray-200">
            {{ formattedWindowEnd }}
          </dd>
        </div>
      </dl>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserBenefitGrant } from '@/api/benefitGrants'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  grant: UserBenefitGrant
}>()

const { t } = useI18n()

const DECIMAL_SCALE = 100_000_000n
const PERCENT_DIVISOR = 100n * DECIMAL_SCALE

const isPercentageGrant = computed(() => props.grant.grant_mode === 'percentage_24h')
const formattedWindowStart = computed(() => (
  props.grant.window_start ? formatDateTime(props.grant.window_start) : t('common.notAvailable')
))
const formattedWindowEnd = computed(() => (
  props.grant.window_end ? formatDateTime(props.grant.window_end) : t('common.notAvailable')
))

const calculationLines = computed(() => {
  const balanceBase = props.grant.include_subscription
    ? props.grant.balance_base_cost
    : props.grant.base_cost
  const lines = [buildCalculationLine(
    'balance',
    'benefits.calculation.balanceSpending',
    balanceBase,
    props.grant.percentage,
  )]

  if (props.grant.include_subscription) {
    lines.push(buildCalculationLine(
      'subscription',
      'benefits.calculation.subscriptionSpending',
      props.grant.subscription_base_cost,
      props.grant.subscription_percentage,
    ))
  }
  return lines
})

const calculatedTotalUnits = computed(() => calculationLines.value.reduce(
  (total, line) => total + line.calculatedUnits,
  0n,
))
const actualAmountUnits = computed(() => parseDecimalUnits(props.grant.amount))
const adjustmentUnits = computed(() => actualAmountUnits.value - calculatedTotalUnits.value)
const hasRuleAdjustment = computed(() => adjustmentUnits.value !== 0n)
const calculatedTotal = computed(() => formatMoneyUnits(calculatedTotalUnits.value))
const actualAmount = computed(() => formatMoneyUnits(actualAmountUnits.value))
const formattedAdjustment = computed(() => {
  const value = adjustmentUnits.value
  const sign = value > 0n ? '+' : '-'
  return `${sign}$${formatMoneyUnits(value < 0n ? -value : value)}`
})

function buildCalculationLine(
  key: string,
  labelKey: string,
  base?: string,
  percentage?: string,
) {
  const calculatedUnits = calculatePercentageAmount(base, percentage)
  return {
    key,
    labelKey,
    base: formatMoneyUnits(parseDecimalUnits(base)),
    percentage: formatPercentage(percentage),
    calculatedAmount: formatMoneyUnits(calculatedUnits),
    calculatedUnits,
  }
}

function parseDecimalUnits(value?: string): bigint {
  const match = /^([+-]?)(\d+)(?:\.(\d+))?$/.exec(value?.trim() || '')
  if (!match) return 0n

  const sign = match[1] === '-' ? -1n : 1n
  const fraction = (match[3] || '').padEnd(8, '0').slice(0, 8)
  return sign * (BigInt(match[2]) * DECIMAL_SCALE + BigInt(fraction))
}

function calculatePercentageAmount(base?: string, percentage?: string): bigint {
  const product = parseDecimalUnits(base) * parseDecimalUnits(percentage)
  if (product <= 0n) return 0n
  return (product + PERCENT_DIVISOR / 2n) / PERCENT_DIVISOR
}

function formatMoneyUnits(value: bigint): string {
  const whole = value / DECIMAL_SCALE
  let fraction = (value % DECIMAL_SCALE).toString().padStart(8, '0')
  while (fraction.length > 2 && fraction.endsWith('0')) {
    fraction = fraction.slice(0, -1)
  }
  return `${whole}.${fraction}`
}

function formatPercentage(value?: string): string {
  if (!value) return t('common.notAvailable')
  return `${value.replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1')}%`
}
</script>

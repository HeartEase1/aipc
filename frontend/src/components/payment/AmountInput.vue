<template>
  <div class="space-y-5">
    <!-- Custom Amount Input -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-sm font-semibold text-gray-400 dark:text-dark-500">
          {{ currencyLabel }}
        </span>
        <input
          type="text"
          inputmode="decimal"
          :value="customText"
          :placeholder="placeholderText"
          class="input h-16 w-full pl-20 pr-4 text-xl font-semibold sm:text-2xl"
          @input="handleInput"
        />
      </div>
    </div>

    <!-- Quick Amount Buttons -->
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.quickAmounts') }}
      </label>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          :class="[
            'min-h-12 rounded-xl border-2 px-4 py-3 text-center font-medium transition-colors',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/40 dark:text-primary-300'
              : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500',
          ]"
          @click="selectAmount(amt)"
        >
          <span :data-testid="`quick-amount-${amt}`" class="flex flex-col items-center gap-0.5">
            <span class="inline-flex items-center gap-1">
              <span>{{ amt }}</span>
              <span
                v-if="bonusFor(amt)"
                data-testid="quick-amount-bonus-badge"
                class="rounded bg-primary-100 px-1 text-xs text-primary-700 dark:bg-primary-900/40 dark:text-primary-300"
              >{{ bonusLabel(bonusFor(amt)!) }}</span>
            </span>
            <span v-if="bonusTiers?.length" data-testid="quick-amount-credited" class="text-xs text-gray-500 dark:text-gray-400">
              {{ creditedLabel(amt) }}
            </span>
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  currency?: string
  bonusTiers?: Array<{ min_amount: number; bonus_percent: number }>
  bonusMode?: 'bonus' | 'discount'
  multiplier?: number
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  currency: 'USD',
  bonusMode: 'bonus',
  multiplier: 1,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')
const currencyLabel = computed(() => props.currency.trim().toUpperCase() || 'USD')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

function bonusFor(amount: number) {
  return props.bonusTiers?.filter((tier) => amount >= tier.min_amount)
    .sort((a, b) => b.min_amount - a.min_amount)[0]
}

function creditedAmount(amount: number) {
  const tier = bonusFor(amount)
  if (!tier) return amount * (props.multiplier ?? 1)
  if (props.bonusMode === 'discount') return amount * (1 - tier.bonus_percent / 100)
  return amount * (props.multiplier ?? 1) * (1 + tier.bonus_percent / 100)
}

function bonusLabel(tier: { bonus_percent: number }) {
  return props.bonusMode === 'discount' ? `${tier.bonus_percent}% OFF` : `+${tier.bonus_percent}%`
}

function creditedLabel(amount: number) {
  const symbol = currencyLabel.value === 'USD' ? '$' : currencyLabel.value === 'CNY' ? '¥' : `${currencyLabel.value} `
  return `${symbol}${creditedAmount(amount).toFixed(2)}`
}

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>

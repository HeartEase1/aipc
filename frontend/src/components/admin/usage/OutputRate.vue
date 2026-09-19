<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import UsageInfoPopover from '@/components/common/UsageInfoPopover.vue'
import { formatGenerationOutputRate, formatOutputRate } from '@/utils/outputRate'
defineProps<{ outputTokens?: number | null; durationMs?: number | null; firstTokenMs?: number | null }>()
const { t } = useI18n()
</script>
<template>
  <UsageInfoPopover :label="t('usage.outputRateDetails')">
    <template #trigger>
      <span class="text-gray-400">{{ t('usage.outputRate') }}</span>
      <span data-testid="output-rate" class="font-medium tabular-nums text-violet-600 dark:text-violet-300">{{ formatOutputRate(outputTokens, durationMs) }}</span>
    </template>
    <p class="mb-2 font-semibold">{{ t('usage.outputRateDetails') }}</p>
    <p>{{ t('usage.outputRateAverage') }}: <strong>{{ formatOutputRate(outputTokens, durationMs) }}</strong></p>
    <p class="text-gray-500">{{ t('usage.outputRateAverageFormula') }}</p>
    <p class="mt-2">{{ t('usage.outputRateGeneration') }}: <strong>{{ formatGenerationOutputRate(outputTokens, durationMs, firstTokenMs) }}</strong></p>
    <p class="text-gray-500">{{ t('usage.outputRateGenerationFormula') }}</p>
    <p class="mt-2 text-gray-500">{{ t('usage.outputRateGenerationWarning') }}</p>
  </UsageInfoPopover>
</template>

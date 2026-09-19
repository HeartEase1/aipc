<template>
  <section
    class="leaderboard-share-overview border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800 sm:p-5"
    data-testid="leaderboard-share-overview"
    :aria-labelledby="titleId"
  >
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex items-start gap-3">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-700 shadow-sm ring-1 ring-primary-100 dark:bg-primary-500/10 dark:text-primary-300 dark:ring-primary-500/20">
          <Icon name="chart" size="sm" />
        </span>
        <div>
          <h3 :id="titleId" class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('leaderboard.shareOverview') }}
          </h3>
          <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ description }}
          </p>
        </div>
      </div>
      <span class="w-fit shrink-0 rounded-full border border-gray-200 bg-gray-50 px-3 py-1 text-xs font-semibold text-gray-600 dark:border-dark-600 dark:bg-dark-900/60 dark:text-dark-300">
        {{ t('leaderboard.top20') }} + {{ t('leaderboard.otherUsers') }}
      </span>
    </div>

    <DoughnutDistributionLayout
      v-if="hasShareData"
      class="mt-4"
      :data="chartData"
      :options="chartOptions"
      :chart-label="t('leaderboard.shareOverview')"
    >
      <table class="w-full table-fixed text-[11px] sm:table-auto sm:text-xs">
        <thead>
          <tr class="text-gray-500 dark:text-dark-400">
            <th class="w-[44%] pb-2 text-left font-medium sm:w-auto">{{ t('leaderboard.user') }}</th>
            <th class="w-[36%] pb-2 text-right font-medium sm:w-auto">{{ valueHeading }}</th>
            <th class="w-[20%] pb-2 text-right font-medium sm:w-auto">{{ t('leaderboard.share') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(item, index) in items"
            :key="item.key"
            class="border-t border-gray-100 dark:border-dark-700"
            :class="item.isOther ? 'bg-gray-50/70 dark:bg-dark-700/20' : ''"
            :data-share-rank="item.rank ?? 'other'"
          >
            <td class="max-w-40 py-1.5 pr-2 sm:pr-3">
              <div class="flex min-w-0 items-center gap-2">
                <span
                  class="h-2.5 w-2.5 shrink-0 rounded-full"
                  :style="{ backgroundColor: segmentColor(item, index) }"
                ></span>
                <span class="truncate font-medium text-gray-900 dark:text-white" :title="item.displayName">
                  {{ item.displayName }}
                </span>
              </div>
            </td>
            <td class="py-1.5 text-right tabular-nums text-gray-600 dark:text-dark-300">
              {{ item.valueLabel }}
            </td>
            <td class="py-1.5 pl-2 text-right font-semibold tabular-nums text-gray-900 dark:text-white sm:pl-3">
              {{ item.percentageLabel }}
            </td>
          </tr>
        </tbody>
      </table>
    </DoughnutDistributionLayout>

    <div v-else class="mt-4 flex h-48 items-center justify-center text-sm text-gray-500 dark:text-dark-400">
      {{ t('leaderboard.empty') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChartData, ChartOptions } from 'chart.js'
import DoughnutDistributionLayout from '@/components/charts/DoughnutDistributionLayout.vue'
import { DISTRIBUTION_CHART_COLORS, DISTRIBUTION_OTHER_COLOR } from '@/components/charts/distributionChart'
import Icon from '@/components/icons/Icon.vue'

interface LeaderboardShareItem {
  key: string
  rank?: number
  displayName: string
  percentage: number
  percentageLabel: string
  valueLabel: string
  isOther?: boolean
}

const props = defineProps<{
  items: LeaderboardShareItem[]
  description: string
  valueHeading: string
}>()

const { t } = useI18n()
const titleId = `leaderboard-share-${useId()}`

function segmentColor(item: LeaderboardShareItem, index: number): string {
  return item.isOther
    ? DISTRIBUTION_OTHER_COLOR
    : DISTRIBUTION_CHART_COLORS[index % DISTRIBUTION_CHART_COLORS.length]!
}

const hasShareData = computed(() => props.items.some((item) => item.percentage > 0))

const chartData = computed<ChartData<'doughnut'>>(() => ({
  labels: props.items.map((item) => item.displayName),
  datasets: [{
    data: props.items.map((item) => item.percentage),
    backgroundColor: props.items.map((item, index) => segmentColor(item, index)),
    borderWidth: 0
  }]
}))

const chartOptions: ChartOptions<'doughnut'> = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    },
    tooltip: {
      displayColors: true,
      callbacks: {
        label(context) {
          const item = props.items[context.dataIndex]
          return item ? `${item.displayName}: ${item.valueLabel} (${item.percentageLabel})` : ''
        }
      }
    }
  }
}
</script>

<style scoped>
.leaderboard-share-overview {
  border-radius: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.92),
    0 4px 0 rgb(209 213 219 / 0.78),
    0 18px 30px -24px rgb(15 23 42 / 0.62);
}

:global(.dark) .leaderboard-share-overview {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 4px 0 rgb(2 6 23 / 0.9),
    0 18px 30px -22px rgb(0 0 0 / 0.95);
}
</style>

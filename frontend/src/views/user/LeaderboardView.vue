<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
            {{ t('leaderboard.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.description') }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.privacyNote') }}
          </p>
        </div>
        <label class="flex cursor-pointer items-center gap-3 text-sm text-gray-700 dark:text-dark-200">
          <span>{{ t('leaderboard.participation') }}</span>
          <input
            v-model="participating"
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            :disabled="loading || savingParticipation"
            @change="saveParticipation"
          />
        </label>
      </header>

      <div class="flex flex-wrap gap-1 rounded-md border border-gray-200 p-1 dark:border-dark-700" role="tablist">
        <button
          v-for="option in periodOptions"
          :key="option.value"
          type="button"
          class="min-w-16 rounded px-3 py-1.5 text-sm transition-colors"
          :class="period === option.value ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700'"
          :aria-selected="period === option.value"
          @click="period = option.value"
        >
          {{ option.label }}
        </button>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="data">
        <div class="grid gap-3 sm:grid-cols-3">
          <div v-for="item in summaryItems" :key="item.label" class="border border-gray-200 p-4 dark:border-dark-700">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ item.label }}</p>
            <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          </div>
        </div>

        <div class="flex border-b border-gray-200 dark:border-dark-700" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="border-b-2 px-4 py-2.5 text-sm font-medium"
            :class="activeTab === tab.value ? 'border-primary-600 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-white'"
            :aria-selected="activeTab === tab.value"
            @click="activeTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-if="!data.participating" class="border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('leaderboard.notParticipating') }}
        </div>

        <section v-if="activeTab === 'usage'" class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-sm">
            <thead class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.rank') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.user') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.requests') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.tokens') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.cost') }}</th>
              </tr>
            </thead>
            <tbody>
              <UsageRow v-if="data.usage.current" :entry="data.usage.current" :self="true" />
              <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                <td colspan="5" class="px-3 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
              </tr>
              <UsageRow v-for="entry in data.usage.entries" :key="entry.rank" :entry="entry" />
              <tr v-if="data.usage.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                <td colspan="5" class="px-3 py-8 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section v-else-if="activeTab === 'consumption'" class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-sm">
            <thead class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.rank') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.user') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.requests') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.tokens') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.cost') }}</th>
              </tr>
            </thead>
            <tbody>
              <UsageRow v-if="data.consumption.current" :entry="data.consumption.current" :self="true" />
              <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                <td colspan="5" class="px-3 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
              </tr>
              <UsageRow v-for="entry in data.consumption.entries" :key="entry.rank" :entry="entry" />
              <tr v-if="data.consumption.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                <td colspan="5" class="px-3 py-8 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </section>

        <section v-else class="overflow-x-auto">
          <table class="w-full min-w-[680px] text-left text-sm">
            <thead class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.rank') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('leaderboard.user') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.invitedUsers') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.rebateCount') }}</th>
                <th class="px-3 py-2 text-right font-medium">{{ t('leaderboard.rebateAmount') }}</th>
              </tr>
            </thead>
            <tbody>
              <RebateRow v-if="data.rebate.current" :entry="data.rebate.current" :self="true" />
              <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                <td colspan="5" class="px-3 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
              </tr>
              <RebateRow v-for="entry in data.rebate.entries" :key="entry.rank" :entry="entry" />
              <tr v-if="data.rebate.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                <td colspan="5" class="px-3 py-8 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, type PropType, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { leaderboardAPI, type LeaderboardPeriod, type LeaderboardRebateEntry, type LeaderboardResponse, type LeaderboardUsageEntry } from '@/api/leaderboard'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const period = ref<LeaderboardPeriod>('24h')
const activeTab = ref<'usage' | 'consumption' | 'rebate'>('usage')
const data = ref<LeaderboardResponse | null>(null)
const loading = ref(false)
const savingParticipation = ref(false)
const participating = ref(true)

const periodOptions = computed(() => [
  { value: '24h' as const, label: t('leaderboard.period24h') },
  { value: '72h' as const, label: t('leaderboard.period72h') },
  { value: '7d' as const, label: t('leaderboard.period7d') },
  { value: '30d' as const, label: t('leaderboard.period30d') }
])

const tabs = computed(() => [
  { value: 'usage' as const, label: t('leaderboard.usageTab') },
  { value: 'consumption' as const, label: t('leaderboard.consumptionTab') },
  { value: 'rebate' as const, label: t('leaderboard.rebateTab') }
])

const summaryItems = computed(() => {
  if (!data.value) return []
  if (activeTab.value === 'rebate') {
    return [
      { label: t('leaderboard.invitedUsers'), value: formatNumber(data.value.rebate.summary.invited_users) },
      { label: t('leaderboard.rebateCount'), value: formatNumber(data.value.rebate.summary.rebate_count) },
      { label: t('leaderboard.rebateAmount'), value: formatCurrency(data.value.rebate.summary.rebate_amount) }
    ]
  }
  const summary = activeTab.value === 'usage' ? data.value.usage.summary : data.value.consumption.summary
  return [
    { label: t('leaderboard.requests'), value: formatNumber(summary.request_count) },
    { label: t('leaderboard.tokens'), value: formatNumber(summary.total_tokens) },
    { label: t('leaderboard.cost'), value: formatCurrency(summary.actual_cost) }
  ]
})

function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value || 0)
}

async function load(): Promise<void> {
  loading.value = true
  try {
    data.value = await leaderboardAPI.getLeaderboard(period.value)
    participating.value = data.value.participating
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('leaderboard.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function saveParticipation(): Promise<void> {
  if (savingParticipation.value) return
  savingParticipation.value = true
  try {
    await leaderboardAPI.updateLeaderboardParticipation(participating.value)
    await load()
  } catch (error) {
    participating.value = !participating.value
    appStore.showError(extractApiErrorMessage(error, t('leaderboard.saveFailed')))
  } finally {
    savingParticipation.value = false
  }
}

const UsageRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardUsageEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('tr', { class: props.self ? 'border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20' : 'border-b border-gray-100 dark:border-dark-800' }, [
      h('td', { class: 'px-3 py-3 font-semibold text-primary-700 dark:text-primary-300' }, props.self
        ? [h('div', '0'), h('div', { class: 'text-xs font-normal' }, t('leaderboard.currentRank', { rank: props.entry.rank }))]
        : String(props.entry.rank)),
      h('td', { class: 'px-3 py-3 font-medium text-gray-900 dark:text-white' }, props.entry.display_name),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.request_count)),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.total_tokens)),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatCurrency(props.entry.actual_cost))
    ])
  }
})

const RebateRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardRebateEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('tr', { class: props.self ? 'border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20' : 'border-b border-gray-100 dark:border-dark-800' }, [
      h('td', { class: 'px-3 py-3 font-semibold text-primary-700 dark:text-primary-300' }, props.self
        ? [h('div', '0'), h('div', { class: 'text-xs font-normal' }, t('leaderboard.currentRank', { rank: props.entry.rank }))]
        : String(props.entry.rank)),
      h('td', { class: 'px-3 py-3 font-medium text-gray-900 dark:text-white' }, props.entry.display_name),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.invited_users)),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.rebate_count)),
      h('td', { class: 'px-3 py-3 text-right text-gray-700 dark:text-dark-300' }, formatCurrency(props.entry.rebate_amount))
    ])
  }
})

watch(period, () => { void load() }, { immediate: true })
</script>

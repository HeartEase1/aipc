<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex items-start gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-amber-100 text-amber-700 shadow-sm dark:bg-amber-500/15 dark:text-amber-300">
            <Icon name="trophy" size="lg" />
          </div>
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('leaderboard.title') }}
            </h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.description') }}
            </p>
            <p class="mt-1.5 flex items-center gap-1.5 text-xs text-gray-500 dark:text-dark-400">
              <Icon name="infoCircle" size="xs" class="shrink-0" />
              <span>{{ t('leaderboard.privacyNote') }}</span>
            </p>
          </div>
        </div>
      </header>

      <div class="flex flex-col gap-4 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.periodLabel') }}
          </p>
          <div class="inline-flex max-w-full overflow-x-auto rounded-lg bg-gray-100 p-1 dark:bg-dark-900" role="tablist">
            <button
              v-for="option in periodOptions"
              :key="option.value"
              type="button"
              class="min-w-16 rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="period === option.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'"
              :aria-selected="period === option.value"
              @click="period = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <label class="flex cursor-pointer items-center justify-between gap-5 rounded-lg bg-gray-50 px-3 py-2.5 dark:bg-dark-900/70 sm:min-w-64">
          <span>
            <span class="block text-sm font-medium text-gray-800 dark:text-dark-100">
              {{ t('leaderboard.participation') }}
            </span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.participationHint') }}
            </span>
          </span>
          <span class="relative inline-flex shrink-0">
            <input
              v-model="participating"
              type="checkbox"
              role="switch"
              class="peer sr-only"
              :disabled="loading || savingParticipation"
              @change="saveParticipation"
            />
            <span class="h-6 w-11 rounded-full bg-gray-300 transition-colors peer-checked:bg-primary-600 peer-disabled:cursor-not-allowed peer-disabled:opacity-50 dark:bg-dark-600"></span>
            <span class="pointer-events-none absolute left-1 top-1 h-4 w-4 rounded-full bg-white shadow-sm transition-transform peer-checked:translate-x-5"></span>
          </span>
        </label>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="data">
        <section aria-labelledby="leaderboard-summary-title">
          <div class="mb-3 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 id="leaderboard-summary-title" class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ activeTabMeta.label }} / {{ selectedPeriodLabel }}
              </h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.summaryScope') }}
              </p>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-3">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="relative min-h-28 overflow-hidden rounded-lg border border-gray-200 border-t-2 bg-white p-4 shadow-sm transition duration-200 hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-800"
              :class="item.borderClass"
            >
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="text-sm font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
                  <p class="mt-2 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ item.value }}</p>
                </div>
                <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg" :class="item.iconClass">
                  <Icon :name="item.icon" size="sm" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <div class="grid grid-cols-3 gap-1 rounded-lg bg-gray-100 p-1 dark:bg-dark-900" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="flex min-h-10 items-center justify-center gap-2 rounded-md px-2 py-2 text-sm font-medium transition-colors"
            :class="activeTab === tab.value ? tab.activeClass : 'text-gray-500 hover:bg-white/70 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-white'"
            :aria-selected="activeTab === tab.value"
            @click="activeTab = tab.value"
          >
            <Icon :name="tab.icon" size="sm" />
            <span>{{ tab.label }}</span>
          </button>
        </div>

        <div v-if="!data.participating" class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('leaderboard.notParticipating') }}
        </div>

        <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-2 border-b border-gray-200 bg-gray-50/80 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/50 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex items-start gap-2">
              <Icon :name="activeTabMeta.icon" size="sm" class="mt-0.5 shrink-0" :class="activeTabMeta.iconClass" />
              <p class="text-sm text-gray-600 dark:text-dark-300">{{ activeTabMeta.rule }}</p>
            </div>
            <span class="w-fit shrink-0 rounded-md border border-gray-200 bg-white px-2 py-1 text-xs font-semibold text-gray-600 shadow-sm dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300">
              {{ t('leaderboard.top20') }}
            </span>
          </div>

          <section v-if="activeTab === 'usage'" class="overflow-x-auto">
            <table class="w-full min-w-[680px] text-left text-sm">
              <thead class="border-b border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
                <tr>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.rank') }}</th>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.user') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.requests') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.tokens') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.cost') }}</th>
                </tr>
              </thead>
              <tbody>
                <UsageRow v-if="data.usage.current" :entry="data.usage.current" :self="true" />
                <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                  <td colspan="5" class="border-l-4 border-primary-500 px-4 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
                </tr>
                <UsageRow v-for="entry in data.usage.entries" :key="entry.rank" :entry="entry" />
                <tr v-if="data.usage.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                  <td colspan="5" class="px-4 py-10 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </section>

          <section v-else-if="activeTab === 'consumption'" class="overflow-x-auto">
            <table class="w-full min-w-[680px] text-left text-sm">
              <thead class="border-b border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
                <tr>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.rank') }}</th>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.user') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.requests') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.tokens') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.cost') }}</th>
                </tr>
              </thead>
              <tbody>
                <UsageRow v-if="data.consumption.current" :entry="data.consumption.current" :self="true" />
                <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                  <td colspan="5" class="border-l-4 border-primary-500 px-4 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
                </tr>
                <UsageRow v-for="entry in data.consumption.entries" :key="entry.rank" :entry="entry" />
                <tr v-if="data.consumption.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                  <td colspan="5" class="px-4 py-10 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </section>

          <section v-else class="overflow-x-auto">
            <table class="w-full min-w-[680px] text-left text-sm">
              <thead class="border-b border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
                <tr>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.rank') }}</th>
                  <th scope="col" class="px-4 py-3 font-medium">{{ t('leaderboard.user') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.invitedUsers') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.rebateCount') }}</th>
                  <th scope="col" class="px-4 py-3 text-right font-medium">{{ t('leaderboard.rebateAmount') }}</th>
                </tr>
              </thead>
              <tbody>
                <RebateRow v-if="data.rebate.current" :entry="data.rebate.current" :self="true" />
                <tr v-else-if="data.participating" class="border-b border-primary-200 bg-primary-50/70 dark:border-primary-900/40 dark:bg-primary-950/20">
                  <td colspan="5" class="border-l-4 border-primary-500 px-4 py-3 text-center text-sm text-primary-700 dark:text-primary-300">{{ t('leaderboard.notRanked') }}</td>
                </tr>
                <RebateRow v-for="entry in data.rebate.entries" :key="entry.rank" :entry="entry" />
                <tr v-if="data.rebate.entries.length === 0" class="border-b border-gray-100 dark:border-dark-800">
                  <td colspan="5" class="px-4 py-10 text-center text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </section>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, type PropType, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { leaderboardAPI, type LeaderboardPeriod, type LeaderboardRebateEntry, type LeaderboardResponse, type LeaderboardUsageEntry } from '@/api/leaderboard'
import { useAppStore } from '@/stores/app'
import { formatCurrency } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

type LeaderboardTabKey = 'usage' | 'consumption' | 'rebate'
type LeaderboardIcon = 'chart' | 'dollar' | 'gift' | 'document' | 'cube' | 'users'

interface LeaderboardTab {
  value: LeaderboardTabKey
  label: string
  icon: LeaderboardIcon
  rule: string
  activeClass: string
  iconClass: string
}

interface SummaryItem {
  label: string
  value: string
  icon: LeaderboardIcon
  borderClass: string
  iconClass: string
}

const { t } = useI18n()
const appStore = useAppStore()
const period = ref<LeaderboardPeriod>('24h')
const activeTab = ref<LeaderboardTabKey>('usage')
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

const selectedPeriodLabel = computed(() =>
  periodOptions.value.find((option) => option.value === period.value)?.label ?? t('leaderboard.period24h')
)

const tabs = computed<LeaderboardTab[]>(() => [
  {
    value: 'usage',
    label: t('leaderboard.usageTab'),
    icon: 'chart',
    rule: t('leaderboard.usageRule'),
    activeClass: 'bg-white text-blue-700 shadow-sm dark:bg-dark-700 dark:text-blue-300',
    iconClass: 'text-blue-600 dark:text-blue-400'
  },
  {
    value: 'consumption',
    label: t('leaderboard.consumptionTab'),
    icon: 'dollar',
    rule: t('leaderboard.consumptionRule'),
    activeClass: 'bg-white text-emerald-700 shadow-sm dark:bg-dark-700 dark:text-emerald-300',
    iconClass: 'text-emerald-600 dark:text-emerald-400'
  },
  {
    value: 'rebate',
    label: t('leaderboard.rebateTab'),
    icon: 'gift',
    rule: t('leaderboard.rebateRule'),
    activeClass: 'bg-white text-amber-700 shadow-sm dark:bg-dark-700 dark:text-amber-300',
    iconClass: 'text-amber-600 dark:text-amber-400'
  }
])

const activeTabMeta = computed(() =>
  tabs.value.find((tab) => tab.value === activeTab.value) ?? tabs.value[0]!
)

const summaryItems = computed<SummaryItem[]>(() => {
  if (!data.value) return []
  if (activeTab.value === 'rebate') {
    return [
      { label: t('leaderboard.totalInvitedUsers'), value: formatNumber(data.value.rebate.summary.invited_users), icon: 'users', borderClass: 'border-t-blue-500', iconClass: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300' },
      { label: t('leaderboard.totalRebateCount'), value: formatNumber(data.value.rebate.summary.rebate_count), icon: 'document', borderClass: 'border-t-violet-500', iconClass: 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300' },
      { label: t('leaderboard.totalRebateAmount'), value: formatCurrency(data.value.rebate.summary.rebate_amount), icon: 'gift', borderClass: 'border-t-amber-500', iconClass: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300' }
    ]
  }
  const summary = activeTab.value === 'usage' ? data.value.usage.summary : data.value.consumption.summary
  return [
    { label: t('leaderboard.totalRequests'), value: formatNumber(summary.request_count), icon: 'document', borderClass: 'border-t-blue-500', iconClass: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300' },
    { label: t('leaderboard.totalTokens'), value: formatNumber(summary.total_tokens), icon: 'cube', borderClass: 'border-t-violet-500', iconClass: 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300' },
    { label: t('leaderboard.totalCost'), value: formatCurrency(summary.actual_cost), icon: 'dollar', borderClass: 'border-t-emerald-500', iconClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300' }
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

function rankBadge(rank: number) {
  const rankClass = rank === 1
    ? 'bg-amber-100 text-amber-800 ring-amber-200 dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-500/30'
    : rank === 2
      ? 'bg-gray-100 text-gray-700 ring-gray-200 dark:bg-gray-500/15 dark:text-gray-300 dark:ring-gray-500/30'
      : rank === 3
        ? 'bg-orange-100 text-orange-800 ring-orange-200 dark:bg-orange-500/15 dark:text-orange-300 dark:ring-orange-500/30'
        : 'text-gray-600 dark:text-dark-300'

  return h('span', {
    class: `inline-flex h-8 min-w-8 items-center justify-center rounded-md px-1.5 font-semibold tabular-nums ${rankClass} ${rank <= 3 ? 'ring-1 ring-inset shadow-sm' : ''}`
  }, String(rank))
}

function currentRankCell(rank: number) {
  return h('div', { class: 'flex items-center gap-2' }, [
    h('span', { class: 'inline-flex h-8 min-w-8 items-center justify-center rounded-md bg-primary-600 px-1.5 font-semibold text-white shadow-sm' }, '0'),
    h('span', { class: 'text-xs font-medium text-primary-700 dark:text-primary-300' }, t('leaderboard.currentRank', { rank }))
  ])
}

function displayNameCell(displayName: string, self: boolean) {
  return h('div', { class: 'flex items-center gap-2' }, [
    h('span', { class: 'font-medium text-gray-900 dark:text-white' }, displayName),
    self ? h('span', { class: 'rounded bg-primary-100 px-1.5 py-0.5 text-[11px] font-semibold text-primary-700 dark:bg-primary-500/15 dark:text-primary-300' }, t('leaderboard.myData')) : null
  ])
}

const UsageRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardUsageEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('tr', {
      class: props.self
        ? 'border-b border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'border-b border-gray-100 transition-colors hover:bg-gray-50/80 dark:border-dark-800 dark:hover:bg-dark-700/40'
    }, [
      h('td', { class: `px-4 py-3 ${props.self ? 'border-l-4 border-primary-500' : ''}` }, props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank)),
      h('td', { class: 'px-4 py-3' }, displayNameCell(props.entry.display_name, props.self)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.request_count)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.total_tokens)),
      h('td', { class: 'px-4 py-3 text-right font-medium tabular-nums text-gray-800 dark:text-dark-200' }, formatCurrency(props.entry.actual_cost))
    ])
  }
})

const RebateRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardRebateEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('tr', {
      class: props.self
        ? 'border-b border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'border-b border-gray-100 transition-colors hover:bg-gray-50/80 dark:border-dark-800 dark:hover:bg-dark-700/40'
    }, [
      h('td', { class: `px-4 py-3 ${props.self ? 'border-l-4 border-primary-500' : ''}` }, props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank)),
      h('td', { class: 'px-4 py-3' }, displayNameCell(props.entry.display_name, props.self)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.invited_users)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.rebate_count)),
      h('td', { class: 'px-4 py-3 text-right font-medium tabular-nums text-gray-800 dark:text-dark-200' }, formatCurrency(props.entry.rebate_amount))
    ])
  }
})

watch(period, () => { void load() }, { immediate: true })
</script>

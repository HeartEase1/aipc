<template>
  <AppLayout>
    <div class="leaderboard-page space-y-6">
      <header class="leaderboard-heading-card flex flex-col gap-4 bg-white p-4 dark:bg-dark-800 sm:flex-row sm:items-start sm:justify-between sm:p-5">
        <div class="flex items-start gap-4">
          <div class="leaderboard-trophy flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
            <Icon name="trophy" size="lg" />
          </div>
          <div>
            <h1 class="text-xl font-bold text-gray-900 dark:text-white">
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

      <div class="leaderboard-toolbar flex flex-col gap-5 bg-white p-4 dark:bg-dark-800 sm:flex-row sm:items-center sm:justify-between sm:p-5">
        <div class="min-w-0">
          <p class="mb-2 text-xs font-medium text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.periodLabel') }}
          </p>
          <div class="leaderboard-period-switch inline-flex max-w-full overflow-x-auto bg-gray-100 p-1 dark:bg-dark-900" role="tablist">
            <button
              v-for="option in periodOptions"
              :key="option.value"
              type="button"
              class="leaderboard-period-button min-w-16 px-3 py-2 text-sm font-medium"
              :class="period === option.value ? 'leaderboard-period-button-active bg-white text-gray-900 dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:bg-white/60 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-700/60 dark:hover:text-white'"
              :aria-selected="period === option.value"
              @click="period = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <label class="leaderboard-participation flex cursor-pointer items-center justify-between gap-5 border-t border-gray-100 pt-4 dark:border-dark-700 sm:min-w-72 sm:border-l sm:border-t-0 sm:pl-5 sm:pt-0">
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
            <span class="h-7 w-12 rounded-full border border-gray-300 bg-gray-200 shadow-inner transition-colors peer-checked:border-primary-700 peer-checked:bg-primary-600 peer-disabled:cursor-not-allowed peer-disabled:opacity-50 dark:border-dark-600 dark:bg-dark-700"></span>
            <span class="pointer-events-none absolute left-1 top-1 h-5 w-5 rounded-full bg-white shadow-md transition-transform peer-checked:translate-x-5"></span>
          </span>
        </label>
      </div>

      <div v-if="loading" class="leaderboard-loading flex justify-center bg-white py-20 dark:bg-dark-800">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="data">
        <section aria-labelledby="leaderboard-summary-title">
          <div class="leaderboard-summary-heading mb-4 flex items-start gap-3 bg-white p-4 dark:bg-dark-800">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-gray-50 shadow-sm ring-1 ring-gray-200 dark:bg-dark-900/70 dark:ring-dark-600">
              <Icon :name="activeTabMeta.icon" size="sm" :class="activeTabMeta.iconClass" />
            </span>
            <div>
              <h2 id="leaderboard-summary-title" class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ activeTabMeta.label }} / {{ selectedPeriodLabel }}
              </h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.summaryScope') }}
              </p>
            </div>
          </div>

          <div class="grid gap-4 sm:grid-cols-3">
            <div
              v-for="item in summaryItems"
              :key="item.label"
              class="leaderboard-stat-card group relative min-h-28 overflow-hidden border border-t-4 border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800 sm:p-5"
              :class="item.borderClass"
            >
              <div class="flex items-start justify-between gap-3">
                <div>
                  <p class="text-sm font-medium text-gray-500 dark:text-dark-400">{{ item.label }}</p>
                  <p class="mt-2 text-2xl font-bold tabular-nums text-gray-900 dark:text-white">{{ item.value }}</p>
                </div>
                <div class="leaderboard-stat-icon flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" :class="item.iconClass">
                  <Icon :name="item.icon" size="sm" />
                </div>
              </div>
            </div>
          </div>
        </section>

        <div class="leaderboard-tabs grid grid-cols-3 gap-1.5 bg-gray-100 p-1.5 dark:bg-dark-900" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="leaderboard-tab flex min-h-11 items-center justify-center gap-2 px-2 py-2 text-sm font-semibold"
            :class="activeTab === tab.value ? tab.activeClass : 'text-gray-500 hover:bg-white/70 hover:text-gray-900 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-white'"
            :aria-selected="activeTab === tab.value"
            @click="activeTab = tab.value"
          >
            <Icon :name="tab.icon" size="sm" />
            <span>{{ tab.label }}</span>
          </button>
        </div>

        <div v-if="!data.participating" class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 shadow-sm dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('leaderboard.notParticipating') }}
        </div>

        <div class="leaderboard-table-card overflow-hidden border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-3 border-b border-gray-200 bg-gray-50/80 px-4 py-4 dark:border-dark-700 dark:bg-dark-900/50 sm:flex-row sm:items-center sm:justify-between sm:px-5">
            <div class="flex items-start gap-2">
              <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white shadow-sm ring-1 ring-gray-200 dark:bg-dark-800 dark:ring-dark-600">
                <Icon :name="activeTabMeta.icon" size="sm" :class="activeTabMeta.iconClass" />
              </span>
              <p class="text-sm text-gray-600 dark:text-dark-300">{{ activeTabMeta.rule }}</p>
            </div>
            <span class="w-fit shrink-0 rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-semibold text-gray-600 shadow-sm dark:border-dark-600 dark:bg-dark-800 dark:text-dark-300">
              {{ t('leaderboard.top20') }}
            </span>
          </div>

          <section v-if="activeTab === 'usage'" class="overflow-x-auto">
            <div class="space-y-3 p-3 sm:hidden">
              <MobileUsageRow v-if="data.usage.current" :entry="data.usage.current" :self="true" />
              <div v-else-if="data.participating" class="rounded-xl border border-primary-200 bg-primary-50 p-4 text-center text-sm text-primary-700 dark:border-primary-900/40 dark:bg-primary-950/20 dark:text-primary-300">
                {{ t('leaderboard.notRanked') }}
              </div>
              <MobileUsageRow v-for="entry in data.usage.entries" :key="entry.rank" :entry="entry" />
              <div v-if="data.usage.entries.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.empty') }}
              </div>
            </div>
            <table class="leaderboard-desktop-table hidden w-full min-w-[680px] text-left text-sm sm:table">
              <thead class="border-b border-gray-200 bg-gray-50/70 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
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
            <div class="space-y-3 p-3 sm:hidden">
              <MobileUsageRow v-if="data.consumption.current" :entry="data.consumption.current" :self="true" />
              <div v-else-if="data.participating" class="rounded-xl border border-primary-200 bg-primary-50 p-4 text-center text-sm text-primary-700 dark:border-primary-900/40 dark:bg-primary-950/20 dark:text-primary-300">
                {{ t('leaderboard.notRanked') }}
              </div>
              <MobileUsageRow v-for="entry in data.consumption.entries" :key="entry.rank" :entry="entry" />
              <div v-if="data.consumption.entries.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.empty') }}
              </div>
            </div>
            <table class="leaderboard-desktop-table hidden w-full min-w-[680px] text-left text-sm sm:table">
              <thead class="border-b border-gray-200 bg-gray-50/70 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
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
            <div class="space-y-3 p-3 sm:hidden">
              <MobileRebateRow v-if="data.rebate.current" :entry="data.rebate.current" :self="true" />
              <div v-else-if="data.participating" class="rounded-xl border border-primary-200 bg-primary-50 p-4 text-center text-sm text-primary-700 dark:border-primary-900/40 dark:bg-primary-950/20 dark:text-primary-300">
                {{ t('leaderboard.notRanked') }}
              </div>
              <MobileRebateRow v-for="entry in data.rebate.entries" :key="entry.rank" :entry="entry" />
              <div v-if="data.rebate.entries.length === 0" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.empty') }}
              </div>
            </div>
            <table class="leaderboard-desktop-table hidden w-full min-w-[680px] text-left text-sm sm:table">
              <thead class="border-b border-gray-200 bg-gray-50/70 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-400">
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
    activeClass: 'leaderboard-tab-active border-blue-200 bg-white text-blue-700 dark:border-blue-500/30 dark:bg-dark-700 dark:text-blue-300',
    iconClass: 'text-blue-600 dark:text-blue-400'
  },
  {
    value: 'consumption',
    label: t('leaderboard.consumptionTab'),
    icon: 'dollar',
    rule: t('leaderboard.consumptionRule'),
    activeClass: 'leaderboard-tab-active border-emerald-200 bg-white text-emerald-700 dark:border-emerald-500/30 dark:bg-dark-700 dark:text-emerald-300',
    iconClass: 'text-emerald-600 dark:text-emerald-400'
  },
  {
    value: 'rebate',
    label: t('leaderboard.rebateTab'),
    icon: 'gift',
    rule: t('leaderboard.rebateRule'),
    activeClass: 'leaderboard-tab-active border-amber-200 bg-white text-amber-700 dark:border-amber-500/30 dark:bg-dark-700 dark:text-amber-300',
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
      { label: t('leaderboard.totalRebateAmount'), value: formatLeaderboardCurrency(data.value.rebate.summary.rebate_amount), icon: 'gift', borderClass: 'border-t-amber-500', iconClass: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300' }
    ]
  }
  const summary = activeTab.value === 'usage' ? data.value.usage.summary : data.value.consumption.summary
  return [
    { label: t('leaderboard.totalRequests'), value: formatNumber(summary.request_count), icon: 'document', borderClass: 'border-t-blue-500', iconClass: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300' },
    { label: t('leaderboard.totalTokens'), value: formatNumber(summary.total_tokens), icon: 'cube', borderClass: 'border-t-violet-500', iconClass: 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300' },
    { label: t('leaderboard.totalCost'), value: formatLeaderboardCurrency(summary.actual_cost), icon: 'dollar', borderClass: 'border-t-emerald-500', iconClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300' }
  ]
})

function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value || 0)
}

function formatLeaderboardCurrency(value: number): string {
  const amount = Number.isFinite(value) ? value : 0
  const fractionDigits = amount > 0 && amount < 0.01 ? 6 : 2
  const formattedAmount = new Intl.NumberFormat(undefined, {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits
  }).format(amount)
  return `$${formattedAmount}`
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
    ? 'bg-amber-100 text-amber-800 ring-amber-200 shadow-[0_3px_0_#d97706] dark:bg-amber-500/15 dark:text-amber-300 dark:ring-amber-500/30 dark:shadow-[0_3px_0_#92400e]'
    : rank === 2
      ? 'bg-gray-100 text-gray-700 ring-gray-200 shadow-[0_3px_0_#9ca3af] dark:bg-gray-500/15 dark:text-gray-300 dark:ring-gray-500/30 dark:shadow-[0_3px_0_#4b5563]'
      : rank === 3
        ? 'bg-orange-100 text-orange-800 ring-orange-200 shadow-[0_3px_0_#c2410c] dark:bg-orange-500/15 dark:text-orange-300 dark:ring-orange-500/30 dark:shadow-[0_3px_0_#7c2d12]'
        : 'text-gray-600 dark:text-dark-300'

  return h('span', {
    class: `inline-flex h-8 min-w-8 items-center justify-center rounded-lg px-1.5 font-semibold tabular-nums ${rankClass} ${rank <= 3 ? 'ring-1 ring-inset' : ''}`
  }, String(rank))
}

function currentRankCell(rank: number) {
  return h('div', { class: 'flex items-center gap-2' }, [
    h('span', { class: 'inline-flex h-8 min-w-8 items-center justify-center rounded-lg bg-primary-600 px-1.5 font-semibold text-white shadow-[0_3px_0_#0f766e] dark:shadow-[0_3px_0_#134e4a]' }, '0'),
    h('span', { class: 'text-xs font-medium text-primary-700 dark:text-primary-300' }, t('leaderboard.currentRank', { rank }))
  ])
}

function displayNameCell(displayName: string, self: boolean) {
  return h('div', { class: 'flex items-center gap-2' }, [
    h('span', { class: 'font-medium text-gray-900 dark:text-white' }, displayName),
    self ? h('span', { class: 'rounded-full bg-primary-100 px-2 py-0.5 text-[11px] font-semibold text-primary-700 ring-1 ring-inset ring-primary-200 dark:bg-primary-500/15 dark:text-primary-300 dark:ring-primary-500/20' }, t('leaderboard.myData')) : null
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
        ? 'leaderboard-self-row border-b border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'border-b border-gray-100 transition-colors hover:bg-gray-50/80 dark:border-dark-800 dark:hover:bg-dark-700/40'
    }, [
      h('td', { class: `px-4 py-3 ${props.self ? 'border-l-4 border-primary-500' : ''}` }, props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank)),
      h('td', { class: 'px-4 py-3' }, displayNameCell(props.entry.display_name, props.self)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.request_count)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.total_tokens)),
      h('td', { class: 'px-4 py-3 text-right font-medium tabular-nums text-gray-800 dark:text-dark-200' }, formatLeaderboardCurrency(props.entry.actual_cost))
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
        ? 'leaderboard-self-row border-b border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'border-b border-gray-100 transition-colors hover:bg-gray-50/80 dark:border-dark-800 dark:hover:bg-dark-700/40'
    }, [
      h('td', { class: `px-4 py-3 ${props.self ? 'border-l-4 border-primary-500' : ''}` }, props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank)),
      h('td', { class: 'px-4 py-3' }, displayNameCell(props.entry.display_name, props.self)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.invited_users)),
      h('td', { class: 'px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-300' }, formatNumber(props.entry.rebate_count)),
      h('td', { class: 'px-4 py-3 text-right font-medium tabular-nums text-gray-800 dark:text-dark-200' }, formatLeaderboardCurrency(props.entry.rebate_amount))
    ])
  }
})

function mobileMetric(label: string, value: string, emphasized = false) {
  return h('div', { class: 'min-w-0 px-2 text-center' }, [
    h('span', { class: 'block text-[11px] leading-4 text-gray-500 dark:text-dark-400' }, label),
    h('span', {
      class: `mt-1 block truncate text-xs font-semibold tabular-nums ${emphasized ? 'text-gray-900 dark:text-white' : 'text-gray-700 dark:text-dark-200'}`
    }, value)
  ])
}

const MobileUsageRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardUsageEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('article', {
      class: props.self
        ? 'leaderboard-mobile-row leaderboard-mobile-row-self border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'leaderboard-mobile-row border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800'
    }, [
      h('div', { class: 'flex items-center justify-between gap-3' }, [
        props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank),
        displayNameCell(props.entry.display_name, props.self)
      ]),
      h('div', { class: 'mt-3 grid grid-cols-3 divide-x divide-gray-200 rounded-lg bg-gray-50/90 py-2.5 dark:divide-dark-700 dark:bg-dark-900/55' }, [
        mobileMetric(t('leaderboard.requests'), formatNumber(props.entry.request_count)),
        mobileMetric(t('leaderboard.tokens'), formatNumber(props.entry.total_tokens)),
        mobileMetric(t('leaderboard.cost'), formatLeaderboardCurrency(props.entry.actual_cost), true)
      ])
    ])
  }
})

const MobileRebateRow = defineComponent({
  props: {
    entry: { type: Object as PropType<LeaderboardRebateEntry>, required: true },
    self: { type: Boolean, default: false }
  },
  setup(props) {
    return () => h('article', {
      class: props.self
        ? 'leaderboard-mobile-row leaderboard-mobile-row-self border-primary-200 bg-primary-50/80 dark:border-primary-900/40 dark:bg-primary-950/20'
        : 'leaderboard-mobile-row border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800'
    }, [
      h('div', { class: 'flex items-center justify-between gap-3' }, [
        props.self ? currentRankCell(props.entry.rank) : rankBadge(props.entry.rank),
        displayNameCell(props.entry.display_name, props.self)
      ]),
      h('div', { class: 'mt-3 grid grid-cols-3 divide-x divide-gray-200 rounded-lg bg-gray-50/90 py-2.5 dark:divide-dark-700 dark:bg-dark-900/55' }, [
        mobileMetric(t('leaderboard.invitedUsers'), formatNumber(props.entry.invited_users)),
        mobileMetric(t('leaderboard.rebateCount'), formatNumber(props.entry.rebate_count)),
        mobileMetric(t('leaderboard.rebateAmount'), formatLeaderboardCurrency(props.entry.rebate_amount), true)
      ])
    ])
  }
})

watch(period, () => { void load() }, { immediate: true })
</script>

<style scoped>
.leaderboard-trophy {
  border: 1px solid rgb(245 158 11 / 0.28);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.82),
    0 4px 0 rgb(217 119 6 / 0.2),
    0 12px 22px -14px rgb(146 64 14 / 0.7);
}

.leaderboard-heading-card,
.leaderboard-summary-heading {
  border: 1px solid rgb(229 231 235);
  border-radius: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.92),
    0 4px 0 rgb(209 213 219 / 0.78),
    0 16px 28px -22px rgb(15 23 42 / 0.55);
}

.leaderboard-summary-heading {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.92),
    0 3px 0 rgb(209 213 219 / 0.72),
    0 12px 22px -20px rgb(15 23 42 / 0.5);
}

.leaderboard-toolbar,
.leaderboard-loading {
  border: 1px solid rgb(229 231 235);
  border-radius: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.9),
    0 4px 0 rgb(209 213 219 / 0.8),
    0 16px 28px -22px rgb(15 23 42 / 0.55);
}

.leaderboard-period-switch,
.leaderboard-tabs {
  border: 1px solid rgb(229 231 235);
  border-radius: 0.75rem;
  box-shadow: inset 0 1px 3px rgb(15 23 42 / 0.08);
}

.leaderboard-period-button,
.leaderboard-tab {
  border: 1px solid transparent;
  border-radius: 0.625rem;
  transition:
    color 160ms ease,
    background-color 160ms ease,
    border-color 160ms ease,
    box-shadow 160ms ease,
    transform 160ms ease;
}

.leaderboard-period-button-active {
  border-color: rgb(209 213 219);
  box-shadow:
    0 2px 0 rgb(209 213 219),
    0 5px 10px -7px rgb(15 23 42 / 0.55);
  transform: translateY(-1px);
}

.leaderboard-stat-card {
  border-radius: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.92),
    0 4px 0 rgb(209 213 219 / 0.85),
    0 18px 30px -24px rgb(15 23 42 / 0.65);
  transition:
    transform 180ms ease,
    box-shadow 180ms ease;
}

.leaderboard-stat-card:hover {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.92),
    0 6px 0 rgb(209 213 219 / 0.85),
    0 22px 34px -24px rgb(15 23 42 / 0.72);
  transform: translateY(-2px);
}

.leaderboard-stat-icon {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.72),
    0 3px 0 rgb(15 23 42 / 0.08);
}

.leaderboard-tab-active {
  box-shadow:
    0 3px 0 rgb(148 163 184 / 0.45),
    0 8px 14px -10px rgb(15 23 42 / 0.65);
  transform: translateY(-1px);
}

.leaderboard-table-card {
  border-radius: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.9),
    0 5px 0 rgb(209 213 219 / 0.78),
    0 22px 38px -28px rgb(15 23 42 / 0.7);
}

.leaderboard-desktop-table {
  table-layout: fixed;
}

.leaderboard-desktop-table th:nth-child(1),
.leaderboard-desktop-table td:nth-child(1) {
  width: 26%;
}

.leaderboard-desktop-table th:nth-child(2),
.leaderboard-desktop-table td:nth-child(2) {
  width: 29%;
}

.leaderboard-desktop-table th:nth-child(n + 3),
.leaderboard-desktop-table td:nth-child(n + 3) {
  width: 15%;
  text-align: right;
  white-space: nowrap;
}

.leaderboard-mobile-row {
  border-width: 1px;
  border-radius: 0.75rem;
  padding: 0.875rem;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.88),
    0 3px 0 rgb(209 213 219 / 0.8),
    0 12px 20px -18px rgb(15 23 42 / 0.65);
}

.leaderboard-mobile-row-self {
  box-shadow:
    inset 4px 0 0 rgb(13 148 136),
    inset 0 1px 0 rgb(255 255 255 / 0.75),
    0 3px 0 rgb(13 148 136 / 0.32),
    0 12px 20px -18px rgb(15 23 42 / 0.65);
}

.leaderboard-self-row {
  box-shadow: inset 4px 0 0 rgb(13 148 136);
}

:global(.dark) .leaderboard-trophy {
  border-color: rgb(245 158 11 / 0.22);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.08),
    0 4px 0 rgb(120 53 15 / 0.62),
    0 14px 24px -15px rgb(0 0 0 / 0.9);
}

:global(.dark) .leaderboard-heading-card,
:global(.dark) .leaderboard-summary-heading {
  border-color: rgb(51 65 85);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 4px 0 rgb(2 6 23 / 0.9),
    0 18px 32px -22px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-summary-heading {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 3px 0 rgb(2 6 23 / 0.88),
    0 14px 24px -20px rgb(0 0 0 / 0.92);
}

:global(.dark) .leaderboard-toolbar,
:global(.dark) .leaderboard-loading,
:global(.dark) .leaderboard-table-card {
  border-color: rgb(51 65 85);
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 4px 0 rgb(2 6 23 / 0.9),
    0 18px 32px -22px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-period-switch,
:global(.dark) .leaderboard-tabs {
  border-color: rgb(51 65 85);
  box-shadow: inset 0 1px 4px rgb(0 0 0 / 0.42);
}

:global(.dark) .leaderboard-period-button-active {
  border-color: rgb(71 85 105);
  box-shadow:
    0 2px 0 rgb(15 23 42),
    0 7px 12px -9px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-stat-card {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 4px 0 rgb(2 6 23 / 0.9),
    0 18px 30px -22px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-stat-card:hover {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.07),
    0 6px 0 rgb(2 6 23 / 0.9),
    0 22px 34px -22px rgb(0 0 0 / 0.98);
}

:global(.dark) .leaderboard-stat-icon {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.08),
    0 3px 0 rgb(0 0 0 / 0.24);
}

:global(.dark) .leaderboard-mobile-row {
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 3px 0 rgb(2 6 23 / 0.9),
    0 14px 22px -18px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-mobile-row-self {
  box-shadow:
    inset 4px 0 0 rgb(13 148 136),
    inset 0 1px 0 rgb(255 255 255 / 0.055),
    0 3px 0 rgb(19 78 74 / 0.75),
    0 14px 22px -18px rgb(0 0 0 / 0.95);
}

:global(.dark) .leaderboard-tab-active {
  box-shadow:
    0 3px 0 rgb(15 23 42),
    0 9px 15px -11px rgb(0 0 0 / 0.98);
}

@media (prefers-reduced-motion: reduce) {
  .leaderboard-period-button,
  .leaderboard-tab,
  .leaderboard-stat-card {
    transition-duration: 1ms;
  }

  .leaderboard-period-button-active,
  .leaderboard-tab-active,
  .leaderboard-stat-card:hover {
    transform: none;
  }
}
</style>

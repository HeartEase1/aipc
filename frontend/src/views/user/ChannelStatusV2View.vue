<template>
  <AppLayout>
    <div class="space-y-6 pb-12">
      <section class="monitor-hero">
        <header class="page-header monitor-hero__header mb-0">
          <div class="flex min-w-0 items-start gap-3.5">
            <span class="monitor-hero__icon">
              <Icon name="chart" size="sm" />
            </span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-x-2.5 gap-y-1.5">
                <h1 class="page-title text-lg font-black text-gray-900 dark:text-white">
                  {{ t('channelMonitorV2.title') }}
                </h1>
                <span class="monitor-hero__live">
                  <i
                    class="h-1.5 w-1.5 shrink-0 rounded-full"
                    :class="loading || refreshing ? 'bg-gray-400' : 'bg-emerald-500'"
                    aria-hidden="true"
                  />
                  <span v-if="refreshing" class="inline-flex items-center gap-1 text-primary-600 dark:text-primary-300">
                    <LoadingSpinner size="sm" />
                    {{ t('channelMonitorV2.updating') }}
                  </span>
                  <span v-else-if="snapshot?.coverage.data_through">
                    {{ t('channelMonitorV2.updatedTo', { time: formatTime(snapshot.coverage.data_through) }) }}
                  </span>
                  <span v-else>{{ t('common.loading') }}</span>
                </span>
                <span v-if="snapshot && !snapshot.coverage.coverage_complete && !bootstrapActive" class="badge badge-warning">
                  {{ t('channelMonitorV2.partialCoverage') }}
                </span>
              </div>
              <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
                {{ t('channelMonitorV2.overview.description') }}
              </p>
            </div>
          </div>

          <div class="monitor-hero__controls">
            <div class="monitor-hero__range" role="group" :aria-label="t('channelMonitorV2.timeRange')">
              <button
                v-for="option in ranges"
                :key="option.value"
                type="button"
                class="tab monitor-hero__range-button"
                :class="filter.range === option.value ? 'tab-active is-active' : ''"
                @click="setRange(option.value)"
              >
                {{ option.label }}
              </button>
            </div>
            <button
              class="btn btn-secondary monitor-hero__refresh"
              type="button"
              :title="t('common.refresh')"
              :disabled="loading"
              @click="reload(false)"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </header>

        <!-- First-upgrade silent backfill: show until 30d product window is covered -->
        <div
          v-if="bootstrapActive"
          class="border-t border-blue-100 bg-blue-50/90 px-5 py-3 dark:border-blue-900/40 dark:bg-blue-950/40 sm:px-6"
          role="status"
          aria-live="polite"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold text-blue-900 dark:text-blue-100">
                {{ t('channelMonitorV2.bootstrap.title') }}
              </p>
              <p class="mt-0.5 text-xs text-blue-800/80 dark:text-blue-200/80">
                {{ t('channelMonitorV2.bootstrap.description') }}
              </p>
            </div>
            <span class="shrink-0 text-xs font-medium tabular-nums text-blue-700 dark:text-blue-300">
              {{ t('channelMonitorV2.bootstrap.progress', { percent: bootstrapPercent }) }}
            </span>
          </div>
          <div
            class="mt-2.5 h-1.5 overflow-hidden rounded-full bg-blue-200/80 dark:bg-blue-900/60"
            role="progressbar"
            :aria-valuenow="bootstrapPercent"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-label="t('channelMonitorV2.bootstrap.working')"
          >
            <div
              class="h-full rounded-full bg-blue-500 transition-[width] duration-500 ease-out dark:bg-blue-400"
              :style="{ width: `${bootstrapPercent}%` }"
            />
          </div>
        </div>

        <dl v-if="snapshot" class="monitor-summary">
          <div class="monitor-summary__item monitor-summary__item--total">
            <dt><i />{{ t('channelMonitorV2.overview.totalGroups') }}</dt>
            <dd>{{ overviewCounts.total }}</dd>
          </div>
          <div class="monitor-summary__item monitor-summary__item--healthy">
            <dt><i />{{ t('channelMonitorV2.overview.healthyGroups') }}</dt>
            <dd>{{ overviewCounts.healthy }}</dd>
          </div>
          <div class="monitor-summary__item monitor-summary__item--attention">
            <dt><i />{{ t('channelMonitorV2.overview.attentionGroups') }}</dt>
            <dd>{{ overviewCounts.attention }}</dd>
          </div>
          <div class="monitor-summary__item monitor-summary__item--unknown">
            <dt><i />{{ t('channelMonitorV2.overview.noDataGroups') }}</dt>
            <dd>{{ overviewCounts.unknown }}</dd>
          </div>
        </dl>
      </section>

      <section :aria-label="t('channelMonitorV2.overview.title')">
        <div class="mb-3 flex flex-wrap items-end justify-between gap-3 px-1">
          <div>
            <h2 class="text-base font-black text-gray-900 dark:text-white">{{ t('channelMonitorV2.overview.title') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t(showDetailedAnalysis ? 'channelMonitorV2.overview.hint' : 'channelMonitorV2.overview.summaryHint') }}
            </p>
          </div>
          <div class="flex items-center gap-3 text-[11px] text-gray-400 dark:text-gray-500" :aria-label="t('channelMonitorV2.overview.legend')">
            <span class="inline-flex items-center gap-1.5"><i class="h-1.5 w-1.5 rounded-full bg-emerald-500" />{{ t('channelMonitorV2.overview.status.healthy') }}</span>
            <span class="inline-flex items-center gap-1.5"><i class="h-1.5 w-1.5 rounded-full bg-amber-500" />{{ t('channelMonitorV2.overview.status.warning') }}</span>
            <span class="inline-flex items-center gap-1.5"><i class="h-1.5 w-1.5 rounded-full bg-red-500" />{{ t('channelMonitorV2.overview.status.critical') }}</span>
          </div>
        </div>

        <div v-if="overviewRows.length" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          <ChannelHealthCard
            v-for="row in overviewRows"
            :key="`${row.platform}:${row.group_id || row.group_name}`"
            :row="row"
            :rate="rateForRow(row)"
            :interactive="showDetailedAnalysis"
            @select="openGroupDetails(row)"
          />
        </div>
        <div v-else-if="loading" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4" aria-hidden="true">
          <div v-for="i in 6" :key="i" class="h-60 animate-pulse rounded-2xl bg-white/70 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800/70 dark:ring-dark-700" />
        </div>
        <div v-else class="card empty-state !rounded-2xl !border-0 py-14 shadow-sm ring-1 ring-gray-900/5 dark:ring-dark-700">
          <p class="empty-state-title">{{ t('channelMonitorV2.empty.title') }}</p>
          <p class="empty-state-description">{{ t('channelMonitorV2.empty.description') }}</p>
        </div>
      </section>

      <section v-if="showDetailedAnalysis" ref="advancedSection" class="card overflow-hidden !rounded-2xl !border-0 p-0 shadow-sm ring-1 ring-gray-900/5 dark:!bg-dark-800 dark:ring-dark-700">
        <button
          type="button"
          class="flex w-full items-center justify-between gap-4 px-5 py-4 text-left transition-colors hover:bg-gray-50/80 dark:hover:bg-dark-700/40 sm:px-6"
          :aria-expanded="advancedOpen"
          @click="advancedOpen = !advancedOpen"
        >
          <span class="min-w-0">
            <span class="flex items-center gap-2 text-sm font-black text-gray-900 dark:text-white">
              <Icon name="chartBar" size="sm" class="text-primary-500" />
              {{ t('channelMonitorV2.overview.advancedTitle') }}
              <span v-if="hasDimensionFilter" class="badge badge-primary">{{ t('channelMonitorV2.overview.activeFilters', { count: activeFilterCount }) }}</span>
            </span>
            <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ t('channelMonitorV2.overview.advancedHint') }}</span>
          </span>
          <Icon name="chevronDown" size="sm" :class="['shrink-0 text-gray-400 transition-transform', advancedOpen ? 'rotate-180' : '']" />
        </button>

        <div v-if="advancedOpen" class="space-y-5 border-t border-gray-100 p-4 dark:border-dark-700 sm:p-5">
          <div class="monitor-toolbar flex flex-nowrap items-center gap-1.5 overflow-x-auto rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-dark-900/35 sm:gap-2">
          <FilterMultiSelect
            v-model="filter.platforms"
            compact
            :label="t('channelMonitorV2.filters.platform')"
            :all-label="t('channelMonitorV2.filters.allPlatforms')"
            :options="platformOptions"
          />
          <FilterMultiSelect
            v-model="selectedGroupIds"
            compact
            :label="t('channelMonitorV2.filters.group')"
            :all-label="t('channelMonitorV2.filters.allGroups')"
            :options="groupOptions"
          />
          <FilterMultiSelect
            v-model="filter.models"
            compact
            :label="t('channelMonitorV2.filters.model')"
            :all-label="t('channelMonitorV2.filters.allModels')"
            :options="modelOptions"
          />
          <button
            type="button"
            class="btn btn-ghost btn-sm shrink-0 !px-2 !py-1 text-xs"
            :disabled="!hasDimensionFilter"
            :class="!hasDimensionFilter ? 'opacity-40' : ''"
            @click="clearDimensions"
          >
            {{ t('channelMonitorV2.clearFilters') }}
          </button>

          <span class="mx-0.5 hidden h-5 w-px shrink-0 bg-gray-200 dark:bg-dark-700 md:block" aria-hidden="true"></span>

          <Select
            v-model="matrixGroupBy"
            :options="matrixGroupOptions"
            :placeholder="t('channelMonitorV2.groupBy.label')"
            class="monitor-toolbar-select w-[7.5rem] shrink-0 sm:w-[8.5rem]"
          />

          <div
            class="tabs inline-flex shrink-0"
            role="group"
            :aria-label="t('channelMonitorV2.trendView.label')"
          >
            <button
              type="button"
              class="tab !px-2 !py-1 text-xs"
              :class="trendView === 'pulse' ? 'tab-active' : ''"
              @click="trendView = 'pulse'"
            >
              {{ t('channelMonitorV2.trendView.pulse') }}
            </button>
            <button
              type="button"
              class="tab !px-2 !py-1 text-xs"
              :class="trendView === 'line' ? 'tab-active' : ''"
              @click="trendView = 'line'"
            >
              {{ t('channelMonitorV2.trendView.line') }}
            </button>
          </div>

          <div
            v-if="trendView === 'pulse'"
            class="tabs inline-flex shrink-0"
            role="group"
            :aria-label="t('channelMonitorV2.healthMode.label')"
          >
            <button
              v-for="option in healthModeOptions"
              :key="option.value"
              type="button"
              class="tab !px-2 !py-1 text-xs"
              :class="healthMode === option.value ? 'tab-active' : ''"
              @click="healthMode = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

      <!-- Overview KPI: success · TTFT · tokens/s(optional) · cache · (+ RPM when throughput visible) -->
      <section
        v-if="snapshot"
        class="grid grid-cols-2 gap-3 sm:grid-cols-3"
        :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
        :aria-label="t('channelMonitorV2.summaryAria')"
      >
        <MetricCell
          :label="t('channelMonitorV2.metrics.successRate')"
          :value="formatPercent(1 - snapshot.metrics.error_rate)"
          :detail="t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(snapshot.metrics.error_rate) })"
          :state="snapshot.health.error_rate"
        />
        <MetricCell
          :label="t('channelMonitorV2.metrics.ttftP50')"
          :value="formatMs(snapshot.metrics.ttft.p50_ms)"
          :detail="latencyKpiSecondary(snapshot.metrics.ttft)"
          :title="latencyDetail(snapshot.metrics.ttft)"
          :state="snapshot.health.ttft"
        />
        <MetricCell
          v-if="showThroughput"
          :label="t('channelMonitorV2.metrics.tps')"
          :value="formatTps(snapshot.metrics.tpm)"
          :detail="t('channelMonitorV2.metrics.tpsDetail')"
          :title="exactTps(snapshot.metrics.tpm)"
        />
        <MetricCell
          :label="t('channelMonitorV2.metrics.cacheRate')"
          :value="formatPercent(snapshot.metrics.cache_rate)"
          :detail="t('channelMonitorV2.metrics.cacheDetail')"
          :state="snapshot.health.cache || snapshot.health.overall"
        />
        <MetricCell
          v-if="showThroughput"
          :label="t('channelMonitorV2.metrics.rpm')"
          :value="formatRate(snapshot.metrics.rpm)"
          :detail="t('channelMonitorV2.metrics.rpmDetail')"
          :title="exactRate(snapshot.metrics.rpm)"
        />
      </section>
      <section
        v-else-if="loading"
        class="grid grid-cols-2 gap-3 sm:grid-cols-3"
        :class="showThroughput ? 'xl:grid-cols-5' : 'xl:grid-cols-4'"
        aria-hidden="true"
      >
        <div
          v-for="i in (showThroughput ? 5 : 4)"
          :key="i"
          class="h-24 animate-pulse rounded-2xl bg-gray-50 dark:bg-dark-900/30"
        />
      </section>

      <div class="relative min-h-[320px]">
        <MonitorTrendChart
          v-if="trendView === 'line'"
          :trend="snapshot?.trend || []"
          :coverage="snapshot?.coverage || null"
          :loading="loading && !snapshot"
        />
        <RelayPulseMatrix
          v-else-if="matrix"
          :rows="matrixRows"
          :coverage="matrix.coverage"
          :health-mode="healthMode"
          :show-throughput="showThroughput"
        />
        <div
          v-else-if="loading"
          class="card flex min-h-[320px] items-center justify-center !rounded-3xl !border-0 text-sm text-gray-400 shadow-sm ring-1 ring-gray-900/5 dark:ring-dark-700"
        >
          <span class="animate-pulse">{{ t('common.loading') }}</span>
        </div>
      </div>

      <section class="card flex min-h-0 flex-col overflow-hidden !rounded-3xl !border-0 shadow-sm ring-1 ring-gray-900/5 dark:!bg-dark-800 dark:ring-dark-700">
        <div class="border-b border-gray-100 px-5 pt-4 dark:border-dark-700 sm:px-6">
          <nav class="tabs w-full max-w-md sm:w-auto" role="tablist" :aria-label="t('channelMonitorV2.tabs.aria')">
            <button
              v-for="item in tabs"
              :key="item.value"
              type="button"
              role="tab"
              class="tab flex-1 sm:flex-none"
              :aria-selected="activeTab === item.value"
              :class="activeTab === item.value ? 'tab-active' : ''"
              @click="activeTab = item.value"
            >
              {{ item.label }}
            </button>
          </nav>
        </div>
        <div class="min-h-0 max-h-[min(52vh,520px)] overflow-auto p-4 sm:p-5">
          <div v-if="activeTab === 'models'" class="table-container border-0">
            <table class="table monitor-table min-w-[720px]">
              <thead>
                <tr>
                  <th>{{ t('channelMonitorV2.table.platformModel') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in modelRows"
                  :key="`${row.platform}:${row.model}`"
                  class="cursor-pointer"
                  @click="drillModel(row)"
                >
                  <td>
                    <div class="flex items-center gap-2">
                      <span :class="statusDot(row.health)" aria-hidden="true"></span>
                      <div>
                        <span class="block text-xs text-gray-500 dark:text-dark-400">{{ row.platform }}</span>
                        <strong class="font-semibold text-gray-900 dark:text-white">
                          {{ row.model === '__other__' ? t('channelMonitorV2.otherModels') : row.model }}
                        </strong>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="block">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                    <small class="text-xs text-gray-400">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</small>
                  </td>
                  <td>
                    <span class="block">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                    <small class="text-xs text-gray-400">{{ latencyDetail(row.metrics.ttft) }}</small>
                  </td>
                  <td v-if="showThroughput" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</td>
                  <td>{{ formatPercent(row.metrics.cache_rate) }}</td>
                  <td v-if="showThroughput">{{ formatRate(row.metrics.rpm) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else-if="activeTab === 'errors'" class="space-y-3">
            <div
              v-for="row in errorRows"
              :key="row.category"
              class="rounded-2xl bg-gray-50 p-4 text-sm dark:bg-dark-900/30"
              :class="row.ignored ? 'opacity-60' : ''"
            >
              <button
                type="button"
                class="grid w-full grid-cols-[minmax(100px,200px)_1fr_auto_auto] items-center gap-3 text-left"
                @click="toggleError(row.category)"
              >
                <span class="flex min-w-0 items-center gap-1.5 truncate text-gray-700 dark:text-gray-200">
                  <span class="truncate">{{ errorLabel(row.category) }}</span>
                  <span v-if="row.ignored" class="badge badge-gray shrink-0 !px-1.5 !py-0 text-[10px]">{{ t('channelMonitorV2.ignored') }}</span>
                </span>
                <span class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <i
                    class="block h-full rounded-full"
                    :class="row.ignored ? 'bg-gray-400 dark:bg-gray-500' : 'bg-gradient-to-r from-red-400 to-red-500'"
                    :style="{ width: `${Math.max(2, row.rate * 100)}%` }"
                  ></i>
                </span>
                <small
                  class="w-14 text-right text-xs tabular-nums"
                  :class="row.ignored ? 'text-gray-400' : 'text-gray-500'"
                >{{ formatPercent(row.rate) }}</small>
                <Icon name="chevronDown" size="sm" :class="['text-gray-400 transition-transform', expandedErrors.has(row.category) ? 'rotate-180' : '']" />
              </button>
              <div v-if="expandedErrors.has(row.category)" class="mt-3 space-y-2 border-t border-gray-100 pt-3 dark:border-dark-700">
                <template v-if="isAdmin && (row.details || []).length">
                  <div
                    v-for="(detail, index) in row.details || []"
                    :key="`${row.category}:${index}:${detail.message}`"
                    class="rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-900/50 dark:text-dark-300"
                  >
                    <div class="mb-1 flex flex-wrap items-center gap-2">
                      <span class="badge badge-gray !px-1.5 !py-0 text-[10px]">{{ detail.platform || '-' }}</span>
                      <span class="truncate font-medium">{{ detail.model || '-' }}</span>
                      <span v-if="detail.status_code" class="text-gray-400">{{ t('channelMonitorV2.errorDetail.http', { code: detail.status_code }) }}</span>
                      <span v-if="detail.upstream_status_code" class="text-gray-400">{{ t('channelMonitorV2.errorDetail.upstream', { code: detail.upstream_status_code }) }}</span>
                      <span class="ml-auto text-gray-400">×{{ detail.count }}</span>
                    </div>
                    <p class="break-words leading-relaxed">{{ detail.message || detail.error_type || t('channelMonitorV2.errorDetail.noMessage') }}</p>
                  </div>
                </template>
                <p v-else class="text-xs text-gray-400">{{ t('channelMonitorV2.errorDetail.empty') }}</p>
              </div>
            </div>
          </div>

          <div v-else class="table-container border-0">
            <table class="table monitor-table min-w-[640px]">
              <thead>
                <tr>
                  <th class="w-16">{{ t('channelMonitorV2.table.rank') }}</th>
                  <th>{{ t('channelMonitorV2.table.user') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.successRate') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.ttftP50') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.tps') }}</th>
                  <th>{{ t('channelMonitorV2.metrics.cacheRate') }}</th>
                  <th v-if="showThroughput">{{ t('channelMonitorV2.metrics.rpm') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in userRows"
                  :key="row.user_id || row.display_label"
                  :class="row.is_self
                    ? 'bg-primary-50 ring-1 ring-inset ring-primary-200/80 dark:bg-primary-900/25 dark:ring-primary-700/50'
                    : ''"
                >
                  <td><MonitorRankBadge :rank="row.rank" /></td>
                  <td>
                    <strong
                      class="font-semibold"
                      :class="row.is_self ? 'text-primary-700 dark:text-primary-300' : 'text-gray-900 dark:text-white'"
                    >
                      {{ row.display_label }}
                      <span
                        v-if="row.is_self"
                        class="badge badge-primary ml-2 !px-1.5 !py-0 text-[10px]"
                      >{{ t('channelMonitorV2.currentUser') }}</span>
                    </strong>
                  </td>
                  <td>
                    <span class="block">{{ formatPercent(1 - row.metrics.error_rate) }}</span>
                    <small class="text-xs text-gray-400">{{ t('channelMonitorV2.metrics.errorRateValue', { value: formatPercent(row.metrics.error_rate) }) }}</small>
                  </td>
                  <td>
                    <span class="block">{{ formatMs(row.metrics.ttft.p50_ms) }}</span>
                    <small class="text-xs text-gray-400">{{ latencyDetail(row.metrics.ttft) }}</small>
                  </td>
                  <td v-if="showThroughput" :title="exactTps(row.metrics.tpm)">{{ formatTps(row.metrics.tpm) }}</td>
                  <td>{{ formatPercent(row.metrics.cache_rate) }}</td>
                  <td v-if="showThroughput">{{ formatRate(row.metrics.rpm) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-if="tabLoading" class="empty-state py-10 text-sm text-gray-400">{{ t('common.loading') }}</div>
          <div v-else-if="activeRowsEmpty" class="empty-state py-10">
            <p class="empty-state-title text-base">
              {{
                bootstrapActive
                  ? t('channelMonitorV2.bootstrap.title')
                  : t('channelMonitorV2.empty.title')
              }}
            </p>
            <p class="empty-state-description">
              {{
                bootstrapActive
                  ? t('channelMonitorV2.bootstrap.description')
                  : t('channelMonitorV2.empty.description')
              }}
            </p>
          </div>
        </div>
      </section>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'
import MetricCell from '@/features/channel-monitor-v2/MetricCell.vue'
import MonitorRankBadge from '@/features/channel-monitor-v2/MonitorRankBadge.vue'
import MonitorTrendChart from '@/features/channel-monitor-v2/MonitorTrendChart.vue'
import RelayPulseMatrix from '@/features/channel-monitor-v2/RelayPulseMatrix.vue'
import ChannelHealthCard from '@/features/channel-monitor-v2/ChannelHealthCard.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userGroupsAPI } from '@/api/groups'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  isChannelMonitorThroughputHidden,
  isChannelMonitorV2DetailedAnalysisEnabled,
} from '@/utils/featureFlags'
import * as api from '@/api/channelMonitorV2'
import type {
  HealthState,
  MonitorDimensions,
  MonitorErrorRow,
  MonitorFilter,
  MonitorHealth,
  MonitorMatrixGroupBy,
  MonitorMatrixRow,
  MonitorMatrixResponse,
  MonitorModelRow,
  MonitorRange,
  MonitorSnapshot,
  MonitorUserRow,
} from '@/api/channelMonitorV2'
import {
  formatLatencyKpiSecondary,
  formatLatencyPrivacy,
  formatMonitorMs,
  formatMonitorPercent,
  formatMonitorThroughput,
  formatMonitorTokensPerSecond,
  tokensPerSecondFromTpm,
  healthScoreClass,
  monitorErrorCategoryLabel,
} from '@/features/channel-monitor-v2/monitorFormat'

type Tab = 'models' | 'errors' | 'users'
type HealthMode = 'overall' | 'success' | 'ttft' | 'cache'
type TrendView = 'pulse' | 'line'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const { t, te, locale } = useI18n()
const isAdmin = computed(() => authStore.isAdmin)
const showDetailedAnalysis = computed(() => isChannelMonitorV2DetailedAnalysisEnabled())
/** Admins always see RPM/TPM; users honor the hide-throughput system setting. */
const showThroughput = computed(() => isAdmin.value || !isChannelMonitorThroughputHidden())

const ranges = computed(() => [
  { value: '90m' as MonitorRange, label: t('channelMonitorV2.ranges.90m') },
  { value: '24h' as MonitorRange, label: t('channelMonitorV2.ranges.24h') },
  { value: '7d' as MonitorRange, label: t('channelMonitorV2.ranges.7d') },
  { value: '30d' as MonitorRange, label: t('channelMonitorV2.ranges.30d') },
])
const tabs = computed(() => [
  { value: 'models' as Tab, label: t('channelMonitorV2.tabs.models') },
  { value: 'errors' as Tab, label: t('channelMonitorV2.tabs.errors') },
  { value: 'users' as Tab, label: t('channelMonitorV2.tabs.users') },
])
const matrixGroupOptions = computed(() => [
  { value: 'platform' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platform') },
  { value: 'platform_group' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroup') },
  { value: 'platform_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformModel') },
  { value: 'platform_group_model' as MonitorMatrixGroupBy, label: t('channelMonitorV2.groupBy.platformGroupModel') },
])
const healthModeOptions = computed(() => [
  { value: 'overall' as HealthMode, label: t('channelMonitorV2.healthMode.overall') },
  { value: 'success' as HealthMode, label: t('channelMonitorV2.healthMode.success') },
  { value: 'ttft' as HealthMode, label: t('channelMonitorV2.healthMode.ttft') },
  { value: 'cache' as HealthMode, label: t('channelMonitorV2.healthMode.cache') },
])

const filter = ref<MonitorFilter>({
  range: parseRange(route.query.range),
  platforms: showDetailedAnalysis.value ? csv(route.query.platform) : [],
  groupIds: showDetailedAnalysis.value ? csv(route.query.group).map(Number).filter(Boolean) : [],
  models: showDetailedAnalysis.value ? csv(route.query.model) : [],
})
const activeTab = ref<Tab>(
  (['models', 'errors', 'users'].includes(String(route.query.tab)) ? route.query.tab : 'models') as Tab
)
const matrixGroupBy = ref<MonitorMatrixGroupBy>(
  showDetailedAnalysis.value ? parseMatrixGroupBy(route.query.group_by) : 'platform_group',
)
const healthMode = ref<HealthMode>(parseHealthMode(route.query.health_mode))
const trendView = ref<TrendView>(parseTrendView(route.query.trend_view))
const advancedOpen = ref(
  showDetailedAnalysis.value && (route.query.details === '1' || hasQueryDimensions(route.query)),
)
const advancedSection = ref<HTMLElement | null>(null)
const dimensions = ref<MonitorDimensions>({ platforms: [], groups: [], models: [] })
const snapshot = ref<MonitorSnapshot | null>(null)
const matrix = ref<MonitorMatrixResponse | null>(null)
const overviewMatrix = ref<MonitorMatrixResponse | null>(null)
const displayedGroupRates = ref<Record<number, number>>({})
const modelRows = ref<MonitorModelRow[]>([])
const errorRows = ref<MonitorErrorRow[]>([])
const userRows = ref<MonitorUserRow[]>([])
const loading = ref(false)
const tabLoading = ref(false)
const refreshing = ref(false)
const expandedErrors = ref(new Set<string>())
let controller: AbortController | null = null
let sequence = 0
let autoRefreshTimer: number | null = null
let detailScrollTimer: number | null = null

const hasDimensionFilter = computed(
  () => filter.value.platforms.length + filter.value.groupIds.length + filter.value.models.length > 0
)
const activeFilterCount = computed(
  () => filter.value.platforms.length + filter.value.groupIds.length + filter.value.models.length,
)
const overviewRows = computed(() =>
  (overviewMatrix.value?.items || []).filter(
    (row) => row.group_id != null && Number(row.group_id) > 0,
  ),
)
const overviewCounts = computed(() => {
  const counts = { total: overviewRows.value.length, healthy: 0, attention: 0, unknown: 0 }
  for (const row of overviewRows.value) {
    if (!row.metrics.request_count || row.health.overall === 'unknown') counts.unknown += 1
    else if (row.health.overall === 'healthy') counts.healthy += 1
    else counts.attention += 1
  }
  return counts
})
// Full platform catalog (never pruned). Groups/models cascade by selected platforms
// so choosing a platform narrows the other pickers without collapsing platforms.
const platformOptions = computed(() =>
  (dimensions.value.platforms || []).map((item) => ({
    value: item.value,
    label: item.label,
  }))
)
const selectedPlatforms = computed(() => new Set(filter.value.platforms))
const groupOptions = computed(() =>
  (dimensions.value.groups || [])
    .filter(
      (item) =>
        selectedPlatforms.value.size === 0 ||
        !item.platform ||
        selectedPlatforms.value.has(item.platform),
    )
    .map((item) => ({
      value: String(item.id),
      label: item.platform ? `${item.platform} / ${item.name || `#${item.id}`}` : item.name || `#${item.id}`,
    }))
)
const modelOptions = computed(() =>
  (dimensions.value.models || [])
    .filter(
      (item) =>
        selectedPlatforms.value.size === 0 ||
        !item.platform ||
        selectedPlatforms.value.has(item.platform),
    )
    .map((item) => ({
      value: item.value,
      label:
        item.platform && !item.label.includes(item.platform)
          ? `${item.platform} / ${item.label}`
          : item.label,
    }))
)
const selectedGroupIds = computed({
  get: () => filter.value.groupIds.map(String),
  set: (value: string[]) => {
    filter.value.groupIds = value.map(Number).filter((id) => Number.isInteger(id) && id > 0)
  },
})
// Soft-prune group/model selections that fall outside the platform cascade.
// Do NOT wipe when options are temporarily empty (loading); only drop invalid ids.
watch(
  [groupOptions, modelOptions],
  () => {
    if (groupOptions.value.length > 0) {
      const allowed = new Set(groupOptions.value.map((item) => item.value))
      const next = filter.value.groupIds.filter((id) => allowed.has(String(id)))
      if (next.length !== filter.value.groupIds.length) {
        filter.value.groupIds = next
      }
    }
    if (modelOptions.value.length > 0) {
      const allowed = new Set(modelOptions.value.map((item) => item.value))
      const next = filter.value.models.filter((model) => allowed.has(model))
      if (next.length !== filter.value.models.length) {
        filter.value.models = next
      }
    }
  },
  { flush: 'post' },
)
const activeRowsEmpty = computed(() =>
  activeTab.value === 'models'
    ? modelRows.value.length === 0
    : activeTab.value === 'errors'
      ? errorRows.value.length === 0
      : userRows.value.length === 0
)
/** First-upgrade backfill toward 90m/24h/7d/30d; banner hides when backend omits bootstrap. */
const bootstrapActive = computed(() => Boolean(snapshot.value?.coverage?.bootstrap?.active))
const bootstrapPercent = computed(() => {
  const raw = snapshot.value?.coverage?.bootstrap?.progress_percent
  if (typeof raw !== 'number' || Number.isNaN(raw)) return 0
  return Math.min(100, Math.max(0, Math.round(raw)))
})
const matrixRows = computed(() => {
  const items = matrix.value?.items || []
  // platform_group views should only show real groups, never bare platform placeholders.
  if (matrixGroupBy.value === 'platform_group' || matrixGroupBy.value === 'platform_group_model') {
    return items.filter((row) => row.group_id != null && Number(row.group_id) > 0)
  }
  return items
})

function csv(value: unknown) {
  return typeof value === 'string' ? value.split(',').filter(Boolean) : []
}
function hasQueryDimensions(query: Record<string, unknown>) {
  return Boolean(query.platform || query.group || query.model)
}
function parseRange(value: unknown): MonitorRange {
  return ['90m', '24h', '7d', '30d'].includes(String(value)) ? (value as MonitorRange) : '90m'
}
function parseMatrixGroupBy(value: unknown): MonitorMatrixGroupBy {
  const allowed: MonitorMatrixGroupBy[] = [
    'platform',
    'platform_group',
    'platform_model',
    'platform_group_model',
  ]
  return allowed.includes(value as MonitorMatrixGroupBy)
    ? (value as MonitorMatrixGroupBy)
    : 'platform_group'
}
function parseHealthMode(value: unknown): HealthMode {
  const allowed: HealthMode[] = ['overall', 'success', 'ttft', 'cache']
  return allowed.includes(value as HealthMode) ? (value as HealthMode) : 'overall'
}
function parseTrendView(value: unknown): TrendView {
  return value === 'line' ? 'line' : 'pulse'
}
function syncQuery() {
  void router.replace({
    query: {
      range: filter.value.range,
      platform: filter.value.platforms.join(',') || undefined,
      group: filter.value.groupIds.join(',') || undefined,
      model: filter.value.models.join(',') || undefined,
      group_by: matrixGroupBy.value,
      health_mode: healthMode.value,
      trend_view: trendView.value === 'line' ? 'line' : undefined,
      tab: activeTab.value,
      details: showDetailedAnalysis.value && advancedOpen.value ? '1' : undefined,
    },
  })
}
/** Dimensions catalog: range only — never re-filtered by platform/group/model selection. */
async function loadDimensions(signal?: AbortSignal, id = sequence) {
  const rangeOnly: MonitorFilter = {
    range: filter.value.range,
    platforms: [],
    groupIds: [],
    models: [],
  }
  const next = await api.getDimensions(rangeOnly, isAdmin.value, signal)
  if (id !== sequence) return
  dimensions.value = next
}

async function loadMetrics(signal?: AbortSignal, id = sequence, refreshOverview = true) {
  const overviewFilter: MonitorFilter = {
    range: filter.value.range,
    platforms: [],
    groupIds: [],
    models: [],
  }
  const canReuseMatrix = !hasDimensionFilter.value && matrixGroupBy.value === 'platform_group'
  const overviewRequest = canReuseMatrix
    ? null
    : refreshOverview || !overviewMatrix.value
      ? api.getMatrix(overviewFilter, 'platform_group', isAdmin.value, signal)
      : Promise.resolve(overviewMatrix.value)
  const [nextSnapshot, nextMatrix, nextOverviewMatrix] = await Promise.all([
    api.getSnapshot(filter.value, isAdmin.value, signal),
    api.getMatrix(filter.value, matrixGroupBy.value, isAdmin.value, signal),
    overviewRequest,
  ])
  if (id !== sequence) return
  snapshot.value = nextSnapshot
  matrix.value = nextMatrix
  overviewMatrix.value = canReuseMatrix ? nextMatrix : nextOverviewMatrix
  scheduleAutoRefresh()
  if (showDetailedAnalysis.value) await loadTab(signal, id)
}

async function reload(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    // Catalog + metrics in parallel; catalog ignores dimension filters so options never shrink.
    await Promise.all([
      loadDimensions(request.signal, id),
      loadMetrics(request.signal, id),
    ])
  } catch (error) {
    if ((error as { name?: string }).name !== 'CanceledError') {
      appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.loadFailed')))
    }
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}

/** When only range changes, still refresh dimensions; dimension filters only re-load metrics. */
async function reloadMetricsOnly(silent = true) {
  controller?.abort()
  const request = new AbortController()
  controller = request
  const id = ++sequence
  refreshing.value = true
  if (!silent) loading.value = true
  try {
    await loadMetrics(request.signal, id, false)
  } catch (error) {
    if ((error as { name?: string }).name !== 'CanceledError') {
      appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.loadFailed')))
    }
  } finally {
    if (id === sequence) {
      loading.value = false
      tabLoading.value = false
      refreshing.value = false
    }
  }
}
async function loadTab(signal?: AbortSignal, id = sequence) {
  if (!showDetailedAnalysis.value) return
  tabLoading.value = true
  try {
    if (activeTab.value === 'models') {
      modelRows.value = (await api.getModels(filter.value, isAdmin.value, signal)).items || []
    } else if (activeTab.value === 'errors') {
      errorRows.value = (await api.getErrors(filter.value, isAdmin.value, signal)).items || []
    } else {
      userRows.value = (await api.getUsers(filter.value, isAdmin.value, signal)).items || []
    }
  } catch (error) {
    const e = error as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.name === 'CanceledError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(error, t('channelMonitorV2.detailLoadFailed')))
  } finally {
    if (id === sequence) tabLoading.value = false
  }
}
function setRange(value: MonitorRange) {
  filter.value.range = value
}
function clearDimensions() {
  // Replace arrays so deep watch always fires and metrics reload full window.
  filter.value = {
    ...filter.value,
    platforms: [],
    groupIds: [],
    models: [],
  }
}
function rateForRow(row: MonitorMatrixRow) {
  if (!row.group_id) return null
  const rate = displayedGroupRates.value[row.group_id]
  return Number.isFinite(rate) ? rate : null
}
function openGroupDetails(row: MonitorMatrixRow) {
  if (!showDetailedAnalysis.value) return
  filter.value = {
    ...filter.value,
    platforms: [row.platform],
    groupIds: row.group_id ? [row.group_id] : [],
    models: [],
  }
  advancedOpen.value = true
  if (detailScrollTimer) window.clearTimeout(detailScrollTimer)
  void nextTick(() => {
    // Query synchronization and the filtered request both update the page after
    // this click. Scroll after those first updates so router scroll restoration
    // cannot pull the user back to the selected card.
    detailScrollTimer = window.setTimeout(() => {
      advancedSection.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      detailScrollTimer = null
    }, 300)
  })
}
async function loadDisplayedGroupRates() {
  try {
    const [groupsResult, ratesResult] = await Promise.allSettled([
      userGroupsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates(),
    ])
    const next: Record<number, number> = {}
    if (groupsResult.status === 'fulfilled') {
      for (const group of groupsResult.value) {
        const rate = group.effective_rate_multiplier ?? group.rate_multiplier
        if (Number.isFinite(rate)) next[group.id] = rate
      }
    }
    if (ratesResult.status === 'fulfilled') {
      for (const [groupID, rate] of Object.entries(ratesResult.value)) {
        const id = Number(groupID)
        if (Number.isInteger(id) && Number.isFinite(rate)) next[id] = rate
      }
    }
    displayedGroupRates.value = next
  } catch {
    // Multiplier labels are optional context and must never block monitor data.
  }
}
function scheduleAutoRefresh() {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  // Poll faster while first-upgrade bootstrap is filling 90m→30d so the progress bar moves.
  const seconds = bootstrapActive.value
    ? 10
    : snapshot.value?.config?.refresh_interval_seconds || 300
  autoRefreshTimer = window.setInterval(() => {
    if (!loading.value && !refreshing.value) {
      void reload(true)
    }
  }, Math.max(bootstrapActive.value ? 10 : 60, seconds) * 1000)
}
function drillModel(row: MonitorModelRow) {
  filter.value.platforms = [row.platform]
  filter.value.models = [row.model]
}
function formatRate(value: number) {
  return formatMonitorThroughput(value)
}
function exactRate(value: number) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 2 }).format(value || 0)
}
function formatTps(tpm: number | null | undefined) {
  return formatMonitorTokensPerSecond(tpm)
}
function exactTps(tpm: number | null | undefined) {
  return Intl.NumberFormat(locale.value || undefined, { maximumFractionDigits: 3 }).format(
    tokensPerSecondFromTpm(tpm),
  )
}
function formatPercent(value: number) {
  return formatMonitorPercent(value)
}
function formatMs(value: number | null) {
  return formatMonitorMs(value)
}
function latencyDetail(metric: {
  p50_ms: number | null
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyPrivacy(metric.p50_ms, metric.p90_ms, metric.avg_ms, metric.p95_ms)
}
/** KPI secondary: AVG · P90 under the P50 primary value. */
function latencyKpiSecondary(metric: {
  p90_ms?: number | null
  p95_ms: number | null
  avg_ms?: number | null
}) {
  return formatLatencyKpiSecondary(metric.avg_ms, metric.p90_ms, metric.p95_ms)
}
function formatTime(value: string) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
function statusDot(health?: MonitorHealth | HealthState) {
  if (!health || typeof health === 'string') {
    return `status-dot health-${health || 'unknown'}`
  }
  // Prefer multi-band score when available; otherwise fall back to the coarse
  // overall state for mixed-version/older payloads.
  const klass =
    health.score != null
      ? healthScoreClass(health, 'overall', 0)
      : `health-${health.overall || 'unknown'}`
  return `status-dot ${klass}`
}
function errorLabel(value: string) {
  const key = `channelMonitorV2.errorCategories.${value}`
  return te(key) ? t(key) : monitorErrorCategoryLabel(value)
}
function toggleError(category: string) {
  const next = new Set(expandedErrors.value)
  if (next.has(category)) next.delete(category)
  else next.add(category)
  expandedErrors.value = next
}

let lastRange: MonitorRange = filter.value.range
watch(
  filter,
  () => {
    syncQuery()
    const rangeChanged = filter.value.range !== lastRange
    lastRange = filter.value.range
    if (rangeChanged) void reload(true)
    else void reloadMetricsOnly(true)
  },
  { deep: true }
)
watch(matrixGroupBy, () => {
  syncQuery()
  void reloadMetricsOnly(true)
})
watch(healthMode, syncQuery)
watch(trendView, syncQuery)
watch(advancedOpen, syncQuery)
watch(activeTab, () => {
  syncQuery()
  if (showDetailedAnalysis.value) void loadTab()
})
onMounted(() => {
  void reload(false)
  void loadDisplayedGroupRates()
})
onBeforeUnmount(() => {
  controller?.abort()
  if (autoRefreshTimer) window.clearInterval(autoRefreshTimer)
  if (detailScrollTimer) window.clearTimeout(detailScrollTimer)
})
</script>

<style scoped>
.monitor-hero {
  overflow: hidden;
  border: 1px solid rgb(226 232 240 / 0.92);
  border-radius: 1.125rem;
  background: rgb(255 255 255 / 0.92);
  box-shadow:
    0 1px 2px rgb(15 23 42 / 0.03),
    0 12px 30px rgb(15 23 42 / 0.07);
}

.monitor-hero__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.125rem 0.875rem;
}

.monitor-hero__icon {
  display: inline-flex;
  width: 2.5rem;
  height: 2.5rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(191 219 254 / 0.75);
  border-radius: 0.75rem;
  background: rgb(239 246 255 / 0.95);
  color: rgb(37 99 235);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.9);
}

.monitor-hero__live {
  display: inline-flex;
  min-height: 1.5rem;
  align-items: center;
  gap: 0.375rem;
  border-radius: 9999px;
  background: rgb(248 250 252 / 0.95);
  padding: 0.2rem 0.55rem;
  color: rgb(100 116 139);
  font-size: 0.6875rem;
  font-weight: 600;
}

.monitor-hero__controls {
  display: flex;
  max-width: 100%;
  flex: none;
  align-items: center;
  gap: 0.5rem;
}

.monitor-hero__range {
  display: inline-flex;
  align-items: center;
  gap: 0.125rem;
  border: 1px solid rgb(226 232 240);
  border-radius: 0.75rem;
  background: rgb(248 250 252);
  padding: 0.1875rem;
}

.monitor-hero__range-button {
  min-width: 2.65rem;
  border-radius: 0.55rem;
  padding: 0.35rem 0.6rem;
  color: rgb(100 116 139);
  font-size: 0.75rem;
  font-weight: 650;
  transition: color 160ms ease, background-color 160ms ease, box-shadow 160ms ease;
}

.monitor-hero__range-button:hover {
  color: rgb(30 64 175);
}

.monitor-hero__range-button.is-active {
  background: white;
  color: rgb(37 99 235);
  box-shadow: 0 1px 3px rgb(15 23 42 / 0.12), inset 0 0 0 1px rgb(191 219 254 / 0.72);
}

.monitor-hero__refresh {
  display: inline-flex;
  width: 2.125rem;
  height: 2.125rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(226 232 240);
  border-radius: 0.7rem;
  background: white;
  padding: 0;
  color: rgb(100 116 139);
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.04);
  transition: border-color 160ms ease, color 160ms ease, transform 160ms ease;
}

.monitor-hero__refresh:hover:not(:disabled) {
  border-color: rgb(147 197 253);
  color: rgb(37 99 235);
  transform: translateY(-1px);
}

.monitor-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.5rem;
  border-top: 1px solid rgb(241 245 249);
  background: rgb(248 250 252 / 0.72);
  padding: 0.625rem 1.125rem;
}

.monitor-summary__item {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: 0.65rem;
  padding: 0.35rem 0.55rem;
}

.monitor-summary__item dt {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.45rem;
  color: rgb(100 116 139);
  font-size: 0.6875rem;
  font-weight: 600;
}

.monitor-summary__item dt i {
  width: 0.4rem;
  height: 0.4rem;
  flex: none;
  border-radius: 9999px;
  background: currentColor;
  box-shadow: 0 0 0 3px rgb(148 163 184 / 0.12);
}

.monitor-summary__item dd {
  flex: none;
  color: rgb(15 23 42);
  font-size: 0.95rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.monitor-summary__item--total dt { color: rgb(59 130 246); }
.monitor-summary__item--healthy dt,
.monitor-summary__item--healthy dd { color: rgb(5 150 105); }
.monitor-summary__item--attention dt,
.monitor-summary__item--attention dd { color: rgb(217 119 6); }
.monitor-summary__item--unknown dt { color: rgb(148 163 184); }
.monitor-summary__item--unknown dd { color: rgb(100 116 139); }

:global(.dark) .monitor-hero {
  border-color: rgb(51 65 85 / 0.9);
  background: rgb(30 41 59 / 0.94);
  box-shadow: 0 14px 34px rgb(0 0 0 / 0.22);
}

:global(.dark) .monitor-hero__icon {
  border-color: rgb(30 64 175 / 0.65);
  background: rgb(30 58 138 / 0.32);
  color: rgb(147 197 253);
  box-shadow: none;
}

:global(.dark) .monitor-hero__live,
:global(.dark) .monitor-hero__range,
:global(.dark) .monitor-summary {
  border-color: rgb(51 65 85);
  background: rgb(15 23 42 / 0.42);
}

:global(.dark) .monitor-hero__range-button { color: rgb(148 163 184); }
:global(.dark) .monitor-hero__range-button:hover { color: rgb(191 219 254); }
:global(.dark) .monitor-hero__range-button.is-active {
  background: rgb(51 65 85);
  color: rgb(147 197 253);
  box-shadow: inset 0 0 0 1px rgb(59 130 246 / 0.35);
}

:global(.dark) .monitor-hero__refresh {
  border-color: rgb(51 65 85);
  background: rgb(30 41 59);
  color: rgb(148 163 184);
}

:global(.dark) .monitor-summary__item dd { color: rgb(241 245 249); }
:global(.dark) .monitor-summary__item--healthy dd { color: rgb(52 211 153); }
:global(.dark) .monitor-summary__item--attention dd { color: rgb(251 191 36); }
:global(.dark) .monitor-summary__item--unknown dd { color: rgb(148 163 184); }

@media (max-width: 767px) {
  .monitor-hero__header {
    align-items: stretch;
    flex-direction: column;
    padding: 0.875rem 0.875rem 0.75rem;
  }

  .monitor-hero__controls {
    overflow-x: auto;
  }

  .monitor-hero__range {
    flex: 1;
  }

  .monitor-hero__range-button {
    flex: 1;
  }

  .monitor-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    padding: 0.5rem 0.75rem;
  }

  .monitor-summary__item {
    padding-inline: 0.4rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .monitor-hero__range-button,
  .monitor-hero__refresh {
    transition: none;
  }
}

.status-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}
/* Multi-stop green → yellow → red score bands */
.health-score10 { background: #16a34a; }
.health-score9  { background: #22c55e; }
.health-score8  { background: #4ade80; }
.health-score7  { background: #a3e635; }
.health-score6  { background: #facc15; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #f59e0b; }
.health-score3  { background: #f97316; }
.health-score2  { background: #fb7185; }
.health-score1  { background: #f87171; }
.health-score0  { background: rgb(239, 67, 67); }
.health-healthy  { background: #22c55e; }
.health-warning  { background: #f59e0b; }
.health-critical { background: #ef4444; }
.health-unknown  { background: #9ca3af; }
.matrix-select {
  min-width: 10rem;
}
details > summary::-webkit-details-marker {
  display: none;
}
</style>

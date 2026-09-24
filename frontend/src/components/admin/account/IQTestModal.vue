<template>
  <BaseDialog :show="show" :title="t('admin.accounts.pelicanTest.title')" width="full" @close="handleClose">
    <div class="space-y-5">
      <div v-if="account" class="flex flex-col items-start gap-3 rounded-xl border border-amber-200 bg-amber-50/70 p-3 dark:border-amber-800/60 dark:bg-amber-950/20 sm:flex-row sm:items-center sm:justify-between">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500 text-white">
            <Icon name="brain" size="md" :stroke-width="2" />
          </div>
          <div>
            <div class="font-semibold text-gray-900 dark:text-gray-100">{{ account.name }}</div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ account.platform }} · {{ t('admin.accounts.pelicanTest.subtitle') }}</div>
          </div>
        </div>
        <span class="whitespace-nowrap rounded-full bg-white px-2.5 py-1 text-xs font-medium text-amber-700 shadow-sm dark:bg-dark-800 dark:text-amber-300">
          {{ t('admin.accounts.pelicanTest.noScoring') }}
        </span>
      </div>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
        <TextArea
          v-model="prompt"
          :label="t('admin.accounts.pelicanTest.promptLabel')"
          :disabled="running"
          :rows="5"
          :hint="t('admin.accounts.pelicanTest.promptHint')"
        />
        <div class="space-y-3">
          <div>
            <label for="pelican-model" class="input-label mb-1.5 block">{{ t('admin.accounts.pelicanTest.model') }}</label>
            <Select id="pelican-model" v-model="modelId" :options="modelOptions" :searchable="true" :creatable="true" :loading="loadingModels" :disabled="running || loadingModels" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.pelicanTest.modelHint') }}</p>
          </div>
          <div>
            <label for="pelican-effort" class="input-label mb-1.5 block">{{ t('admin.accounts.pelicanTest.reasoning') }}</label>
            <Select id="pelican-effort" v-model="reasoningEffort" :options="reasoningOptions" :disabled="running || loadingCapabilities" />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.pelicanTest.reasoningHint') }}</p>
          </div>
          <p v-if="settingsError" role="alert" class="text-sm text-red-600 dark:text-red-300">{{ settingsError }}</p>
          <Input
            v-model="parallelCount"
            type="number"
            :label="t('admin.accounts.pelicanTest.parallel')"
            :disabled="running"
            :hint="t('admin.accounts.pelicanTest.parallelHint')"
          />
        </div>
      </div>

      <div class="rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800/70 dark:text-gray-300">
        <div class="flex items-start gap-2">
          <Icon name="shield" size="sm" class="mt-0.5 shrink-0 text-emerald-500" />
          <span>{{ deliveryContract }}</span>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 pb-2 dark:border-dark-600">
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
            :class="activeTab === 'results' ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'"
            @click="activeTab = 'results'"
          >
            {{ t('admin.accounts.pelicanTest.results') }}
          </button>
          <button
            type="button"
            class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
            :class="activeTab === 'history' ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-dark-700'"
            @click="activeTab = 'history'"
          >
            {{ t('admin.accounts.pelicanTest.history') }}<span v-if="records.length" class="ml-1">({{ records.length }})</span>
          </button>
        </div>
        <span v-if="running" class="flex items-center gap-1.5 text-xs text-primary-600 dark:text-primary-300">
          <Icon name="refresh" size="sm" class="animate-spin" />
          {{ t('admin.accounts.pelicanTest.running', { count: runs.length }) }}
        </span>
      </div>

      <div v-if="activeTab === 'history'" class="space-y-2">
        <button v-if="records.length" type="button" class="btn btn-secondary btn-sm" :disabled="running" @click="clearHistory">{{ t('admin.accounts.pelicanTest.clearHistory') }}</button>
        <div v-if="records.length === 0" class="rounded-lg border border-dashed border-gray-300 py-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
          {{ t('admin.accounts.pelicanTest.noHistory') }}
        </div>
        <button
          v-for="record in records"
          :key="record.id"
          type="button"
          :disabled="running"
          class="flex w-full items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-left transition-colors hover:border-primary-300 hover:bg-primary-50/50 dark:border-dark-600 dark:hover:border-primary-700 dark:hover:bg-primary-900/10"
          @click="loadRecord(record)"
        >
          <span class="min-w-0">
            <span class="block truncate text-sm font-medium text-gray-800 dark:text-gray-100">{{ record.prompt }}</span>
            <span class="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">{{ formatDate(record.createdAt) }} · {{ record.modelId }} · {{ record.reasoningEffort || t('admin.accounts.pelicanTest.reasoningDefault') }} · {{ record.runs.length }} {{ t('admin.accounts.pelicanTest.outputs') }}</span>
          </span>
          <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400" />
        </button>
      </div>

      <div v-else>
        <div v-if="runs.length === 0" class="rounded-lg border border-dashed border-gray-300 py-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
          {{ t('admin.accounts.pelicanTest.emptyResults') }}
        </div>
        <div v-else class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <article v-for="(run, index) in runs" :key="run.id" class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
            <header class="flex items-center justify-between gap-2 border-b border-gray-200 px-3 py-2 dark:border-dark-600">
              <div class="flex items-center gap-2">
                <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-100 text-xs font-semibold text-primary-700 dark:bg-primary-900/50 dark:text-primary-300">{{ index + 1 }}</span>
                <span class="text-sm font-medium text-gray-800 dark:text-gray-100">{{ t('admin.accounts.pelicanTest.output') }} {{ index + 1 }}</span>
              </div>
              <div class="flex items-center gap-1">
                <span v-if="run.status === 'running'" class="text-xs text-amber-600 dark:text-amber-300">{{ t('admin.accounts.pelicanTest.runningShort') }}</span>
                <span v-else-if="run.status === 'success'" class="text-xs text-emerald-600 dark:text-emerald-300">{{ t('admin.accounts.pelicanTest.success') }}</span>
                <span v-else class="text-xs text-red-600 dark:text-red-300">{{ t('admin.accounts.pelicanTest.failed') }}</span>
                <button v-if="run.html" type="button" class="rounded-md p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-300" :title="t('admin.accounts.pelicanTest.download')" @click="downloadHtml(run)">
                  <Icon name="download" size="sm" />
                </button>
              </div>
            </header>
            <div v-if="run.html" class="aspect-[4/3] bg-white dark:bg-white">
              <iframe :srcdoc="run.html" class="h-full w-full border-0" sandbox="allow-scripts" referrerpolicy="no-referrer" :title="`${t('admin.accounts.pelicanTest.output')} ${index + 1}`"></iframe>
            </div>
            <p v-if="run.error" role="alert" class="px-3 py-2 text-sm text-red-600 dark:text-red-300">{{ run.error }}</p>
            <pre class="max-h-48 overflow-auto whitespace-pre-wrap break-words border-t border-gray-200 bg-gray-950 p-3 text-xs leading-relaxed text-gray-200 dark:border-dark-600">{{ run.output || t('admin.accounts.pelicanTest.waiting') }}</pre>
          </article>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full flex-wrap items-center justify-between gap-3">
        <button type="button" class="btn btn-secondary" :disabled="running || !hasDownloadable" @click="downloadAll">
          <Icon name="download" size="sm" />
          {{ t('admin.accounts.pelicanTest.downloadAll') }}
        </button>
        <div class="flex gap-3">
          <button type="button" class="btn btn-secondary" @click="handleClose">{{ running ? t('admin.accounts.pelicanTest.stopAndClose') : t('common.close') }}</button>
          <button type="button" class="btn btn-primary flex items-center gap-2" :disabled="running || !canStart" @click="startTest">
            <Icon v-if="running" name="refresh" size="sm" class="animate-spin" />
            <Icon v-else name="play" size="sm" />
            {{ running ? t('admin.accounts.pelicanTest.generating') : t('admin.accounts.pelicanTest.start') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import Select from '@/components/common/Select.vue'
import { Icon } from '@/components/icons'
import { getAvailableModels } from '@/api/admin/accounts'
import { getPelicanCapabilities, startPelicanRun } from '@/api/admin/pelicanTest'
import { consumePelicanStream, extractPelicanHtml, isPelicanTextModel, PELICAN_HISTORY_KEY, PELICAN_HISTORY_LIMIT, PELICAN_MAX_OUTPUT } from '@/utils/pelicanTest'
import type { Account } from '@/types'

const { t } = useI18n()
const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ (event: 'close'): void }>()
type RunStatus = 'running' | 'success' | 'error'
interface TestRun { id: string; status: RunStatus; output: string; html: string; error: string }
interface TestRecord { id: string; accountID: number; createdAt: string; prompt: string; modelId: string; reasoningEffort: string; runs: TestRun[] }
const prompt = ref('')
const modelId = ref('')
const reasoningEffort = ref('')
const parallelCount = ref<string | number>(1)
const activeTab = ref<'results' | 'history'>('results')
const running = ref(false)
const runs = ref<TestRun[]>([])
const records = ref<TestRecord[]>([])
const modelOptions = ref<{ value: string; label: string }[]>([])
const supportedEfforts = ref<string[]>([])
const loadingModels = ref(false)
const loadingCapabilities = ref(false)
const settingsError = ref('')
const controllers = new Set<AbortController>()
let capabilityController: AbortController | undefined
let generation = 0
let modelLoadID = 0
let capabilityLoadID = 0
const deliveryContract = computed(() => t('admin.accounts.pelicanTest.deliveryContract'))
const reasoningOptions = computed(() => [
  { value: '', label: t('admin.accounts.pelicanTest.reasoningDefault') },
  ...supportedEfforts.value.map(value => ({ value, label: t(`admin.accounts.pelicanTest.effort.${value}`) }))
])
const canStart = computed(() => Boolean(props.account && prompt.value.trim() && prompt.value.length <= 15000 && modelId.value.trim() && modelId.value.length <= 256 && !loadingModels.value && !loadingCapabilities.value && !settingsError.value))
const hasDownloadable = computed(() => runs.value.some(run => Boolean(run.html)))
// One bounded history store per signed-in administrator, across all accounts.
function storageKey() {
  try { return `${PELICAN_HISTORY_KEY}:${JSON.parse(localStorage.getItem('auth_user') || '{}').id || 'local'}` } catch { return `${PELICAN_HISTORY_KEY}:local` }
}
function normalizeCount() {
  const count = Number(parallelCount.value)
  return Number.isFinite(count) ? Math.min(8, Math.max(1, Math.floor(count))) : 1
}
function readHistory(): TestRecord[] {
  try {
    const raw = localStorage.getItem(storageKey()) || '[]'
    if (raw.length > PELICAN_HISTORY_LIMIT) return []
    const items: unknown = JSON.parse(raw)
    if (!Array.isArray(items)) return []
    return items.slice(0, 8).filter((item): item is TestRecord => Boolean(item && typeof item.id === 'string' && Number.isInteger(item.accountID) && typeof item.prompt === 'string' && item.prompt.length <= 15000 && typeof item.modelId === 'string' && typeof item.reasoningEffort === 'string' && typeof item.createdAt === 'string' && Number.isFinite(Date.parse(item.createdAt)) && Array.isArray(item.runs) && item.runs.length <= 8 && item.runs.every((run: TestRun) => run && typeof run.id === 'string' && typeof run.output === 'string' && run.output.length <= PELICAN_MAX_OUTPUT && typeof run.error === 'string' && ['success', 'error'].includes(run.status))))
  } catch { return [] }
}
function readRecords() { records.value = readHistory().filter(record => record.accountID === props.account?.id) }
function saveRecord(record: TestRecord, key: string) {
  if (key !== storageKey()) return
  // Store raw output only. Never restore executable preview HTML from storage.
  const history = [record, ...readHistory()].slice(0, 8).map(item => ({ ...item, runs: item.runs.map(run => ({ ...run, html: '' })) }))
  while (history.length && JSON.stringify(history).length > PELICAN_HISTORY_LIMIT) history.pop()
  try { localStorage.setItem(key, JSON.stringify(history)) } catch { /* Current results remain available for download. */ }
  readRecords()
}
function clearHistory() {
  const history = readHistory().filter(record => record.accountID !== props.account?.id)
  try { localStorage.setItem(storageKey(), JSON.stringify(history)) } catch { /* Storage can be disabled. */ }
  records.value = []
}
function formatDate(value: string) { return new Intl.DateTimeFormat(undefined, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value)) }
function loadRecord(record: TestRecord) {
  if (running.value) return
  prompt.value = record.prompt
  modelId.value = record.modelId
  reasoningEffort.value = record.reasoningEffort
  runs.value = record.runs.map(run => ({ ...run, html: run.status === 'success' ? extractPelicanHtml(run.output) : '' }))
  activeTab.value = 'results'
}
function cancelRuns() {
  generation++
  for (const controller of controllers) controller.abort()
  controllers.clear()
  running.value = false
}
function handleClose() { cancelRuns(); emit('close') }
function errorText(error: unknown) {
  return typeof error === 'object' && error && 'message' in error ? String(error.message) : t('admin.accounts.pelicanTest.failed')
}
async function loadCapabilities() {
  const id = ++capabilityLoadID
  capabilityController?.abort()
  supportedEfforts.value = []
  settingsError.value = ''
  if (!props.show || !props.account || !modelId.value.trim()) { loadingCapabilities.value = false; return }
  const controller = new AbortController()
  capabilityController = controller
  loadingCapabilities.value = true
  try {
    const result = await getPelicanCapabilities(props.account.id, modelId.value.trim(), controller.signal)
    if (id !== capabilityLoadID || controller.signal.aborted) return
    supportedEfforts.value = result.supported_reasoning_levels || []
    if (!supportedEfforts.value.includes(reasoningEffort.value)) {
      reasoningEffort.value = supportedEfforts.value.includes('medium') ? 'medium' : ''
    }
  } catch (error) {
    if (id === capabilityLoadID && !controller.signal.aborted) settingsError.value = errorText(error)
  } finally { if (id === capabilityLoadID) loadingCapabilities.value = false }
}
async function initialize() {
  const id = ++modelLoadID
  cancelRuns()
  capabilityController?.abort()
  capabilityLoadID++
  loadingCapabilities.value = false
  modelOptions.value = []
  supportedEfforts.value = []
  settingsError.value = ''
  modelId.value = ''
  runs.value = []
  if (!props.show || !props.account) { loadingModels.value = false; return }
  const accountID = props.account.id
  readRecords()
  prompt.value = t('admin.accounts.pelicanTest.defaultPrompt')
  reasoningEffort.value = 'medium'
  parallelCount.value = 1
  activeTab.value = 'results'
  loadingModels.value = true
  try {
    const models = await getAvailableModels(accountID)
    if (id !== modelLoadID) return
    modelOptions.value = models.filter(model => isPelicanTextModel(model.id)).map(model => ({ value: model.id, label: model.display_name ? `${model.display_name} · ${model.id}` : model.id }))
    modelId.value = modelOptions.value.find(model => model.value === 'gpt-6-astra')?.value || modelOptions.value[0]?.value || ''
  } catch { /* A custom model can still be entered and validated by the server. */ }
  finally { if (id === modelLoadID) loadingModels.value = false }
}
async function startTest() {
  if (running.value || !canStart.value || !props.account) return
  const id = ++generation
  const accountID = props.account.id
  const key = storageKey()
  const snapshot = { prompt: prompt.value.trim(), modelId: modelId.value.trim(), reasoningEffort: reasoningEffort.value }
  const payload = { model_id: snapshot.modelId, prompt: `${snapshot.prompt}\n\n${deliveryContract.value}`, reasoning_effort: snapshot.reasoningEffort }
  parallelCount.value = normalizeCount()
  runs.value = Array.from({ length: Number(parallelCount.value) }, (_, index) => ({ id: `${Date.now()}-${id}-${index}`, status: 'running', output: '', html: '', error: '' }))
  const batch = runs.value
  running.value = true
  activeTab.value = 'results'
  await Promise.all(batch.map(async run => {
    const controller = new AbortController()
    controllers.add(controller)
    try {
      const response = await startPelicanRun(accountID, payload, controller.signal)
      await consumePelicanStream(response, text => { run.output += text }, t('admin.accounts.pelicanTest.incompleteResponse'))
      if (controller.signal.aborted) return
      run.html = extractPelicanHtml(run.output)
      if (!run.html) throw new Error(t('admin.accounts.pelicanTest.invalidHtml'))
      run.status = 'success'
    } catch (error) {
      run.status = 'error'
      run.error = errorText(error)
    } finally { controller.abort(); controllers.delete(controller) }
  }))
  if (id !== generation) return
  running.value = false
  if (key !== storageKey()) return
  saveRecord({ id: `${Date.now()}-${id}`, accountID, createdAt: new Date().toISOString(), ...snapshot, runs: batch.map(run => ({ ...run })) }, key)
}
function downloadHtml(run: TestRun) {
  const content = extractPelicanHtml(run.output)
  if (!content) return
  const url = URL.createObjectURL(new Blob([content], { type: 'text/html;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = `pelican-${run.id}.html`
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
function downloadAll() { runs.value.filter(run => run.html).forEach(downloadHtml) }
watch(() => [props.show, props.account?.id], initialize, { immediate: true })
watch(modelId, loadCapabilities)
onBeforeUnmount(() => { cancelRuns(); modelLoadID++; capabilityLoadID++; capabilityController?.abort() })
</script>

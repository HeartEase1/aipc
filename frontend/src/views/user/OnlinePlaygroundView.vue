<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px] space-y-4">
      <header class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-5">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="flex items-start gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600 ring-1 ring-primary-100 dark:bg-primary-950/30 dark:text-primary-300 dark:ring-primary-900/50">
              <Icon name="sparkles" size="lg" :stroke-width="1.8" />
            </span>
            <div class="min-w-0">
              <h1 class="text-xl font-bold text-gray-900 dark:text-white">
                {{ t('onlinePlayground.title') }}
              </h1>
              <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-dark-400">
                {{ t('onlinePlayground.description') }}
              </p>
              <div class="mt-2 flex flex-wrap gap-2">
                <span class="inline-flex items-center gap-1.5 rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700 dark:bg-blue-950/30 dark:text-blue-300">
                  <Icon name="chat" size="xs" />
                  {{ t('onlinePlayground.textChat') }}
                </span>
                <span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300">
                  <Icon name="sparkles" size="xs" />
                  {{ t('onlinePlayground.imageGeneration') }}
                </span>
              </div>
            </div>
          </div>

          <div
            v-if="canMountPlayground"
            class="inline-flex w-fit shrink-0 items-center gap-2 rounded-full px-3 py-1.5 text-xs font-medium"
            :class="connectionStatusClass"
            role="status"
          >
            <span
              class="h-2 w-2 rounded-full"
              :class="connectionFailed ? 'bg-red-500' : iframeReady ? 'bg-emerald-500' : 'animate-pulse bg-amber-500'"
            />
            {{ connectionStatusLabel }}
          </div>
        </div>
      </header>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:p-5" aria-labelledby="playground-config-title">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-end">
          <div class="min-w-0 flex-1">
            <h2 id="playground-config-title" class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('onlinePlayground.configuration') }}
            </h2>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
              {{ t('onlinePlayground.keyPrivacyNotice') }}
            </p>
          </div>

          <button
            type="button"
            class="btn btn-secondary w-full shrink-0 xl:w-auto"
            :disabled="keysLoading || modelsLoading || activeKeys.length === 0"
            :title="t('onlinePlayground.refreshModels')"
            @click="refreshModels"
          >
            <Icon name="refresh" size="sm" class="mr-2" :class="keysLoading || modelsLoading ? 'animate-spin' : ''" />
            {{ t('onlinePlayground.refreshModels') }}
          </button>
        </div>

        <div class="mt-4 grid gap-4 md:grid-cols-3">
          <label class="min-w-0">
            <span class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-dark-200">
              <Icon name="key" size="sm" class="text-gray-400" />
              {{ t('onlinePlayground.selectKey') }}
            </span>
            <Select
              v-model="selectedKeyId"
              :options="keyOptions"
              :placeholder="t('onlinePlayground.selectKeyPlaceholder')"
              :disabled="keysLoading || activeKeys.length === 0"
              searchable
            />
          </label>

          <label class="min-w-0">
            <span class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-dark-200">
              <Icon name="chat" size="sm" class="text-gray-400" />
              {{ t('onlinePlayground.selectTextModel') }}
            </span>
            <Select
              v-model="selectedTextModel"
              :options="textModelOptions"
              :placeholder="t('onlinePlayground.selectModelPlaceholder')"
              :disabled="modelsLoading || textModelOptions.length === 0"
              searchable
            />
          </label>

          <label class="min-w-0">
            <span class="mb-1.5 flex items-center gap-1.5 text-sm font-medium text-gray-700 dark:text-dark-200">
              <Icon name="sparkles" size="sm" class="text-gray-400" />
              {{ t('onlinePlayground.selectImageModel') }}
            </span>
            <Select
              v-model="selectedImageModel"
              :options="imageModelOptions"
              :placeholder="t('onlinePlayground.selectModelPlaceholder')"
              :disabled="modelsLoading || imageModelOptions.length === 0"
              searchable
            />
          </label>
        </div>

        <div v-if="keysLoading" class="mt-4 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400" role="status">
          <span class="h-4 w-4 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
          {{ t('onlinePlayground.loadingKeys') }}
        </div>

        <div v-else-if="keysLoadFailed" class="mt-4 flex flex-col gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center sm:justify-between" role="alert">
          <span class="flex items-start gap-2">
            <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
            {{ t('onlinePlayground.keysLoadFailed') }}
          </span>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="loadKeys">
            {{ t('onlinePlayground.retry') }}
          </button>
        </div>

        <div v-else-if="activeKeys.length === 0" class="mt-4 flex flex-col gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/20 dark:text-amber-200 sm:flex-row sm:items-center sm:justify-between">
          <span class="flex items-start gap-2">
            <Icon name="key" size="sm" class="mt-0.5 shrink-0" />
            <span>
              <strong class="block font-semibold">{{ t('onlinePlayground.noActiveKey') }}</strong>
              <span class="mt-0.5 block text-xs opacity-80">{{ t('onlinePlayground.createKeyHint') }}</span>
            </span>
          </span>
          <router-link to="/keys" class="btn btn-primary btn-sm shrink-0">
            {{ t('onlinePlayground.goToKeys') }}
          </router-link>
        </div>

        <div v-else-if="modelsLoading" class="mt-4 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400" role="status">
          <span class="h-4 w-4 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
          {{ t('onlinePlayground.loadingModels') }}
        </div>

        <div v-else-if="modelsLoadFailed" class="mt-4 flex flex-col gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300 sm:flex-row sm:items-center sm:justify-between" role="alert">
          <span class="flex items-start gap-2">
            <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
            <span>
              <strong class="block font-semibold">{{ t('onlinePlayground.modelsLoadFailed') }}</strong>
              <span v-if="modelRequestStatus" class="mt-0.5 block text-xs opacity-80">
                {{ t('onlinePlayground.httpStatus', { status: modelRequestStatus }) }}
              </span>
            </span>
          </span>
          <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="loadModels">
            {{ t('onlinePlayground.retry') }}
          </button>
        </div>
      </section>

      <section
        v-if="canMountPlayground"
        ref="workspaceRef"
        class="playground-workspace relative overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900"
        aria-labelledby="playground-workspace-title"
      >
        <div class="flex h-11 items-center justify-between border-b border-gray-200 bg-gray-50 px-3 dark:border-dark-700 dark:bg-dark-800 sm:px-4">
          <h2 id="playground-workspace-title" class="flex min-w-0 items-center gap-2 text-sm font-semibold text-gray-700 dark:text-dark-200">
            <Icon name="sparkles" size="sm" class="shrink-0 text-primary-500" />
            <span class="truncate">{{ t('onlinePlayground.workspace') }}</span>
          </h2>

          <button
            v-if="fullscreenSupported && iframeReady"
            type="button"
            class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-white hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 disabled:cursor-wait disabled:opacity-60 dark:text-dark-300 dark:hover:bg-dark-700 dark:hover:text-white dark:focus-visible:ring-offset-dark-800"
            :aria-label="fullscreenButtonLabel"
            :aria-pressed="isFullscreen"
            :disabled="fullscreenPending"
            :title="fullscreenButtonLabel"
            @click="toggleFullscreen"
          >
            <svg v-if="isFullscreen" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v4a2 2 0 01-2 2H3m18 0h-4a2 2 0 01-2-2V3M3 15h4a2 2 0 012 2v4m6 0v-4a2 2 0 012-2h4" />
            </svg>
            <svg v-else class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
          </button>
        </div>

        <iframe
          :key="sessionId"
          ref="iframeRef"
          :src="iframeSrc"
          :title="t('onlinePlayground.iframeTitle')"
          class="playground-frame block w-full border-0 bg-white dark:bg-dark-950"
          sandbox="allow-downloads allow-forms allow-modals allow-same-origin allow-scripts"
          allow="clipboard-write"
          referrerpolicy="no-referrer"
          @load="handleIframeLoad"
        />

        <div
          v-if="!iframeReady"
          class="absolute inset-0 flex items-center justify-center bg-white/95 p-6 text-center backdrop-blur-sm dark:bg-dark-900/95"
        >
          <div v-if="connectionFailed" class="max-w-md">
            <span class="mx-auto flex h-11 w-11 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-950/30 dark:text-red-300">
              <Icon name="exclamationCircle" size="lg" />
            </span>
            <p class="mt-3 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('onlinePlayground.connectionFailed') }}
            </p>
            <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
              {{ t('onlinePlayground.connectionFailedHint') }}
            </p>
            <button type="button" class="btn btn-primary btn-sm mt-4" @click="reconnectPlayground">
              <Icon name="refresh" size="sm" class="mr-2" />
              {{ t('onlinePlayground.reconnect') }}
            </button>
          </div>
          <div v-else role="status">
            <span class="mx-auto block h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" />
            <p class="mt-3 text-sm font-medium text-gray-700 dark:text-dark-200">
              {{ t('onlinePlayground.connecting') }}
            </p>
          </div>
        </div>
      </section>

      <div v-if="canMountPlayground" class="flex items-start gap-2 px-1 text-xs leading-5 text-gray-500 dark:text-dark-400">
        <Icon name="infoCircle" size="xs" class="mt-0.5 shrink-0" />
        <span>{{ t('onlinePlayground.localHistoryNotice') }}</span>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'
import {
  PLAYGROUND_CLEAR_MESSAGE,
  PLAYGROUND_CONFIG_MESSAGE,
  PlaygroundModelRequestError,
  isHostedPlaygroundReadyEvent,
  playgroundAPI,
  type PlaygroundModels,
} from '@/api/playground'

const { t } = useI18n()
const authStore = useAuthStore()

const activeKeys = ref<ApiKey[]>([])
const selectedKeyId = ref<number | null>(null)
const models = ref<PlaygroundModels>({ all: [], text: [], image: [] })
const selectedTextModel = ref<string | null>(null)
const selectedImageModel = ref<string | null>(null)
const keysLoading = ref(false)
const modelsLoading = ref(false)
const keysLoadFailed = ref(false)
const modelsLoadFailed = ref(false)
const modelRequestStatus = ref<number | null>(null)
const iframeRef = ref<HTMLIFrameElement | null>(null)
const workspaceRef = ref<HTMLElement | null>(null)
const iframeReady = ref(false)
const connectionFailed = ref(false)
const isFullscreen = ref(false)
const fullscreenPending = ref(false)
const theme = ref<'light' | 'dark'>(
  document.documentElement.classList.contains('dark') ? 'dark' : 'light',
)
const sessionId = ref(createSessionId())

let keysAbortController: AbortController | null = null
let modelsAbortController: AbortController | null = null
let connectionTimer: number | null = null
let themeObserver: MutationObserver | null = null

const userId = computed(() => {
  const id = authStore.user?.id
  return Number.isSafeInteger(id) && Number(id) > 0 ? String(id) : ''
})

const selectedKey = computed(() => (
  activeKeys.value.find((key) => key.id === selectedKeyId.value) ?? null
))

const keyOptions = computed(() => activeKeys.value.map((key) => ({
  value: key.id,
  label: `${key.name || t('onlinePlayground.unnamedKey')} (#${key.id})`,
})))

const textModelOptions = computed(() => models.value.text.map((model) => ({
  value: model.id,
  label: model.id,
})))

const imageModelOptions = computed(() => models.value.image.map((model) => ({
  value: model.id,
  label: model.id,
})))

const canMountPlayground = computed(() => Boolean(
  userId.value
  && selectedKey.value
  && selectedTextModel.value
  && selectedImageModel.value
  && !modelsLoading.value
  && !modelsLoadFailed.value,
))

const fullscreenSupported = computed(() => Boolean(
  workspaceRef.value?.requestFullscreen && document.exitFullscreen,
))

const fullscreenButtonLabel = computed(() => (
  isFullscreen.value
    ? t('onlinePlayground.exitFullscreen')
    : t('onlinePlayground.enterFullscreen')
))

const iframeSrc = computed(() => {
  const params = new URLSearchParams({
    hosted: '1',
    user: userId.value,
    session: sessionId.value,
  })
  return `/playground-app/?${params.toString()}`
})

const connectionStatusLabel = computed(() => {
  if (connectionFailed.value) return t('onlinePlayground.connectionFailed')
  return iframeReady.value
    ? t('onlinePlayground.connected')
    : t('onlinePlayground.connecting')
})

const connectionStatusClass = computed(() => {
  if (connectionFailed.value) {
    return 'bg-red-50 text-red-700 dark:bg-red-950/30 dark:text-red-300'
  }
  if (iframeReady.value) {
    return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300'
  }
  return 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
})

function createSessionId(): string {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID()

  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
}

function resetModels() {
  models.value = { all: [], text: [], image: [] }
  selectedTextModel.value = null
  selectedImageModel.value = null
  modelsLoadFailed.value = false
  modelRequestStatus.value = null
}

function clearConnectionTimer() {
  if (connectionTimer !== null) {
    window.clearTimeout(connectionTimer)
    connectionTimer = null
  }
}

function sendClearMessage() {
  iframeRef.value?.contentWindow?.postMessage({
    type: PLAYGROUND_CLEAR_MESSAGE,
    sessionId: sessionId.value,
  }, window.location.origin)
}

function sendHostedConfig() {
  const target = iframeRef.value?.contentWindow
  const key = selectedKey.value
  if (
    !target
    || !iframeReady.value
    || !key
    || !userId.value
    || !selectedTextModel.value
    || !selectedImageModel.value
  ) return

  target.postMessage({
    type: PLAYGROUND_CONFIG_MESSAGE,
    version: 1,
    sessionId: sessionId.value,
    userId: userId.value,
    keyId: key.id,
    apiKey: key.key,
    baseUrl: new URL('/v1', window.location.origin).toString().replace(/\/$/, ''),
    textModel: selectedTextModel.value,
    imageModel: selectedImageModel.value,
    theme: theme.value,
  }, window.location.origin)
}

function handleWindowMessage(event: MessageEvent) {
  if (!isHostedPlaygroundReadyEvent(event, {
    source: iframeRef.value?.contentWindow ?? null,
    origin: window.location.origin,
    sessionId: sessionId.value,
    userId: userId.value,
  })) return

  clearConnectionTimer()
  connectionFailed.value = false
  iframeReady.value = true
  sendHostedConfig()
}

function handleIframeLoad() {
  clearConnectionTimer()
  if (iframeReady.value) return

  connectionFailed.value = false
  connectionTimer = window.setTimeout(() => {
    if (!iframeReady.value) connectionFailed.value = true
  }, 12_000)
}

async function loadKeys() {
  keysAbortController?.abort()
  modelsAbortController?.abort()
  const controller = new AbortController()
  keysAbortController = controller
  keysLoading.value = true
  keysLoadFailed.value = false
  sendClearMessage()
  iframeReady.value = false
  resetModels()

  try {
    const loadedKeys = await playgroundAPI.listActiveKeys(controller.signal)
    if (controller.signal.aborted || keysAbortController !== controller) return
    activeKeys.value = loadedKeys
    const existingSelection = loadedKeys.some((key) => key.id === selectedKeyId.value)
    selectedKeyId.value = existingSelection ? selectedKeyId.value : (loadedKeys[0]?.id ?? null)
    if (selectedKeyId.value !== null) await loadModels()
  } catch (error) {
    if ((error as { name?: string })?.name === 'CanceledError') return
    activeKeys.value = []
    selectedKeyId.value = null
    keysLoadFailed.value = true
  } finally {
    if (keysAbortController === controller) keysLoading.value = false
  }
}

async function loadModels() {
  modelsAbortController?.abort()
  const key = selectedKey.value
  if (!key) {
    resetModels()
    return
  }

  const controller = new AbortController()
  modelsAbortController = controller
  const requestedKeyId = key.id
  modelsLoading.value = true
  modelsLoadFailed.value = false
  modelRequestStatus.value = null
  sendClearMessage()
  iframeReady.value = false
  clearConnectionTimer()

  const previousTextModel = selectedTextModel.value
  const previousImageModel = selectedImageModel.value
  resetModels()

  try {
    const loadedModels = await playgroundAPI.fetchModels(key.key, controller.signal)
    if (
      controller.signal.aborted
      || modelsAbortController !== controller
      || selectedKeyId.value !== requestedKeyId
    ) return
    models.value = loadedModels
    selectedTextModel.value = loadedModels.text.some((model) => model.id === previousTextModel)
      ? previousTextModel
      : (loadedModels.text[0]?.id ?? null)
    selectedImageModel.value = loadedModels.image.some((model) => model.id === previousImageModel)
      ? previousImageModel
      : (loadedModels.image[0]?.id ?? null)
  } catch (error) {
    if ((error as { name?: string })?.name === 'AbortError') return
    modelsLoadFailed.value = true
    modelRequestStatus.value = error instanceof PlaygroundModelRequestError && error.status > 0
      ? error.status
      : null
  } finally {
    if (modelsAbortController === controller) modelsLoading.value = false
  }
}

async function refreshModels() {
  if (activeKeys.value.length === 0) {
    await loadKeys()
    return
  }
  await loadModels()
}

async function reconnectPlayground() {
  sendClearMessage()
  clearConnectionTimer()
  iframeReady.value = false
  connectionFailed.value = false
  sessionId.value = createSessionId()
  await nextTick()
}

function handleFullscreenChange() {
  isFullscreen.value = document.fullscreenElement === workspaceRef.value
}

async function toggleFullscreen() {
  const workspace = workspaceRef.value
  if (!workspace || !fullscreenSupported.value || fullscreenPending.value) return

  fullscreenPending.value = true
  try {
    if (document.fullscreenElement === workspace) {
      await document.exitFullscreen()
    } else {
      await workspace.requestFullscreen()
    }
  } catch (error) {
    console.warn('Unable to change playground fullscreen state', error)
  } finally {
    handleFullscreenChange()
    fullscreenPending.value = false
  }
}

watch(selectedKeyId, (next, previous) => {
  if (next === previous || keysLoading.value) return
  void loadModels()
})

watch([selectedTextModel, selectedImageModel, theme], () => {
  sendHostedConfig()
})

watch(userId, (next, previous) => {
  if (next === previous) return
  sendClearMessage()
  keysAbortController?.abort()
  modelsAbortController?.abort()
  activeKeys.value = []
  selectedKeyId.value = null
  resetModels()
  sessionId.value = createSessionId()
  if (next) void loadKeys()
})

onMounted(() => {
  window.addEventListener('message', handleWindowMessage)
  document.addEventListener('fullscreenchange', handleFullscreenChange)
  themeObserver = new MutationObserver(() => {
    theme.value = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
  if (userId.value) void loadKeys()
})

onBeforeUnmount(() => {
  sendClearMessage()
  clearConnectionTimer()
  keysAbortController?.abort()
  modelsAbortController?.abort()
  themeObserver?.disconnect()
  window.removeEventListener('message', handleWindowMessage)
  document.removeEventListener('fullscreenchange', handleFullscreenChange)
})
</script>

<style scoped>
.playground-workspace:fullscreen {
  display: flex;
  width: 100vw;
  height: 100dvh;
  max-height: 100dvh;
  flex-direction: column;
  border: 0;
  border-radius: 0;
}

.playground-frame {
  height: max(42rem, calc(100dvh - 19rem));
  max-height: 72rem;
}

.playground-workspace:fullscreen .playground-frame {
  min-height: 0;
  height: auto;
  max-height: none;
  flex: 1 1 auto;
}

@media (min-width: 1024px) {
  .playground-frame {
    height: max(44rem, calc(100dvh - 15rem));
  }
}
</style>

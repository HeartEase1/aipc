import type { AppSettings } from '../types'

export const HOSTED_CONFIG_MESSAGE = 'ipcai:playground-config'
export const HOSTED_CLEAR_MESSAGE = 'ipcai:playground-clear'
export const HOSTED_READY_MESSAGE = 'ipcai:playground-ready'

const HOSTED_STORAGE_PREFIX = 'ipcai-playground:v1:user:'
const HOSTED_SERVICE_WORKER_PATH = '/playground-app/'
const HOSTED_SENSITIVE_QUERY_KEYS = [
  'settings',
  'apiUrl',
  'apiKey',
  'codexCli',
  'apiMode',
  'model',
  'profileName',
  'reasoningEffort',
  'streamImages',
  'streamPartialImages',
]

const searchParams = typeof window === 'undefined'
  ? new URLSearchParams()
  : new URLSearchParams(window.location.search)

const requestedUserId = searchParams.get('user')?.trim() ?? ''
const requestedSessionId = searchParams.get('session')?.trim() ?? ''

export const isHostedModeRequested = searchParams.get('hosted') === '1'
export const isHostedMode = isHostedModeRequested
  && typeof window !== 'undefined'
  && window.parent !== window
  && /^[1-9]\d{0,18}$/.test(requestedUserId)
  && /^[a-f0-9-]{16,64}$/i.test(requestedSessionId)

export const hostedUserId = isHostedMode ? requestedUserId : ''

export interface HostedPlaygroundConfig {
  type: typeof HOSTED_CONFIG_MESSAGE
  version: 1
  sessionId: string
  userId: string
  keyId: number
  apiKey: string
  baseUrl: string
  textModel: string
  imageModel: string
  theme: 'light' | 'dark'
}

let hostedConfig: HostedPlaygroundConfig | null = null
const hostedConfigListeners = new Set<(config: HostedPlaygroundConfig | null) => void>()

export function getHostedConfig() {
  return hostedConfig
}

export function getPlaygroundStorageName() {
  return isHostedMode
    ? `${HOSTED_STORAGE_PREFIX}${hostedUserId}`
    : 'gpt-image-playground'
}

export function scrubHostedSettings(settings: AppSettings): AppSettings {
  if (!isHostedMode) return settings

  return {
    ...settings,
    apiKey: '',
    profiles: settings.profiles.map((profile) => ({
      ...profile,
      apiKey: '',
    })),
  }
}

export function protectHostedRuntimeSettings(
  current: AppSettings,
  candidate: AppSettings,
): AppSettings {
  if (!isHostedMode) return candidate

  return {
    ...candidate,
    baseUrl: current.baseUrl,
    apiKey: current.apiKey,
    model: current.model,
    timeout: current.timeout,
    apiMode: current.apiMode,
    codexCli: current.codexCli,
    apiProxy: current.apiProxy,
    streamImages: current.streamImages,
    streamPartialImages: current.streamPartialImages,
    customProviders: current.customProviders,
    providerOrder: current.providerOrder,
    reuseTaskApiProfileTemporarily: false,
    allowPromptRewrite: false,
    zipDownloadRoutes: [],
    agentMaxToolRounds: current.agentMaxToolRounds,
    agentWebSearch: false,
    agentApiConfigMode: current.agentApiConfigMode,
    agentTextProfileId: current.agentTextProfileId,
    agentImageProfileId: current.agentImageProfileId,
    profiles: current.profiles,
    activeProfileId: current.activeProfileId,
  }
}

export function protectHostedExportOptions<T extends { exportConfig?: boolean, exportTasks?: boolean }>(options: T): T {
  return isHostedMode ? { ...options, exportConfig: false } : options
}

export function protectHostedImportOptions<T extends { importConfig?: boolean, importTasks?: boolean }>(options: T): T {
  return isHostedMode ? { ...options, importConfig: false } : options
}

function parseHostedConfig(event: MessageEvent): HostedPlaygroundConfig | null {
  if (!isHostedMode || event.source !== window.parent || event.origin !== window.location.origin) {
    return null
  }
  if (!event.data || typeof event.data !== 'object') return null

  const value = event.data as Partial<HostedPlaygroundConfig>
  if (
    value.type !== HOSTED_CONFIG_MESSAGE
    || value.version !== 1
    || value.sessionId !== requestedSessionId
    || value.userId !== hostedUserId
    || !Number.isSafeInteger(value.keyId)
    || Number(value.keyId) <= 0
    || typeof value.apiKey !== 'string'
    || !value.apiKey.trim()
    || value.apiKey.length > 2048
    || typeof value.baseUrl !== 'string'
    || !value.baseUrl.trim()
    || value.baseUrl.length > 2048
    || typeof value.textModel !== 'string'
    || !value.textModel.trim()
    || value.textModel.length > 256
    || typeof value.imageModel !== 'string'
    || !value.imageModel.trim()
    || value.imageModel.length > 256
    || (value.theme !== 'light' && value.theme !== 'dark')
  ) {
    return null
  }

  try {
    const baseUrl = new URL(value.baseUrl, window.location.origin)
    if (
      baseUrl.origin !== window.location.origin
      || baseUrl.pathname.replace(/\/+$/, '') !== '/v1'
      || baseUrl.search
      || baseUrl.hash
      || baseUrl.username
      || baseUrl.password
    ) {
      return null
    }
  } catch {
    return null
  }

  const config: HostedPlaygroundConfig = {
    type: HOSTED_CONFIG_MESSAGE,
    version: 1,
    sessionId: requestedSessionId,
    userId: hostedUserId,
    keyId: value.keyId as number,
    apiKey: value.apiKey.trim(),
    baseUrl: `${window.location.origin}/v1`,
    textModel: value.textModel.trim(),
    imageModel: value.imageModel.trim(),
    theme: value.theme,
  }

  if (hostedConfig && (
    hostedConfig.userId !== config.userId
    || hostedConfig.sessionId !== config.sessionId
  )) {
    return null
  }

  return config
}

function isHostedClearMessage(event: MessageEvent) {
  if (!isHostedMode || event.source !== window.parent || event.origin !== window.location.origin) {
    return false
  }
  if (!event.data || typeof event.data !== 'object') return false

  const value = event.data as { type?: unknown, sessionId?: unknown }
  return value.type === HOSTED_CLEAR_MESSAGE && value.sessionId === requestedSessionId
}

function handleHostedMessage(event: MessageEvent) {
  const config = parseHostedConfig(event)
  if (config) {
    hostedConfig = config
    document.documentElement.classList.toggle('dark', config.theme === 'dark')
    for (const listener of hostedConfigListeners) listener(config)
    return
  }

  if (isHostedClearMessage(event)) {
    hostedConfig = null
    for (const listener of hostedConfigListeners) listener(null)
  }
}

export function waitForHostedConfig(timeoutMs = 10_000): Promise<HostedPlaygroundConfig> {
  if (!isHostedMode) return Promise.reject(new Error('托管参数无效，请从站内在线使用页面进入。'))
  if (hostedConfig) return Promise.resolve(hostedConfig)

  window.addEventListener('message', handleHostedMessage)
  window.parent.postMessage({
    type: HOSTED_READY_MESSAGE,
    version: 1,
    sessionId: requestedSessionId,
    userId: hostedUserId,
  }, window.location.origin)

  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      unsubscribe()
      reject(new Error('工作台连接超时，请刷新页面重试。'))
    }, timeoutMs)
    const unsubscribe = subscribeHostedConfig((config) => {
      if (!config) return
      window.clearTimeout(timer)
      unsubscribe()
      resolve(config)
    })
  })
}

export function subscribeHostedConfig(listener: (config: HostedPlaygroundConfig | null) => void) {
  hostedConfigListeners.add(listener)
  return () => {
    hostedConfigListeners.delete(listener)
  }
}

export function clearHostedSensitiveUrlParams() {
  const url = new URL(window.location.href)
  const hadSensitiveParams = HOSTED_SENSITIVE_QUERY_KEYS.some((key) => url.searchParams.has(key))
  if (!hadSensitiveParams) return

  for (const key of HOSTED_SENSITIVE_QUERY_KEYS) url.searchParams.delete(key)
  window.history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`)
}

export async function cleanupHostedServiceWorkers() {
  if (!('serviceWorker' in navigator)) return false

  const registrations = await navigator.serviceWorker.getRegistrations()
  const hostedRegistrations = registrations.filter((registration) => {
    try {
      return new URL(registration.scope).pathname.startsWith(HOSTED_SERVICE_WORKER_PATH)
    } catch {
      return false
    }
  })
  await Promise.all(hostedRegistrations.map((registration) => registration.unregister()))

  const controller = navigator.serviceWorker.controller
  const controlledByHostedWorker = (() => {
    if (!controller) return false
    try {
      return new URL(controller.scriptURL).pathname.startsWith(HOSTED_SERVICE_WORKER_PATH)
    } catch {
      return false
    }
  })()
  if (!controlledByHostedWorker) {
    sessionStorage.removeItem('ipcai-playground:hosted-sw-cleanup')
    return false
  }

  const marker = isHostedMode ? `${hostedUserId}:${requestedSessionId}` : 'direct'
  if (sessionStorage.getItem('ipcai-playground:hosted-sw-cleanup') === marker) return false
  sessionStorage.setItem('ipcai-playground:hosted-sw-cleanup', marker)
  window.location.reload()
  return true
}

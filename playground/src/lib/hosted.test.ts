import { afterEach, describe, expect, it, vi } from 'vitest'
import { strFromU8, unzipSync } from 'fflate'
import { buildExportZip } from './exportZip'
import type { AppSettings } from '../types'

interface HostedTestContext {
  parent: { postMessage: ReturnType<typeof vi.fn> }
  messageListeners: Array<(event: MessageEvent) => void>
  replaceState: ReturnType<typeof vi.fn>
  module: typeof import('./hosted')
}

async function loadHostedModule(
  extraSearch = '',
  baseSearch = 'hosted=1&user=42&session=1234567890abcdef',
): Promise<HostedTestContext> {
  vi.resetModules()
  const parent = { postMessage: vi.fn() }
  const messageListeners: Array<(event: MessageEvent) => void> = []
  const replaceState = vi.fn()
  const location = new URL(`https://site.example/playground-app/?${baseSearch}${extraSearch}`)
  const storage = new Map<string, string>()

  vi.stubGlobal('window', {
    parent,
    location,
    history: { replaceState },
    addEventListener: (type: string, listener: (event: MessageEvent) => void) => {
      if (type === 'message') messageListeners.push(listener)
    },
    setTimeout,
    clearTimeout,
  })
  vi.stubGlobal('document', {
    documentElement: { classList: { toggle: vi.fn() } },
  })
  vi.stubGlobal('sessionStorage', {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
    removeItem: (key: string) => storage.delete(key),
  })
  vi.stubGlobal('navigator', {})

  return {
    parent,
    messageListeners,
    replaceState,
    module: await import('./hosted'),
  }
}

function settingsWithSecret(secret: string): AppSettings {
  return {
    baseUrl: 'https://site.example/v1',
    apiKey: secret,
    model: 'image-model',
    timeout: 600,
    apiMode: 'images',
    codexCli: false,
    apiProxy: false,
    customProviders: [],
    clearInputAfterSubmit: false,
    persistInputOnRestart: true,
    reuseTaskApiProfileTemporarily: false,
    alwaysShowRetryButton: false,
    allowPromptRewrite: false,
    taskCompletionNotification: false,
    enterSubmit: false,
    zipDownloadRoutes: [],
    agentScrollToBottomAfterSubmit: true,
    agentMaxToolRounds: 6,
    agentWebSearch: false,
    agentMathFormattingPrompt: true,
    agentApiConfigMode: 'hybrid',
    agentTextProfileId: 'text',
    agentImageProfileId: 'image',
    profiles: [
      {
        id: 'image',
        name: '站内生图',
        provider: 'openai',
        baseUrl: 'https://site.example/v1',
        apiKey: secret,
        model: 'image-model',
        timeout: 600,
        apiMode: 'images',
        codexCli: false,
        apiProxy: false,
      },
      {
        id: 'text',
        name: '站内对话',
        provider: 'openai',
        baseUrl: 'https://site.example/v1',
        apiKey: secret,
        model: 'text-model',
        timeout: 600,
        apiMode: 'responses',
        codexCli: false,
        apiProxy: false,
      },
    ],
    activeProfileId: 'image',
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.resetModules()
})

describe('hosted playground bridge', () => {
  it('uses a versioned per-user namespace and removes every persisted API key', async () => {
    const { module } = await loadHostedModule()
    const secret = 'sk-hosted-sentinel-secret'
    const scrubbed = module.scrubHostedSettings(settingsWithSecret(secret))

    expect(module.isHostedMode).toBe(true)
    expect(module.getPlaygroundStorageName()).toBe('ipcai-playground:v1:user:42')
    expect(JSON.stringify(scrubbed)).not.toContain(secret)
    expect(scrubbed.apiKey).toBe('')
    expect(scrubbed.profiles.every((profile) => profile.apiKey === '')).toBe(true)
  })

  it('ignores sensitive URL configuration and removes it from browser history', async () => {
    const { module, replaceState } = await loadHostedModule('&apiKey=sk-url-secret&apiUrl=https%3A%2F%2Fevil.example%2Fv1&model=wrong')

    module.clearHostedSensitiveUrlParams()

    expect(replaceState).toHaveBeenCalledOnce()
    const nextUrl = replaceState.mock.calls[0][2] as string
    expect(nextUrl).toContain('hosted=1')
    expect(nextUrl).toContain('user=42')
    expect(nextUrl).not.toContain('apiKey')
    expect(nextUrl).not.toContain('apiUrl')
    expect(nextUrl).not.toContain('model=')
    expect(nextUrl).not.toContain('sk-url-secret')
  })

  it('removes sensitive URL configuration before validating hosted parameters', async () => {
    const { module, replaceState } = await loadHostedModule(
      '&apiKey=sk-invalid-url-secret&settings=encoded-secret',
      'hosted=1&user=invalid&session=short',
    )

    expect(module.isHostedModeRequested).toBe(true)
    expect(module.isHostedMode).toBe(false)
    module.clearHostedSensitiveUrlParams()

    expect(replaceState).toHaveBeenCalledOnce()
    const nextUrl = replaceState.mock.calls[0][2] as string
    expect(nextUrl).toContain('hosted=1')
    expect(nextUrl).toContain('user=invalid')
    expect(nextUrl).not.toContain('apiKey')
    expect(nextUrl).not.toContain('settings')
    expect(nextUrl).not.toContain('secret')
  })

  it('locks hosted API configuration while allowing harmless local preferences', async () => {
    const { module } = await loadHostedModule()
    const current = settingsWithSecret('sk-host-runtime')
    const candidate = {
      ...settingsWithSecret('sk-imported-evil'),
      enterSubmit: true,
      allowPromptRewrite: true,
      agentWebSearch: true,
      zipDownloadRoutes: ['task-selection' as const],
    }

    const protectedSettings = module.protectHostedRuntimeSettings(current, candidate)

    expect(protectedSettings.enterSubmit).toBe(true)
    expect(protectedSettings.apiKey).toBe('sk-host-runtime')
    expect(protectedSettings.profiles.every((profile) => profile.apiKey === 'sk-host-runtime')).toBe(true)
    expect(protectedSettings.allowPromptRewrite).toBe(false)
    expect(protectedSettings.agentWebSearch).toBe(false)
    expect(protectedSettings.zipDownloadRoutes).toEqual([])
    expect(JSON.stringify(protectedSettings)).not.toContain('sk-imported-evil')
  })

  it('forces hosted backup transfers to exclude API configuration', async () => {
    const { module } = await loadHostedModule()

    expect(module.protectHostedExportOptions({ exportConfig: true, exportTasks: true })).toEqual({
      exportConfig: false,
      exportTasks: true,
    })
    expect(module.protectHostedImportOptions({ importConfig: true, importTasks: true })).toEqual({
      importConfig: false,
      importTasks: true,
    })
  })

  it('omits the hosted API key and settings object from the generated ZIP manifest', async () => {
    const { module } = await loadHostedModule()
    const secret = 'sk-export-zip-sentinel'
    const result = await buildExportZip({
      options: module.protectHostedExportOptions({ exportConfig: true, exportTasks: true }),
      exportedAt: Date.UTC(2020, 0, 1),
      settings: settingsWithSecret(secret),
      tasks: [],
      images: [],
      thumbnailsByImageId: new Map(),
      favoriteCollections: [],
      defaultFavoriteCollectionId: null,
      agentConversations: [],
    })
    const manifest = strFromU8(unzipSync(result.bytes)['manifest.json'])

    expect(JSON.parse(manifest)).not.toHaveProperty('settings')
    expect(manifest).not.toContain(secret)
  })

  it('accepts only same-origin parent messages and supports in-session key and model changes', async () => {
    const { module, parent, messageListeners } = await loadHostedModule()
    const configPromise = module.waitForHostedConfig()
    expect(parent.postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: module.HOSTED_READY_MESSAGE,
        sessionId: '1234567890abcdef',
        userId: '42',
      }),
      'https://site.example',
    )

    const message = (patch: Record<string, unknown> = {}, origin = 'https://site.example') => ({
      source: parent,
      origin,
      data: {
        type: module.HOSTED_CONFIG_MESSAGE,
        version: 1,
        sessionId: '1234567890abcdef',
        userId: '42',
        keyId: 7,
        apiKey: 'sk-first',
        baseUrl: 'https://site.example/v1',
        textModel: 'text-a',
        imageModel: 'image-a',
        theme: 'light',
        ...patch,
      },
    }) as unknown as MessageEvent

    messageListeners[0](message({ apiKey: 'sk-evil' }, 'https://evil.example'))
    expect(module.getHostedConfig()).toBeNull()

    messageListeners[0](message())
    await expect(configPromise).resolves.toMatchObject({ keyId: 7, textModel: 'text-a', imageModel: 'image-a' })

    const updates: Array<ReturnType<typeof module.getHostedConfig>> = []
    const unsubscribe = module.subscribeHostedConfig((config) => updates.push(config))
    messageListeners[0](message({ keyId: 8, apiKey: 'sk-second', textModel: 'text-b', imageModel: 'image-b' }))
    unsubscribe()

    expect(module.getHostedConfig()).toMatchObject({ keyId: 8, apiKey: 'sk-second', textModel: 'text-b', imageModel: 'image-b' })
    expect(updates).toHaveLength(1)
  })

  it('unregisters only service workers scoped to the hosted application', async () => {
    const { module } = await loadHostedModule()
    const unregisterRoot = vi.fn(async () => true)
    const unregisterHosted = vi.fn(async () => true)
    vi.stubGlobal('navigator', {
      serviceWorker: {
        controller: null,
        getRegistrations: vi.fn(async () => [
          { scope: 'https://site.example/', unregister: unregisterRoot },
          { scope: 'https://site.example/playground-app/', unregister: unregisterHosted },
        ]),
      },
    })

    await expect(module.cleanupHostedServiceWorkers()).resolves.toBe(false)
    expect(unregisterHosted).toHaveBeenCalledOnce()
    expect(unregisterRoot).not.toHaveBeenCalled()
  })

  it('removes a stale playground service worker even when hosted parameters are invalid', async () => {
    const { module } = await loadHostedModule('', 'hosted=1&user=invalid&session=short')
    const unregisterRoot = vi.fn(async () => true)
    const unregisterHosted = vi.fn(async () => true)
    vi.stubGlobal('navigator', {
      serviceWorker: {
        controller: null,
        getRegistrations: vi.fn(async () => [
          { scope: 'https://site.example/', unregister: unregisterRoot },
          { scope: 'https://site.example/playground-app/', unregister: unregisterHosted },
        ]),
      },
    })

    expect(module.isHostedMode).toBe(false)
    await expect(module.cleanupHostedServiceWorkers()).resolves.toBe(false)
    expect(unregisterHosted).toHaveBeenCalledOnce()
    expect(unregisterRoot).not.toHaveBeenCalled()
  })
})

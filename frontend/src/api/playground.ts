import { buildGatewayUrl } from './client'
import keysAPI from './keys'
import type { ApiKey } from '@/types'

export const PLAYGROUND_CONFIG_MESSAGE = 'ipcai:playground-config'
export const PLAYGROUND_CLEAR_MESSAGE = 'ipcai:playground-clear'
export const PLAYGROUND_READY_MESSAGE = 'ipcai:playground-ready'

const KEY_PAGE_SIZE = 100
const MAX_KEY_PAGES = 1000

export interface PlaygroundModelOption {
  id: string
}

export interface PlaygroundModels {
  all: PlaygroundModelOption[]
  text: PlaygroundModelOption[]
  image: PlaygroundModelOption[]
}

export interface PlaygroundHostedConfig {
  type: typeof PLAYGROUND_CONFIG_MESSAGE
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

export interface PlaygroundReadyExpectation {
  source: MessageEventSource | null
  origin: string
  sessionId: string
  userId: string
}

export class PlaygroundModelRequestError extends Error {
  readonly status: number

  constructor(status: number) {
    super(status > 0 ? `Model request failed with HTTP ${status}` : 'Model response is invalid')
    this.name = 'PlaygroundModelRequestError'
    this.status = status
  }
}

function normalizeModelIds(payload: unknown): PlaygroundModelOption[] {
  if (!payload || typeof payload !== 'object') return []

  const outer = payload as { data?: unknown }
  const candidate = Array.isArray(outer.data)
    ? outer.data
    : outer.data && typeof outer.data === 'object' && Array.isArray((outer.data as { data?: unknown }).data)
      ? (outer.data as { data: unknown[] }).data
      : []

  const seen = new Set<string>()
  const models: PlaygroundModelOption[] = []
  for (const item of candidate) {
    const id = typeof item === 'string'
      ? item.trim()
      : item && typeof item === 'object' && typeof (item as { id?: unknown }).id === 'string'
        ? (item as { id: string }).id.trim()
        : ''
    if (!id || id.length > 256 || seen.has(id)) continue
    seen.add(id)
    models.push({ id })
  }
  return models
}

function isImageModel(id: string): boolean {
  const normalized = id.toLowerCase()
  return [
    'image',
    'dall-e',
    'dalle',
    'imagen',
    'flux',
    'ideogram',
    'stable-diffusion',
    'sdxl',
  ].some((marker) => normalized.includes(marker))
}

function isTextModel(id: string): boolean {
  if (isImageModel(id)) return false
  const normalized = id.toLowerCase()
  return ![
    'embedding',
    'rerank',
    'moderation',
    'whisper',
    'transcription',
    'text-to-speech',
    'speech',
    'tts',
    'audio',
  ].some((marker) => normalized.includes(marker))
}

export function categorizePlaygroundModels(payload: unknown): PlaygroundModels {
  const all = normalizeModelIds(payload)
  const textMatches = all.filter((model) => isTextModel(model.id))
  const imageMatches = all.filter((model) => isImageModel(model.id))

  const prioritize = (preferred: PlaygroundModelOption[]) => {
    const preferredIds = new Set(preferred.map((model) => model.id))
    return [...preferred, ...all.filter((model) => !preferredIds.has(model.id))]
  }

  // Naming is only a default-ordering hint. Keep every gateway alias selectable:
  // custom model names often do not reveal whether they support text or images.
  return {
    all,
    text: prioritize(textMatches),
    image: prioritize(imageMatches),
  }
}

export async function listActivePlaygroundKeys(signal?: AbortSignal): Promise<ApiKey[]> {
  const keys: ApiKey[] = []
  const seen = new Set<number>()
  let page = 1
  let totalPages = 1

  do {
    const response = await keysAPI.list(
      page,
      KEY_PAGE_SIZE,
      { status: 'active', sort_by: 'created_at', sort_order: 'desc' },
      { signal },
    )
    for (const key of response.items ?? []) {
      if (key.status !== 'active' || seen.has(key.id)) continue
      seen.add(key.id)
      keys.push(key)
    }

    const reportedPages = Number(response.pages)
    totalPages = Number.isSafeInteger(reportedPages)
      ? Math.min(Math.max(reportedPages, 1), MAX_KEY_PAGES)
      : page
    page += 1
  } while (page <= totalPages)

  return keys
}

export async function fetchPlaygroundModels(
  apiKey: string,
  signal?: AbortSignal,
): Promise<PlaygroundModels> {
  const modelsUrl = typeof window === 'undefined'
    ? buildGatewayUrl('/v1/models')
    : new URL('/v1/models', window.location.origin).toString()
  const response = await fetch(modelsUrl, {
    method: 'GET',
    headers: {
      Accept: 'application/json',
      Authorization: `Bearer ${apiKey}`,
    },
    credentials: 'same-origin',
    cache: 'no-store',
    referrerPolicy: 'same-origin',
    signal,
  })

  if (!response.ok) {
    // Do not surface the gateway response body. It can contain upstream details.
    throw new PlaygroundModelRequestError(response.status)
  }

  let payload: unknown
  try {
    payload = await response.json()
  } catch {
    throw new PlaygroundModelRequestError(0)
  }

  const models = categorizePlaygroundModels(payload)
  if (models.all.length === 0) throw new PlaygroundModelRequestError(0)
  return models
}

export function isHostedPlaygroundReadyEvent(
  event: MessageEvent,
  expected: PlaygroundReadyExpectation,
): boolean {
  if (event.source !== expected.source || event.origin !== expected.origin) return false
  if (!event.data || typeof event.data !== 'object') return false

  const data = event.data as {
    type?: unknown
    version?: unknown
    sessionId?: unknown
    userId?: unknown
  }
  return data.type === PLAYGROUND_READY_MESSAGE
    && data.version === 1
    && data.sessionId === expected.sessionId
    && data.userId === expected.userId
}

export const playgroundAPI = {
  listActiveKeys: listActivePlaygroundKeys,
  fetchModels: fetchPlaygroundModels,
}

export default playgroundAPI

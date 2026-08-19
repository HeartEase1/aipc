import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listKeys } = vi.hoisted(() => ({
  listKeys: vi.fn(),
}))

vi.mock('../keys', () => ({
  default: { list: listKeys },
}))

vi.mock('../client', () => ({
  buildGatewayUrl: (path: string) => path,
}))

import {
  PLAYGROUND_READY_MESSAGE,
  PlaygroundModelRequestError,
  categorizePlaygroundModels,
  fetchPlaygroundModels,
  isHostedPlaygroundReadyEvent,
  listActivePlaygroundKeys,
} from '@/api/playground'

describe('playground API', () => {
  beforeEach(() => {
    listKeys.mockReset()
    vi.restoreAllMocks()
  })

  it('loads every page of active keys and filters unexpected inactive rows', async () => {
    listKeys
      .mockResolvedValueOnce({
        items: [
          { id: 1, status: 'active' },
          { id: 2, status: 'inactive' },
        ],
        pages: 2,
      })
      .mockResolvedValueOnce({
        items: [
          { id: 1, status: 'active' },
          { id: 3, status: 'active' },
        ],
        pages: 2,
      })

    const result = await listActivePlaygroundKeys()

    expect(listKeys).toHaveBeenCalledTimes(2)
    expect(listKeys).toHaveBeenNthCalledWith(
      1,
      1,
      100,
      { status: 'active', sort_by: 'created_at', sort_order: 'desc' },
      { signal: undefined },
    )
    expect(result.map((key) => key.id)).toEqual([1, 3])
  })

  it('classifies text and image models and falls back for opaque aliases', () => {
    const models = categorizePlaygroundModels({
      data: [
        { id: 'gpt-5' },
        { id: 'gpt-image-1' },
        { id: 'text-embedding-3-large' },
        { id: 'gpt-5' },
      ],
    })

    expect(models.all.map((model) => model.id)).toEqual([
      'gpt-5',
      'gpt-image-1',
      'text-embedding-3-large',
    ])
    expect(models.text.map((model) => model.id)).toEqual([
      'gpt-5',
      'gpt-image-1',
      'text-embedding-3-large',
    ])
    expect(models.image.map((model) => model.id)).toEqual([
      'gpt-image-1',
      'gpt-5',
      'text-embedding-3-large',
    ])

    const aliases = categorizePlaygroundModels({ data: [{ id: 'creative-v2' }] })
    expect(aliases.text).toEqual(aliases.all)
    expect(aliases.image).toEqual(aliases.all)
  })

  it('requests models from same-origin /v1 without reading an error body', async () => {
    const json = vi.fn()
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      json,
    }))

    await expect(fetchPlaygroundModels('sk-secret')).rejects.toEqual(
      expect.objectContaining<Partial<PlaygroundModelRequestError>>({ status: 403 }),
    )
    expect(fetch).toHaveBeenCalledWith(
      `${window.location.origin}/v1/models`,
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer sk-secret' }),
      }),
    )
    expect(json).not.toHaveBeenCalled()
  })

  it('accepts ready messages only from the expected frame, origin, user, and session', () => {
    const data = {
      type: PLAYGROUND_READY_MESSAGE,
      version: 1,
      sessionId: '12345678-1234-1234-1234-123456789abc',
      userId: '42',
    }
    const event = new MessageEvent('message', {
      source: window,
      origin: window.location.origin,
      data,
    })
    const expected = {
      source: window,
      origin: window.location.origin,
      sessionId: data.sessionId,
      userId: data.userId,
    }

    expect(isHostedPlaygroundReadyEvent(event, expected)).toBe(true)
    expect(isHostedPlaygroundReadyEvent(event, { ...expected, userId: '43' })).toBe(false)
    expect(isHostedPlaygroundReadyEvent(event, { ...expected, origin: 'https://evil.example' })).toBe(false)
    expect(isHostedPlaygroundReadyEvent(event, { ...expected, source: null })).toBe(false)
    expect(isHostedPlaygroundReadyEvent(
      new MessageEvent('message', { source: window, origin: window.location.origin, data: { ...data, version: 2 } }),
      expected,
    )).toBe(false)
  })
})

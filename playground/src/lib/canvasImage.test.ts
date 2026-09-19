import { describe, expect, it, vi } from 'vitest'

import { dataUrlToBlob } from './canvasImage'

describe('dataUrlToBlob', () => {
  it('decodes Base64 image data locally without a fetch request', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')

    const blob = await dataUrlToBlob('data:image/png;base64,aW1hZ2U=')

    expect(fetchSpy).not.toHaveBeenCalled()
    expect(blob.type).toBe('image/png')
    expect(new TextDecoder().decode(await blob.arrayBuffer())).toBe('image')
  })

  it('supports URL-encoded data and a fallback MIME type', async () => {
    const blob = await dataUrlToBlob('data:,%3Csvg%3E%3C%2Fsvg%3E', 'image/svg+xml')

    expect(blob.type).toBe('image/svg+xml')
    expect(new TextDecoder().decode(await blob.arrayBuffer())).toBe('<svg></svg>')
  })

  it('rejects malformed image data', async () => {
    await expect(dataUrlToBlob('not-a-data-url')).rejects.toThrow('图片数据格式无效')
    await expect(dataUrlToBlob('data:image/png;base64,%%%')).rejects.toThrow('图片数据格式无效')
  })
})

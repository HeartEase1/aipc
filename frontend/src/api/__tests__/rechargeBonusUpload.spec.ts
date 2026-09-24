import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '../client'
import { rechargeBonusAPI } from '../admin/balanceMarketing'

vi.mock('@/i18n', () => ({ getLocale: () => 'zh-CN' }))

describe('recharge bonus poster upload', () => {
  const originalAdapter = apiClient.defaults.adapter
  afterEach(() => { apiClient.defaults.adapter = originalAdapter })

  it('preserves a 2 MiB file through the real Axios request transformation', async () => {
    const file = new File([new Uint8Array(2 << 20)], 'poster.png', { type: 'image/png' })
    const adapter = vi.fn(async (config) => ({
      status: 200, statusText: 'OK', headers: {}, config,
      data: { code: 0, data: { id: 1, url: '/poster.png' } },
    }))
    apiClient.defaults.adapter = adapter

    const response = await rechargeBonusAPI.upload(file)

    const config = adapter.mock.calls[0][0]
    expect(config.data).toBeInstanceOf(FormData)
    expect(config.data.get('file')).toBe(file)
    expect(config.data.get('file').size).toBe(2 << 20)
    expect(String(config.headers.getContentType() || '')).not.toContain('application/json')
    expect(response.data.id).toBe(1)
  })
})

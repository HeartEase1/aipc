import { beforeEach, describe, expect, it, vi } from 'vitest'

import { list, markRead } from '../benefitGrants'

const { get, put } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }))

vi.mock('../client', () => ({ apiClient: { get, put } }))

describe('benefit grant API contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads account activity from the authenticated user route', async () => {
    const response = { items: [], total: 0, page: 1, page_size: 20, pages: 1 }
    get.mockResolvedValue({ data: response })

    await expect(list(1, 20)).resolves.toEqual(response)
    expect(get).toHaveBeenCalledWith('/user/benefit-grants', {
      params: { page: 1, page_size: 20, unread_only: undefined },
    })
  })

  it('marks an activity as read with the matching PUT route', async () => {
    put.mockResolvedValue({ data: { message: 'ok' } })

    await expect(markRead(42)).resolves.toEqual({ message: 'ok' })
    expect(put).toHaveBeenCalledWith('/user/benefit-grants/42/read')
  })
})

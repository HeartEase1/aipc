import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserErrorRequestsTable from '../UserErrorRequestsTable.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const mountTable = (fixedViewport = false) => mount(UserErrorRequestsTable, {
  props: {
    rows: [],
    total: 0,
    loading: false,
    page: 1,
    pageSize: 20,
    fixedViewport,
  },
  global: {
    stubs: {
      DataTable: { template: '<div class="data-table-stub"><slot name="empty" /></div>' },
      EmptyState: true,
      Pagination: true,
      UserErrorDetailModal: true,
      IpGeoCell: true,
      IpGeoBatchToolbar: true,
    },
  },
})

describe('UserErrorRequestsTable viewport', () => {
  it('uses a bounded table window only when requested', () => {
    expect(mountTable().get('.user-error-requests-table').classes())
      .not.toContain('user-error-requests-table--fixed-viewport')
    expect(mountTable(true).get('.user-error-requests-table').classes())
      .toContain('user-error-requests-table--fixed-viewport')
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import IQTestModal from '../IQTestModal.vue'
import { getAvailableModels } from '@/api/admin/accounts'
import { getPelicanCapabilities } from '@/api/admin/pelicanTest'
import { PELICAN_HISTORY_KEY } from '@/utils/pelicanTest'

vi.mock('@/api/admin/accounts', () => ({ getAvailableModels: vi.fn() }))
vi.mock('@/api/admin/pelicanTest', async importOriginal => ({ ...(await importOriginal<typeof import('@/api/admin/pelicanTest')>()), getPelicanCapabilities: vi.fn() }))
enableAutoUnmount(afterEach)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function streamResponse(events: Array<Record<string, unknown>>) {
  const encoder = new TextEncoder()
  const chunks = events.map((event) => encoder.encode(`data: ${JSON.stringify(event)}\n\n`))
  let index = 0
  return {
    ok: true,
    body: {
      getReader: () => ({
        read: vi.fn(async () => index < chunks.length
          ? { done: false, value: chunks[index++] }
          : { done: true, value: undefined }),
        cancel: vi.fn().mockResolvedValue(undefined),
        releaseLock: vi.fn()
      })
    }
  } as Response
}

function mountModal() {
  return mount(IQTestModal, {
    props: {
      show: true,
      account: {
        id: 42,
        name: 'Astra account',
        platform: 'openai',
        type: 'oauth',
        status: 'active'
      } as any
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Input: true,
        TextArea: true,
        Select: true,
        Icon: true
      }
    }
  })
}

describe('IQTestModal', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('auth_token', 'test-token')
    vi.mocked(getAvailableModels).mockResolvedValue([{ id: 'gpt-6-astra', display_name: 'Astra', type: 'model', created_at: '' }, { id: 'gpt-6-luna', display_name: 'Luna', type: 'model', created_at: '' }])
    vi.mocked(getPelicanCapabilities).mockResolvedValue({ model_id: 'gpt-6-astra', supported_reasoning_levels: ['low', 'medium', 'high', 'xhigh', 'max'] })
    global.fetch = vi.fn(() => Promise.resolve(streamResponse([
      { type: 'test_start', model: 'gpt-6-astra' },
      { type: 'content', text: '<!doctype html><html><head><title>Pelican</title></head><body><svg></svg>' },
      { type: 'content', text: '<script>document.body.dataset.animated="true"</script></body></html>' },
      { type: 'test_complete', success: true }
    ]))) as any
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('uses the dedicated endpoint and sends identical settings to parallel runs', async () => {
    const wrapper = mountModal()
    await flushPromises()
    ;(wrapper.vm as any).parallelCount = 2
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledTimes(2)
    for (const [url, request] of (global.fetch as any).mock.calls) {
      expect(url).toContain('/admin/accounts/42/pelican-test')
      const body = JSON.parse(request.body)
      expect(body).toMatchObject({
        model_id: 'gpt-6-astra',
        reasoning_effort: 'medium'
      })
      expect(body.prompt).toContain('admin.accounts.pelicanTest.defaultPrompt')
      expect(body.prompt).toContain('admin.accounts.pelicanTest.deliveryContract')
    }

    const frames = wrapper.findAll('iframe')
    expect(frames).toHaveLength(2)
    expect(frames[0].attributes('srcdoc')).toContain('Content-Security-Policy')
    expect(frames[0].attributes('srcdoc')).toContain('<svg></svg>')
    expect(wrapper.text()).toContain('admin.accounts.pelicanTest.success')
    expect(localStorage.getItem(`${PELICAN_HISTORY_KEY}:local`)).toContain('gpt-6-astra')
    expect(frames[0].attributes('sandbox')).toBe('allow-scripts')
  })

  it('keeps non-HTML output visible but marks it as failed', async () => {
    global.fetch = vi.fn(() => Promise.resolve(streamResponse([
      { type: 'content', text: 'I cannot provide HTML.' },
      { type: 'test_complete', success: true }
    ]))) as any
    const wrapper = mountModal()
    await flushPromises()
    await (wrapper.vm as any).startTest()
    await flushPromises()

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('I cannot provide HTML.')
    expect(wrapper.text()).toContain('admin.accounts.pelicanTest.failed')
  })

  it('sends a selected model and max effort, and removes unsupported effort after changing models', async () => {
    const wrapper = mountModal()
    await flushPromises()
    const vm = wrapper.vm as any
    vm.modelId = 'gpt-6-luna'
    await flushPromises()
    vm.reasoningEffort = 'max'
    await vm.startTest()
    expect(JSON.parse(vi.mocked(fetch).mock.calls[0][1]!.body as string)).toMatchObject({ model_id: 'gpt-6-luna', reasoning_effort: 'max' })
    vi.mocked(getPelicanCapabilities).mockResolvedValue({ model_id: 'custom-chat', supported_reasoning_levels: [] })
    vm.modelId = 'custom-chat'
    await flushPromises()
    expect(vm.reasoningEffort).toBe('')
  })

  it('ignores stale model discovery after switching accounts', async () => {
    let resolveModels!: (models: any[]) => void
    vi.mocked(getAvailableModels).mockImplementationOnce(() => new Promise(resolve => { resolveModels = resolve }))
    const wrapper = mountModal()
    await wrapper.setProps({ account: { id: 55, platform: 'openai', type: 'apikey' } as any })
    await flushPromises()
    resolveModels([{ id: 'stale-model', display_name: 'Stale' }])
    await flushPromises()
    expect((wrapper.vm as any).modelId).toBe('gpt-6-astra')
  })

  it('aborts on close and never saves cancelled output into another account history', async () => {
    const wrapper = mountModal()
    await flushPromises()
    let signal!: AbortSignal
    vi.stubGlobal('fetch', vi.fn((_url, init) => new Promise((_resolve, reject) => {
      signal = init.signal
      signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
    })))
    const vm = wrapper.vm as any
    const completion = vm.startTest()
    expect(vm.running).toBe(true)
    vm.handleClose()
    await completion
    expect(signal.aborted).toBe(true)
    expect(localStorage.getItem(`${PELICAN_HISTORY_KEY}:local`)).toBeNull()
    expect(wrapper.emitted('close')).toHaveLength(1)
    vi.unstubAllGlobals()
  })

  it('requires a completion event even when the partial output looks like HTML', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => streamResponse([{ type: 'content', text: '<html><body>partial</body></html>' }])))
    const wrapper = mountModal()
    await flushPromises()
    await (wrapper.vm as any).startTest()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain('incompleteResponse')
    vi.unstubAllGlobals()
  })
})

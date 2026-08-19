import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'

import OnlinePlaygroundView from '../OnlinePlaygroundView.vue'
import Select from '@/components/common/Select.vue'

const { listActiveKeys, fetchModels, authState } = vi.hoisted(() => ({
  listActiveKeys: vi.fn(),
  fetchModels: vi.fn(),
  authState: {
    user: { id: 42, role: 'user' } as { id: number; role: string } | null,
  },
}))

const authStore = reactive(authState)
const originalRequestFullscreen = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'requestFullscreen')
const originalExitFullscreen = Object.getOwnPropertyDescriptor(document, 'exitFullscreen')
const originalFullscreenElement = Object.getOwnPropertyDescriptor(document, 'fullscreenElement')

vi.mock('@/api/playground', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/playground')>()
  return {
    ...actual,
    playgroundAPI: {
      listActiveKeys,
      fetchModels,
    },
  }
})

vi.mock('@/stores/auth', async () => {
  const { reactive } = await import('vue')
  const store = reactive(authState)
  return { useAuthStore: () => store }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const activeKey = {
  id: 7,
  key: 'sk-page-secret',
  name: 'Playground key',
  status: 'active',
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function mountView() {
  return mount(OnlinePlaygroundView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

function restoreProperty(target: object, key: PropertyKey, descriptor?: PropertyDescriptor) {
  if (descriptor) {
    Object.defineProperty(target, key, descriptor)
    return
  }
  Reflect.deleteProperty(target, key)
}

describe('OnlinePlaygroundView', () => {
  beforeEach(() => {
    vi.useRealTimers()
    authStore.user = { id: 42, role: 'user' }
    listActiveKeys.mockReset()
    fetchModels.mockReset()
    listActiveKeys.mockResolvedValue([activeKey])
    fetchModels.mockResolvedValue({
      all: [{ id: 'gpt-5' }, { id: 'gpt-5-mini' }, { id: 'gpt-image-1' }],
      text: [{ id: 'gpt-5' }, { id: 'gpt-5-mini' }],
      image: [{ id: 'gpt-image-1' }],
    })
  })

  afterEach(() => {
    restoreProperty(HTMLElement.prototype, 'requestFullscreen', originalRequestFullscreen)
    restoreProperty(document, 'exitFullscreen', originalExitFullscreen)
    restoreProperty(document, 'fullscreenElement', originalFullscreenElement)
  })

  it('keeps the API key out of the iframe URL and sends it only after a valid ready message', async () => {
    const wrapper = mountView()
    await flushPromises()

    const iframe = wrapper.get('iframe').element as HTMLIFrameElement
    const iframeURL = new URL(iframe.src)
    expect(iframeURL.pathname).toBe('/playground-app/')
    expect(iframeURL.searchParams.get('hosted')).toBe('1')
    expect(iframeURL.searchParams.get('user')).toBe('42')
    expect(iframeURL.href).not.toContain(activeKey.key)
    expect(wrapper.html()).not.toContain(activeKey.key)
    expect(iframe.getAttribute('allow')).toBe('clipboard-write')
    expect(iframe.getAttribute('referrerpolicy')).toBe('no-referrer')

    const frameWindow = { postMessage: vi.fn() } as unknown as Window
    Object.defineProperty(iframe, 'contentWindow', { configurable: true, value: frameWindow })
    const postMessage = vi.spyOn(frameWindow, 'postMessage')
    const readyData = {
      type: 'ipcai:playground-ready',
      version: 1,
      sessionId: iframeURL.searchParams.get('session'),
      userId: '42',
    }

    window.dispatchEvent(new MessageEvent('message', {
      source: frameWindow,
      origin: 'https://evil.example',
      data: readyData,
    }))
    expect(postMessage).not.toHaveBeenCalled()

    window.dispatchEvent(new MessageEvent('message', {
      source: frameWindow,
      origin: window.location.origin,
      data: readyData,
    }))

    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'ipcai:playground-config',
        version: 1,
        userId: '42',
        keyId: 7,
        apiKey: activeKey.key,
        baseUrl: `${window.location.origin}/v1`,
        textModel: 'gpt-5',
        imageModel: 'gpt-image-1',
      }),
      window.location.origin,
    )

    postMessage.mockClear()
    wrapper.findAllComponents(Select)[1].vm.$emit('update:modelValue', 'gpt-5-mini')
    await flushPromises()
    expect(postMessage).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'ipcai:playground-config',
        apiKey: activeKey.key,
        textModel: 'gpt-5-mini',
      }),
      window.location.origin,
    )

    wrapper.unmount()
  })

  it('loads after authentication is restored and clears the active session on logout', async () => {
    authStore.user = null
    const wrapper = mountView()
    await flushPromises()

    expect(listActiveKeys).not.toHaveBeenCalled()
    expect(wrapper.find('iframe').exists()).toBe(false)

    authStore.user = { id: 84, role: 'user' }
    await flushPromises()

    expect(listActiveKeys).toHaveBeenCalledOnce()
    const iframe = wrapper.get('iframe').element as HTMLIFrameElement
    expect(new URL(iframe.src).searchParams.get('user')).toBe('84')

    const frameWindow = { postMessage: vi.fn() } as unknown as Window
    Object.defineProperty(iframe, 'contentWindow', { configurable: true, value: frameWindow })
    authStore.user = null
    await flushPromises()

    expect(frameWindow.postMessage).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'ipcai:playground-clear' }),
      window.location.origin,
    )
    expect(wrapper.find('iframe').exists()).toBe(false)

    wrapper.unmount()
  })

  it('ignores stale model responses during rapid key changes', async () => {
    const secondKey = { ...activeKey, id: 8, key: 'sk-second', name: 'Second key' }
    const thirdKey = { ...activeKey, id: 9, key: 'sk-third', name: 'Third key' }
    listActiveKeys.mockResolvedValueOnce([activeKey, secondKey, thirdKey])
    const secondModels = deferred<{
      all: Array<{ id: string }>
      text: Array<{ id: string }>
      image: Array<{ id: string }>
    }>()
    const thirdModels = deferred<{
      all: Array<{ id: string }>
      text: Array<{ id: string }>
      image: Array<{ id: string }>
    }>()
    fetchModels
      .mockResolvedValueOnce({
        all: [{ id: 'text-first' }, { id: 'image-first' }],
        text: [{ id: 'text-first' }],
        image: [{ id: 'image-first' }],
      })
      .mockReturnValueOnce(secondModels.promise)
      .mockReturnValueOnce(thirdModels.promise)

    const wrapper = mountView()
    await flushPromises()
    const keySelect = wrapper.findAllComponents(Select)[0]

    keySelect.vm.$emit('update:modelValue', 8)
    await nextTick()
    keySelect.vm.$emit('update:modelValue', 9)
    await nextTick()

    thirdModels.resolve({
      all: [{ id: 'text-third' }, { id: 'image-third' }],
      text: [{ id: 'text-third' }],
      image: [{ id: 'image-third' }],
    })
    await flushPromises()
    secondModels.resolve({
      all: [{ id: 'text-stale' }, { id: 'image-stale' }],
      text: [{ id: 'text-stale' }],
      image: [{ id: 'image-stale' }],
    })
    await flushPromises()

    const selects = wrapper.findAllComponents(Select)
    expect(selects[1].props('modelValue')).toBe('text-third')
    expect(selects[2].props('modelValue')).toBe('image-third')
    expect(fetchModels).toHaveBeenNthCalledWith(2, 'sk-second', expect.any(AbortSignal))
    expect(fetchModels).toHaveBeenNthCalledWith(3, 'sk-third', expect.any(AbortSignal))

    wrapper.unmount()
  })

  it('shows the connection timeout and creates a fresh session when reconnecting', async () => {
    vi.useFakeTimers()
    const wrapper = mountView()
    await flushPromises()

    const firstIframe = wrapper.get('iframe')
    const firstSession = new URL((firstIframe.element as HTMLIFrameElement).src).searchParams.get('session')
    await firstIframe.trigger('load')
    vi.advanceTimersByTime(12_000)
    await nextTick()

    expect(wrapper.text()).toContain('onlinePlayground.connectionFailed')
    await wrapper.get('button.btn-primary').trigger('click')
    await nextTick()

    const nextSession = new URL((wrapper.get('iframe').element as HTMLIFrameElement).src).searchParams.get('session')
    expect(nextSession).not.toBe(firstSession)
    expect(wrapper.text()).not.toContain('onlinePlayground.connectionFailedHint')

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('shows a link to key management when the user has no active key', async () => {
    listActiveKeys.mockResolvedValueOnce([])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('onlinePlayground.noActiveKey')
    expect(wrapper.get('a[href="/keys"]').exists()).toBe(true)
    expect(wrapper.find('iframe').exists()).toBe(false)

    wrapper.unmount()
  })

  it('enters and exits native fullscreen without remounting the workspace', async () => {
    let fullscreenElement: Element | null = null
    const setFullscreenElement = (element: Element | null) => {
      fullscreenElement = element
    }
    const requestFullscreen = vi.fn(async function (this: HTMLElement) {
      setFullscreenElement(this)
      document.dispatchEvent(new Event('fullscreenchange'))
    })
    const exitFullscreen = vi.fn(async () => {
      setFullscreenElement(null)
      document.dispatchEvent(new Event('fullscreenchange'))
    })
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
      configurable: true,
      value: requestFullscreen,
    })
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: exitFullscreen,
    })
    Object.defineProperty(document, 'fullscreenElement', {
      configurable: true,
      get: () => fullscreenElement,
    })

    const wrapper = mountView()
    await flushPromises()

    const iframe = wrapper.get('iframe').element as HTMLIFrameElement
    const iframeURL = new URL(iframe.src)
    const frameWindow = { postMessage: vi.fn() } as unknown as Window
    Object.defineProperty(iframe, 'contentWindow', { configurable: true, value: frameWindow })
    window.dispatchEvent(new MessageEvent('message', {
      source: frameWindow,
      origin: window.location.origin,
      data: {
        type: 'ipcai:playground-ready',
        version: 1,
        sessionId: iframeURL.searchParams.get('session'),
        userId: '42',
      },
    }))
    await nextTick()

    const workspace = wrapper.get('[aria-labelledby="playground-workspace-title"]').element
    await wrapper.get('button[aria-label="onlinePlayground.enterFullscreen"]').trigger('click')
    await flushPromises()

    expect(requestFullscreen).toHaveBeenCalledOnce()
    expect(fullscreenElement).toBe(workspace)
    expect(wrapper.get('button[aria-label="onlinePlayground.exitFullscreen"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('iframe').element).toBe(iframe)

    await wrapper.get('button[aria-label="onlinePlayground.exitFullscreen"]').trigger('click')
    await flushPromises()

    expect(exitFullscreen).toHaveBeenCalledOnce()
    expect(fullscreenElement).toBeNull()
    expect(wrapper.get('button[aria-label="onlinePlayground.enterFullscreen"]').attributes('aria-pressed')).toBe('false')
    expect(wrapper.get('iframe').element).toBe(iframe)

    await wrapper.get('button[aria-label="onlinePlayground.enterFullscreen"]').trigger('click')
    await flushPromises()
    setFullscreenElement(null)
    document.dispatchEvent(new Event('fullscreenchange'))
    await nextTick()

    expect(wrapper.get('button[aria-label="onlinePlayground.enterFullscreen"]').exists()).toBe(true)
    expect(wrapper.get('iframe').element).toBe(iframe)

    wrapper.unmount()
  })

  it('hides the fullscreen control when the browser does not support it', async () => {
    Object.defineProperty(HTMLElement.prototype, 'requestFullscreen', {
      configurable: true,
      value: undefined,
    })
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: undefined,
    })

    const wrapper = mountView()
    await flushPromises()

    const iframe = wrapper.get('iframe').element as HTMLIFrameElement
    const iframeURL = new URL(iframe.src)
    const frameWindow = { postMessage: vi.fn() } as unknown as Window
    Object.defineProperty(iframe, 'contentWindow', { configurable: true, value: frameWindow })
    window.dispatchEvent(new MessageEvent('message', {
      source: frameWindow,
      origin: window.location.origin,
      data: {
        type: 'ipcai:playground-ready',
        version: 1,
        sessionId: iframeURL.searchParams.get('session'),
        userId: '42',
      },
    }))
    await nextTick()

    expect(wrapper.find('button[aria-label="onlinePlayground.enterFullscreen"]').exists()).toBe(false)

    wrapper.unmount()
  })
})

// Generated HTML runs only inside an opaque sandbox. Always rebuild this wrapper
// from raw output, including when restoring history; stored HTML is not trusted.
export const PELICAN_MAX_OUTPUT = 512 * 1024
export const PELICAN_HISTORY_KEY = 'aipc-pelican-history-v1'
export const PELICAN_HISTORY_LIMIT = 2 * 1024 * 1024

export function extractPelicanHtml(raw: string): string {
  let html = raw.trim()
  const fenced = html.match(/```(?:html|xml|svg)?\s*([\s\S]*?)```/i)
  if (fenced?.[1]) html = fenced[1].trim()
  const match = /<!doctype\s+html\b|<html\b|<svg\b/i.exec(html)
  if (!match) return ''
  html = html.slice(match.index)
  const end = /<\/html\s*>/i.exec(html) || /<\/svg\s*>/i.exec(html)
  if (!end) return ''
  html = html.slice(0, end.index + end[0].length)
  // Place the policy before every byte of model output, even malformed heads.
  const policy = "default-src 'none'; img-src data:; media-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:; connect-src 'none'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'"
  return `<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="${policy}"><meta charset="utf-8"><meta name="referrer" content="no-referrer"></head><body>${html}</body></html>`
}

export function isPelicanTextModel(id: string): boolean {
  return !/(?:image|imagine|video|embedding|rerank|tts|transcri|whisper|realtime|audio|voice)/i.test(id)
}

export async function consumePelicanStream(
  response: Response,
  onText: (text: string) => void,
  errorMessage: string
): Promise<void> {
  if (!response.ok) {
    const data = await response.json().catch(() => null)
    throw new Error(data?.message || `HTTP ${response.status}`)
  }
  const reader = response.body?.getReader()
  if (!reader) throw new Error(errorMessage)
  const decoder = new TextDecoder()
  let buffer = ''
  let complete = false
  let size = 0
  const line = (value: string) => {
    if (!value.startsWith('data:')) return
    let event: { type?: string; text?: string; success?: boolean; error?: string }
    try { event = JSON.parse(value.slice(5).trim()) } catch { return }
    if (event.type === 'error' || (event.type === 'test_complete' && !event.success)) {
      throw new Error(event.error || errorMessage)
    }
    if (event.type === 'content' && typeof event.text === 'string') {
      size += event.text.length
      if (size > PELICAN_MAX_OUTPUT) throw new Error('Output exceeds 512 KiB')
      onText(event.text)
    }
    if (event.type === 'test_complete' && event.success) complete = true
  }
  try {
    while (!complete) {
      const { done, value } = await reader.read()
      buffer += done ? decoder.decode() : decoder.decode(value, { stream: true })
      if (buffer.length > PELICAN_MAX_OUTPUT * 2) throw new Error('Stream frame is too large')
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''
      for (const item of lines) { line(item.trim()); if (complete) break }
      if (done) { if (!complete && buffer.trim()) line(buffer.trim()); break }
    }
    if (!complete) throw new Error(errorMessage)
  } finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}

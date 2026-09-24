import { describe, expect, it } from 'vitest'
import { consumePelicanStream, extractPelicanHtml } from '../pelicanTest'

describe('Pelican generated output', () => {
  it('places CSP before model-controlled HTML including malformed heads', () => {
    const html = extractPelicanHtml('<html><img src="https://example.com/leak"><head><script>fetch("https://example.com")</script></head><body>ok</body></html>')
    expect(html.indexOf('Content-Security-Policy')).toBeLessThan(html.indexOf('<img'))
    expect(html).toContain("connect-src 'none'")
    expect(html).toContain("frame-src 'none'")
  })
  it('accepts fenced SVG and rejects incomplete or non-HTML output', () => {
    expect(extractPelicanHtml('```svg\n<svg><circle /></svg>\n```')).toContain('<svg>')
    expect(extractPelicanHtml('<html><body>unfinished')).toBe('')
    expect(extractPelicanHtml('plain output')).toBe('')
  })
  it('handles split UTF-8 and final SSE lines without a newline', async () => {
    const bytes = new TextEncoder().encode('data: {"type":"content","text":"鹈鹕"}\n\ndata: {"type":"test_complete","success":true}')
    const response = new Response(new ReadableStream({ start(controller) { for (const byte of bytes) controller.enqueue(new Uint8Array([byte])); controller.close() } }))
    let output = ''
    await consumePelicanStream(response, text => { output += text }, 'incomplete')
    expect(output).toBe('鹈鹕')
  })
})

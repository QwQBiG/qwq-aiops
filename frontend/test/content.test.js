import test from 'node:test'
import assert from 'node:assert/strict'
import { JSDOM } from 'jsdom'
import { createContentRenderer } from '../src/lib/content.js'

const window = new JSDOM('').window
const render = createContentRenderer(window)

test('retains useful Markdown structure', () => {
  const content = render('## Status\n\n**Healthy**\n\n```sh\necho hello\n```\n\n[guide](https://example.com/guide)')
  assert.match(content, /<h2>Status<\/h2>/)
  assert.match(content, /<strong>Healthy<\/strong>/)
  assert.match(content, /<pre><code class="language-sh">echo hello/)
  assert.match(content, /href="https:\/\/example.com\/guide"/)
})

test('removes executable markup, unsafe links and external images', () => {
  const content = render('<img src=x onerror="alert(1)"><script>alert(1)</script><svg onload="alert(1)"></svg>\n\n[open](javascript:alert%281%29)\n\n<iframe srcdoc="<script>alert(1)</script>"></iframe>')
  const document = new JSDOM(content).window.document
  assert.equal(document.querySelector('img,script,svg,iframe,form'), null)
  assert.equal(document.querySelector('[onerror],[onload],[srcdoc]'), null)
  assert.equal(document.querySelector('a')?.hasAttribute('href'), false)
  assert.equal(render(null), '')
})

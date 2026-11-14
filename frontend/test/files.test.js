import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'node:http'
import { once } from 'node:events'
import axios from 'axios'
import { readFileText } from '../src/lib/files.js'
import { errorMessage } from '../src/lib/errors.js'

test('reads JSON-looking files as unchanged text and preserves HTTP failures', async t => {
  const content = '  { "items": [1, 2], "enabled": true }\n'
  const server = createServer((request, response) => {
    const path = new URL(request.url, 'http://local').searchParams.get('path')
    response.setHeader('Content-Type', 'application/json')
    if (path === '/denied.json') {
      response.statusCode = 403
      response.end(JSON.stringify({ msg: 'Access denied' }))
    } else response.end(content)
  })
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  t.after(() => new Promise(resolve => server.close(resolve)))
  const client = axios.create({ baseURL: `http://127.0.0.1:${server.address().port}`, proxy: false })
  assert.equal(await readFileText('/config.json', { client }), content)
  await assert.rejects(readFileText('/denied.json', { client }), error => {
    assert.equal(error.response.status, 403)
    assert.equal(errorMessage(error, 'fallback'), 'Access denied')
    return true
  })
})

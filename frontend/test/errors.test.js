import test from 'node:test'
import assert from 'node:assert/strict'
import { errorMessage, isCancellation } from '../src/lib/errors.js'

test('preserves server errors from JSON and text responses', () => {
  for (const data of [{ message: 'message' }, { msg: 'message' }, { error: 'message' }, '{"msg":"message"}', 'message\n']) {
    assert.equal(errorMessage({ response: { data } }, 'fallback'), 'message')
  }
  assert.equal(errorMessage({ response: { data: '<html>Proxy failure</html>' } }, 'fallback'), 'fallback')
  assert.equal(errorMessage(new Error('Network Error'), 'offline'), 'offline')
})

test('recognizes dialog dismissal and aborted requests', () => {
  for (const error of ['cancel', 'close', { code: 'ERR_CANCELED' }, { name: 'AbortError' }]) {
    assert.equal(isCancellation(error), true)
  }
  assert.equal(isCancellation(new Error('failure')), false)
})

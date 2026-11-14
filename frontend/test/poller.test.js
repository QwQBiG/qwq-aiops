import test from 'node:test'
import assert from 'node:assert/strict'
import { setTimeout as wait } from 'node:timers/promises'
import { createPoller } from '../src/lib/poller.js'

test('slow requests never overlap and stopping aborts pending work', async () => {
  let calls = 0
  let active = 0
  let maximum = 0
  let resolveFirst
  let latestSignal
  const poller = createPoller(async signal => {
    calls++
    maximum = Math.max(maximum, ++active)
    latestSignal = signal
    await new Promise(resolve => { resolveFirst = resolve })
    active--
  }, { interval: 1 })
  poller.start()
  poller.start()
  await wait(15)
  assert.equal(calls, 1)
  resolveFirst()
  const deadline = Date.now() + 1000
  while (calls < 2 && Date.now() < deadline) await wait(5)
  assert.equal(calls, 2)
  assert.equal(maximum, 1)
  poller.stop()
  assert.equal(latestSignal.aborted, true)
  resolveFirst()
  await wait(10)
  assert.equal(calls, 2)
})

test('aborting an in-flight request does not surface an error', async () => {
  let errors = 0
  const poller = createPoller(signal => new Promise((resolve, reject) => {
    signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')))
  }), { onError: () => errors++ })
  poller.start()
  poller.stop()
  await wait(5)
  assert.equal(errors, 0)
})

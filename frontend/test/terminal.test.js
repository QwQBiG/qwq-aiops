import test from 'node:test'
import assert from 'node:assert/strict'
import { createTerminalConnection } from '../src/lib/terminal.js'

test('checks readiness, preserves repeated answers and releases socket handlers', () => {
  let socket
  class Socket {
    readyState = 0
    sent = []
    constructor() { socket = this }
    send(text) { this.sent.push(text) }
    close() { this.readyState = 3 }
  }
  const states = [], messages = [], errors = []
  const connection = createTerminalConnection('ws://local/ws/chat', {
    WebSocketImpl: Socket, onState: state => states.push(state),
    onMessage: message => messages.push(message), onError: error => errors.push(error)
  })
  connection.connect()
  assert.equal(connection.send('early'), false)
  socket.readyState = 1
  socket.onopen()
  assert.equal(connection.send('status'), true)
  for (const content of ['same answer', 'same answer']) socket.onmessage({ data: JSON.stringify({ type: 'answer', content }) })
  socket.onmessage({ data: JSON.stringify({ type: 'status', content: 'done' }) })
  socket.onmessage({ data: 'invalid json' })
  assert.equal(messages.length, 3)
  assert.deepEqual(errors, ['invalidMessage'])
  connection.disconnect()
  assert.equal(socket.readyState, 3)
  assert.equal(socket.onmessage, null)
  assert.equal(connection.send('after close'), false)
  assert.deepEqual(states, ['connecting', 'connected', 'disconnected'])
})

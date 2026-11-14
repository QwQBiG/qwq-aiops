export function createTerminalConnection(url, { WebSocketImpl = WebSocket, onState, onMessage, onError }) {
  let socket
  function connect() {
    if (socket && socket.readyState < 2) return
    const current = socket = new WebSocketImpl(url)
    onState('connecting')
    current.onopen = () => { if (socket === current) onState('connected') }
    current.onclose = () => { if (socket === current) onState('disconnected') }
    current.onerror = () => { if (socket === current) onError('connectionError') }
    current.onmessage = event => {
      if (socket !== current) return
      try {
        const data = JSON.parse(event.data)
        if (!['log', 'answer', 'status'].includes(data.type) || typeof data.content !== 'string') throw new Error('Invalid message')
        onMessage(data)
      } catch {
        onError('invalidMessage')
      }
    }
  }
  function disconnect() {
    if (socket) {
      socket.onopen = socket.onclose = socket.onerror = socket.onmessage = null
      socket.close()
      socket = null
    }
    onState('disconnected')
  }
  return {
    connect,
    disconnect,
    send(text) {
      if (!socket || socket.readyState !== 1) return false
      try { socket.send(text); return true } catch { onError('connectionError'); return false }
    }
  }
}

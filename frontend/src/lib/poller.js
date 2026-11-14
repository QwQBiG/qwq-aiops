// Schedule the next request after completion so slow responses cannot overlap.
export function createPoller(task, { interval = 2000, onError = () => {} } = {}) {
  let timer
  let controller
  let generation = 0
  let stopped = true

  async function run(current) {
    controller = new AbortController()
    try {
      await task(controller.signal)
    } catch (error) {
      if (!stopped && current === generation && !controller.signal.aborted) onError(error)
    } finally {
      if (!stopped && current === generation) timer = setTimeout(() => run(current), interval)
    }
  }

  return {
    start() {
      if (!stopped) return
      stopped = false
      run(++generation)
    },
    stop() {
      stopped = true
      generation++
      clearTimeout(timer)
      controller?.abort()
    }
  }
}

import createDOMPurify from 'dompurify'
import { marked } from 'marked'

const options = {
  ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'del', 'blockquote', 'ul', 'ol', 'li', 'pre', 'code', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'hr'],
  ALLOWED_ATTR: ['href', 'title', 'class'],
  ALLOW_DATA_ATTR: false
}

export function createContentRenderer(window) {
  const purifier = createDOMPurify(window)
  return text => purifier.sanitize(marked.parse(String(text ?? '')), options)
}

let renderer
export function renderMarkdown(text) {
  renderer ??= createContentRenderer(globalThis.window)
  return renderer(text)
}

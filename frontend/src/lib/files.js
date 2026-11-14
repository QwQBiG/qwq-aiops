import axios from 'axios'

// File contents may be valid JSON but must remain editable text.
export async function readFileText(path, { client = axios, ...options } = {}) {
  const { data } = await client.get('/api/files/content', {
    ...options, params: { path }, responseType: 'text', transformResponse: [body => body]
  })
  return data
}

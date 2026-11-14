import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync, readdirSync } from 'node:fs'
import { createI18n } from 'vue-i18n'

const messages = Object.fromEntries(['zh-CN', 'en-US'].map(locale => [locale, JSON.parse(readFileSync(new URL(`../src/i18n/locales/${locale}.json`, import.meta.url), 'utf8'))]))
const keys = (object, prefix = '') => Object.entries(object).flatMap(([key, value]) => typeof value === 'object' ? keys(value, `${prefix}${key}.`) : [`${prefix}${key}`])

test('both languages cover every current page label', () => {
  const chinese = keys(messages['zh-CN']), english = keys(messages['en-US'])
  assert.deepEqual(chinese.sort(), english.sort())
  const i18n = createI18n({ legacy: false, locale: 'en-US', messages })
  const directory = new URL('../src/views/', import.meta.url)
  const sources = readdirSync(directory).filter(name => name.endsWith('.vue')).map(name => readFileSync(new URL(name, directory), 'utf8'))
  sources.push(readFileSync(new URL('../src/App.vue', import.meta.url), 'utf8'))
  for (const source of sources) {
    for (const [, key] of source.matchAll(/\bt\('([a-zA-Z][\w.]+)'/g)) assert.equal(i18n.global.te(key, 'en-US'), true, key)
  }
  assert.equal(i18n.global.t('terminal.send'), 'Send')
  i18n.global.locale.value = 'zh-CN'
  assert.equal(i18n.global.t('terminal.send'), '发送')
})

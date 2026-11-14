import { spawnSync } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, readdirSync, rmdirSync, unlinkSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const args = process.argv.slice(2)
if (args.some(arg => arg !== '--frontend-only') || args.length > 1) {
  console.error('Usage: node scripts/build.mjs [--frontend-only]')
  process.exit(1)
}
if (Number(process.versions.node.split('.')[0]) < 22) {
  console.error('Node.js 22 or newer is required.')
  process.exit(1)
}

function run(command, commandArgs, cwd) {
  console.log(`> ${command} ${commandArgs.join(' ')}`)
  const result = command === 'npm' && process.platform === 'win32'
    ? spawnSync(process.env.ComSpec || 'cmd.exe', ['/d', '/s', '/c', `npm.cmd ${commandArgs.join(' ')}`], { cwd, stdio: 'inherit' })
    : spawnSync(command, commandArgs, { cwd, stdio: 'inherit' })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`${command} exited with status ${result.status}`)
}

// Preserve tracked placeholders while removing files from previous builds.
function cleanGenerated(directory, relative = '') {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const name = relative ? `${relative}/${entry.name}` : entry.name
    if (name === 'placeholder.txt' || name === 'assets/placeholder.js') continue
    const target = join(directory, entry.name)
    if (entry.isDirectory()) {
      cleanGenerated(target, name)
      if (name !== 'assets') rmdirSync(target)
    } else {
      unlinkSync(target)
    }
  }
}

try {
  const frontend = join(root, 'frontend')
  run('npm', ['ci', '--no-audit', '--no-fund'], frontend)
  run('npm', ['run', 'build'], frontend)
  const output = join(frontend, 'dist')
  if (!existsSync(join(output, 'index.html'))) throw new Error('Frontend build did not produce index.html')
  const embedded = join(root, 'internal', 'server', 'dist')
  mkdirSync(embedded, { recursive: true })
  console.log('Syncing frontend/dist to internal/server/dist')
  cleanGenerated(embedded)
  cpSync(output, embedded, { recursive: true })
  if (!args.includes('--frontend-only')) {
    const binary = join(root, 'bin', process.platform === 'win32' ? 'qwq.exe' : 'qwq')
    mkdirSync(dirname(binary), { recursive: true })
    run('go', ['build', '-trimpath', '-ldflags=-w -s', '-o', binary, './cmd/qwq'], root)
    console.log(`Built ${binary}`)
  }
} catch (error) {
  console.error(error.message)
  process.exit(1)
}

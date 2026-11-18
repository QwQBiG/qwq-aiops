import { readFileSync } from 'node:fs'

const report = JSON.parse(readFileSync(process.argv[2], 'utf8'))
const { packages } = JSON.parse(readFileSync(new URL('../frontend/package-lock.json', import.meta.url), 'utf8'))
const { vulnerabilities, metadata } = report
if (report.error || report.auditReportVersion !== 2 || !vulnerabilities || !metadata?.vulnerabilities) {
  throw new Error('The frontend audit did not return a valid report')
}

// Remove this exception when a compatible patched picomatch release is available.
const advisory = 'https://github.com/advisories/GHSA-fc5q-39xc-vjpf'
function isKnownBuildFinding(name, ancestors = new Set()) {
  const finding = vulnerabilities[name]
  if (ancestors.has(name) || !finding?.nodes?.length || !finding.via?.length ||
      !finding.nodes.every(node => packages[node]?.dev === true)) return false

  const chain = new Set(ancestors).add(name)
  return finding.via.every(source => typeof source === 'string'
    ? isKnownBuildFinding(source, chain)
    : name === 'picomatch' && source?.dependency === 'picomatch' && source.url === advisory)
}

const counts = metadata.vulnerabilities
const blocking = Object.entries(vulnerabilities).filter(([, finding]) => ['high', 'critical'].includes(finding.severity))
if (!Number.isInteger(counts.high) || !Number.isInteger(counts.critical) || blocking.length !== counts.high + counts.critical) {
  throw new Error('The frontend audit severity counts do not match its findings')
}
for (const [name, finding] of blocking) {
  if (finding.severity === 'high' && isKnownBuildFinding(name)) {
    console.warn(`Development dependency ${name}: accepted build-only finding (${advisory})`)
  } else {
    console.error(`${name}: unaccepted ${finding.severity} advisory`)
    process.exitCode = 1
  }
}

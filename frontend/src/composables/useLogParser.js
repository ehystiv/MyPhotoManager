import { computed } from 'vue'

const LEVEL_RULES = [
  { match: /errore|error|✗/i,                   level: 'error' },
  { match: /⚠|saltat|warn|annullat/i,           level: 'warn' },
  { match: /duplicat|↳ dupl\./i,                level: 'dupe' },
  { match: /senza data|nessuna data/i,          level: 'nodate' },
  { match: /^Trovate |^Scansione|^Watch|^\[/i,  level: 'info' },
]

const SUMMARY_PREFIX = '─'

function classify(line) {
  if (line.startsWith(SUMMARY_PREFIX)) return 'summary'
  for (const r of LEVEL_RULES) {
    if (r.match.test(line)) return r.level
  }
  if (/^\s*\d+ file|^\s*\d+ → senza_data|^\s+\d+ /.test(line)) return 'summary'
  if (/^\s+(foto|RAW\s)/.test(line)) return 'ok'
  return 'plain'
}

// Raggruppa blocchi di righe correlate (entry multi-line tipo "foto X / data / → dest")
export function parseLog(raw) {
  if (!raw) return []
  const lines = raw.split('\n')
  const entries = []
  let buffer = []
  let bufferLevel = 'plain'

  const flush = () => {
    if (buffer.length === 0) return
    entries.push({
      id: entries.length,
      level: bufferLevel,
      text: buffer.join('\n'),
    })
    buffer = []
    bufferLevel = 'plain'
  }

  for (const line of lines) {
    if (line.trim() === '') {
      flush()
      continue
    }
    if (line.startsWith('  ') && buffer.length > 0) {
      buffer.push(line)
      continue
    }
    flush()
    bufferLevel = classify(line)
    buffer.push(line)
  }
  flush()
  return entries
}

// Singleton: condiviso tra TabsNav (per badge) e LogTab (per rendering).
let singletonRef = null

export function useLogParser(logRef) {
  if (singletonRef && logRef === singletonRef.source) {
    return singletonRef.result
  }
  const entries = computed(() => parseLog(logRef.value))
  const errorCount = computed(() => entries.value.filter(e => e.level === 'error').length)
  const warnCount = computed(() => entries.value.filter(e => e.level === 'warn').length)
  const result = { entries, errorCount, warnCount }
  singletonRef = { source: logRef, result }
  return result
}

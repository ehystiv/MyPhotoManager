import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}

export function formatBytes(b) {
  if (b == null) return ''
  if (b >= 1 << 30) return (b / (1 << 30)).toFixed(1) + ' GB'
  if (b >= 1 << 20) return (b / (1 << 20)).toFixed(1) + ' MB'
  if (b >= 1 << 10) return Math.round(b / (1 << 10)) + ' KB'
  return b + ' B'
}

export function formatDuration(sec) {
  if (!sec || sec < 0 || !isFinite(sec)) return '—'
  sec = Math.round(sec)
  if (sec < 60) return sec + 's'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  if (m < 60) return `${m}:${String(s).padStart(2, '0')}`
  const h = Math.floor(m / 60)
  const mm = m % 60
  return `${h}:${String(mm).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

export function truncateMiddle(str, max = 48) {
  if (!str || str.length <= max) return str
  const keep = Math.floor((max - 1) / 2)
  return str.slice(0, keep) + '…' + str.slice(-keep)
}

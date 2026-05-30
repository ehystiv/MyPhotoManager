<script setup>
import { computed } from 'vue'
import { Image, Layers, ImageOff, HardDrive, Loader2 } from '@lucide/vue'
import { useStore } from '../composables/useStore'
import { formatDuration, truncateMiddle, formatBytes } from '../lib/utils'

const { state, progPct } = useStore()

const filename = computed(() => truncateMiddle(state.progress.filename, 42))
const eta = computed(() => formatDuration(state.progress.etaSec))
const throughput = computed(() => {
  const t = state.progress.throughput
  if (!t) return ''
  return `${t.toFixed(t > 5 ? 0 : 1)} file/s`
})

const scan = computed(() => state.scanResult)
const scanChips = computed(() => {
  if (!scan.value || scan.value.total === 0) return []
  const out = [
    { key: 'total',   icon: Image,    label: scan.value.total + ' foto',          color: 'text' },
  ]
  if (scan.value.raw)     out.push({ key: 'raw',    icon: Layers,    label: scan.value.raw + ' RAW',        color: 'text' })
  if (scan.value.others)  out.push({ key: 'others', icon: Image,     label: scan.value.others + ' altri',   color: 'text' })
  if (scan.value.noExif)  out.push({ key: 'noexif', icon: ImageOff,  label: scan.value.noExif + ' senza EXIF', color: 'warning', tip: 'Foto senza data EXIF. Verranno spostate in senza_data/ a meno che non sia attiva l\'opzione "Data da filesystem".' })
  if (scan.value.totalBytes) out.push({ key: 'bytes', icon: HardDrive, label: formatBytes(scan.value.totalBytes), color: 'muted' })
  return out
})
</script>

<template>
  <div class="statusbar" :class="{ active: state.running }">
    <div class="progress-track" v-if="state.running">
      <div
        class="progress-fill"
        :style="{ width: progPct + '%' }"
        role="progressbar"
        :aria-valuenow="progPct"
        aria-valuemin="0"
        aria-valuemax="100"
      ></div>
    </div>

    <!-- Running -->
    <div
      class="row"
      v-if="state.running"
      role="status"
      aria-live="polite"
      :aria-label="`${progPct}% completato, ${state.progress.current} di ${state.progress.total}`"
    >
      <span class="pct">{{ progPct }}%</span>
      <span class="counter">{{ state.progress.current }} / {{ state.progress.total }}</span>
      <span class="sep">·</span>
      <span class="filename">{{ filename }}</span>
      <span class="spacer" />
      <span v-if="throughput" class="meta">{{ throughput }}</span>
      <span v-if="state.progress.etaSec > 0" class="sep">·</span>
      <span v-if="state.progress.etaSec > 0" class="meta">ETA {{ eta }}</span>
    </div>

    <!-- Scanning skeleton -->
    <div class="row idle" v-else-if="state.scanning" aria-live="polite">
      <Loader2 :size="12" class="spin" />
      <span>Scansione in corso…</span>
      <div class="skel-chips">
        <span class="skel"></span>
        <span class="skel"></span>
        <span class="skel"></span>
      </div>
    </div>

    <!-- Scan chips -->
    <div class="row idle chips" v-else-if="scanChips.length">
      <span
        v-for="c in scanChips"
        :key="c.key"
        class="scan-chip"
        :data-color="c.color"
        :title="c.tip || ''"
      >
        <component :is="c.icon" :size="11" />
        {{ c.label }}
      </span>
      <span class="spacer" />
      <span v-if="state.watchStatus" class="watch">● {{ state.watchStatus }}</span>
      <span v-else class="hint">Pronto. <span class="kbd">⌘R</span> per avviare.</span>
    </div>

    <!-- Idle nessuno scan -->
    <div class="row idle" v-else>
      <span v-if="state.scanInfo">{{ state.scanInfo }}</span>
      <span v-else-if="state.watchStatus" class="watch">● {{ state.watchStatus }}</span>
      <span v-else class="hint">Pronto. <span class="kbd">⌘R</span> per avviare.</span>
    </div>
  </div>
</template>

<style scoped>
.statusbar {
  flex-shrink: 0;
  border-top: 1px solid hsl(var(--border));
  background: hsl(var(--bg));
}
.progress-track {
  height: 3px;
  background: hsl(var(--subtle));
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: hsl(var(--accent));
  transition: width .2s ease-out;
}
.row {
  height: 30px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  font-size: 11.5px;
  color: hsl(var(--muted));
}
.pct {
  color: hsl(var(--text));
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.counter { font-variant-numeric: tabular-nums; }
.filename {
  font-family: var(--font-mono, monospace);
  color: hsl(var(--text));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.sep { opacity: .5; }
.spacer { flex: 1; }
.meta { font-variant-numeric: tabular-nums; }
.idle .hint { font-size: 11.5px; }
.idle .watch { color: hsl(var(--success)); }

/* Chips */
.scan-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 20px;
  padding: 0 7px;
  border-radius: 5px;
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--text));
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
.scan-chip[data-color="muted"] { color: hsl(var(--muted)); }
.scan-chip[data-color="warning"] {
  color: hsl(var(--warning));
  background: hsl(var(--warning) / .1);
  border-color: hsl(var(--warning) / .25);
}

/* Skeleton */
.skel-chips { display: flex; gap: 6px; }
.skel {
  display: inline-block;
  width: 64px;
  height: 18px;
  border-radius: 5px;
  background: linear-gradient(90deg,
    hsl(var(--subtle)) 0%,
    hsl(var(--border)) 50%,
    hsl(var(--subtle)) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}
.skel:nth-child(2) { width: 50px; animation-delay: .15s; }
.skel:nth-child(3) { width: 72px; animation-delay: .3s; }
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
.spin {
  animation: spin 1s linear infinite;
  color: hsl(var(--accent));
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>

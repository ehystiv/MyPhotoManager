<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { Eye, Clock, Zap } from '@lucide/vue'
import { useStore } from '../../composables/useStore'

const { state, toggleWatch } = useStore()

const now = ref(Date.now())
let timer = null
onMounted(() => { timer = setInterval(() => now.value = Date.now(), 1000) })
onUnmounted(() => clearInterval(timer))

const lastScanAgo = computed(() => {
  if (!state.watchLastScanAt) return null
  const sec = Math.floor((now.value - state.watchLastScanAt) / 1000)
  if (sec < 5) return 'pochi istanti fa'
  if (sec < 60) return `${sec} secondi fa`
  const m = Math.floor(sec / 60)
  return m === 1 ? '1 minuto fa' : `${m} minuti fa`
})

const nextScanIn = computed(() => {
  if (!state.watchLastScanAt || !state.watchActive) return null
  const elapsed = Math.floor((now.value - state.watchLastScanAt) / 1000)
  const remaining = 10 - (elapsed % 10)
  return remaining
})
</script>

<template>
  <div class="watch-tab">
    <header class="head">
      <h2>Watch automatico</h2>
      <p>Quando attivo, la cartella di input viene scansionata ogni 10 secondi e i nuovi file vengono organizzati con le opzioni correnti.</p>
    </header>

    <div class="control">
      <div class="control-info">
        <Eye :size="20" />
        <div>
          <div class="control-title">Monitoraggio</div>
          <div class="control-desc">
            <span v-if="state.watchActive" class="active-text">Attivo su {{ state.prefs.inputDir || '—' }}</span>
            <span v-else>Non attivo</span>
          </div>
        </div>
      </div>
      <label class="toggle">
        <input
          type="checkbox"
          :checked="state.watchActive"
          @change="toggleWatch($event.target.checked)"
          :disabled="!state.prefs.inputDir && !state.watchActive"
        />
        <span class="track">
          <span class="thumb"></span>
        </span>
      </label>
    </div>

    <div v-if="state.watchActive" class="meta">
      <div class="meta-row" v-if="lastScanAgo">
        <Clock :size="13" />
        <span class="meta-label">Ultima scansione</span>
        <span class="meta-val">{{ lastScanAgo }}</span>
      </div>
      <div class="meta-row" v-if="nextScanIn != null">
        <Zap :size="13" />
        <span class="meta-label">Prossima scansione</span>
        <span class="meta-val">tra {{ nextScanIn }} s</span>
      </div>
      <div class="meta-row" v-if="state.watchStatus && !state.watchStatus.startsWith('Watch')">
        <span class="meta-label">Stato</span>
        <span class="meta-val">{{ state.watchStatus }}</span>
      </div>
    </div>

    <div v-if="!state.prefs.inputDir" class="warn">
      Seleziona una cartella di input per abilitare il watch.
    </div>
  </div>
</template>

<style scoped>
.watch-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  max-width: 720px;
  width: 100%;
  margin: 0 auto;
}
.head h2 {
  font-size: 16px;
  font-weight: 600;
  color: hsl(var(--text));
  margin: 0 0 6px 0;
}
.head p {
  font-size: 12.5px;
  color: hsl(var(--muted));
  margin: 0;
  line-height: 1.5;
}
.control {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
}
.control-info {
  display: flex;
  align-items: center;
  gap: 12px;
}
.control-info > svg { color: hsl(var(--muted)); }
.control-title {
  font-size: 13px;
  font-weight: 600;
  color: hsl(var(--text));
}
.control-desc {
  font-size: 12px;
  color: hsl(var(--muted));
  margin-top: 2px;
}
.active-text { color: hsl(var(--success)); }

.toggle { position: relative; cursor: pointer; flex-shrink: 0; }
.toggle input { position: absolute; opacity: 0; pointer-events: none; }
.toggle .track {
  display: inline-block;
  width: 38px;
  height: 22px;
  border-radius: 999px;
  background: hsl(var(--border));
  position: relative;
  transition: background .2s;
}
.toggle .thumb {
  position: absolute;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: white;
  top: 2px;
  left: 2px;
  transition: transform .2s, background .2s;
  box-shadow: 0 1px 3px rgba(0,0,0,.2);
}
.toggle input:checked + .track {
  background: hsl(var(--success));
}
.toggle input:checked + .track .thumb {
  transform: translateX(16px);
}
.toggle input:disabled + .track { opacity: .4; cursor: not-allowed; }

.meta {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  background: hsl(var(--subtle));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
}
.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
}
.meta-row > svg { color: hsl(var(--muted)); flex-shrink: 0; }
.meta-label { color: hsl(var(--muted)); min-width: 140px; }
.meta-val { color: hsl(var(--text)); font-variant-numeric: tabular-nums; }
.warn {
  padding: 10px 12px;
  background: hsl(var(--warning) / .12);
  color: hsl(var(--warning));
  border: 1px solid hsl(var(--warning) / .3);
  border-radius: 8px;
  font-size: 12px;
}
</style>

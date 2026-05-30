<script setup>
import { ref, computed, watch, nextTick, toRef } from 'vue'
import { Trash2, Copy, Search, ScrollText, ArrowDown } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { VList } from 'virtua/vue'
import { useStore } from '../../composables/useStore'
import { useLogParser } from '../../composables/useLogParser'

const { state } = useStore()

const filter = ref('')
const autoscroll = ref(true)
const scroller = ref(null)
const vlistRef = ref(null)

const { entries, errorCount, warnCount } = useLogParser(toRef(state, 'logText'))

const filtered = computed(() => {
  if (!filter.value) return entries.value
  const q = filter.value.toLowerCase()
  return entries.value.filter(e => e.text.toLowerCase().includes(q))
})

const useVirtual = computed(() => filtered.value.length > 200)

watch(() => filtered.value.length, async (newLen, oldLen) => {
  if (!autoscroll.value) return
  await nextTick()
  if (useVirtual.value && vlistRef.value) {
    vlistRef.value.scrollToIndex(newLen - 1, { align: 'end' })
  } else if (scroller.value) {
    scroller.value.scrollTop = scroller.value.scrollHeight
  }
})

function handleScroll(e) {
  const el = e?.target || scroller.value
  if (!el) return
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30
  autoscroll.value = atBottom
}

function handleVirtualScroll(offset) {
  if (!vlistRef.value) return
  const total = vlistRef.value.scrollSize || 0
  const view = vlistRef.value.viewportSize || 0
  autoscroll.value = total - offset - view < 30
}

function jumpToBottom() {
  autoscroll.value = true
  if (useVirtual.value && vlistRef.value) {
    vlistRef.value.scrollToIndex(filtered.value.length - 1, { align: 'end' })
  } else if (scroller.value) {
    scroller.value.scrollTop = scroller.value.scrollHeight
  }
}

async function copyAll() {
  try {
    await navigator.clipboard.writeText(state.logText || '')
    toast.success('Log copiato negli appunti')
  } catch {
    toast.error('Impossibile copiare')
  }
}

function clearLog() {
  state.logText = ''
  toast.info('Log pulito')
}
</script>

<template>
  <div class="log-tab">
    <div class="log-toolbar">
      <div class="search">
        <Search :size="12" class="search-icon" />
        <input
          v-model="filter"
          class="input search-input"
          placeholder="Filtra il log…"
          aria-label="Filtra log"
        />
      </div>

      <div class="counters" v-if="errorCount > 0 || warnCount > 0">
        <span v-if="errorCount > 0" class="counter danger" :title="`${errorCount} errori`">
          {{ errorCount }} err
        </span>
        <span v-if="warnCount > 0" class="counter warn" :title="`${warnCount} avvisi`">
          {{ warnCount }} warn
        </span>
      </div>

      <div class="actions">
        <span v-if="!autoscroll" class="autoscroll-hint" @click="jumpToBottom" role="button" tabindex="0">
          <ArrowDown :size="11" /> Vai in fondo
        </span>
        <button class="btn btn-ghost btn-sm" @click="copyAll" :disabled="!state.logText" title="Copia tutto">
          <Copy :size="12" /> Copia
        </button>
        <button class="btn btn-ghost btn-sm" @click="clearLog" :disabled="!state.logText" title="Svuota log (⌘L)">
          <Trash2 :size="12" /> Pulisci
        </button>
      </div>
    </div>

    <div v-if="!entries.length" class="empty">
      <ScrollText :size="32" />
      <div class="empty-title">Nessuna operazione</div>
      <div class="empty-desc">Avvia un'organizzazione o una scansione per vedere il log qui.</div>
    </div>

    <!-- Virtualized: >200 entries -->
    <VList
      v-else-if="useVirtual"
      ref="vlistRef"
      :data="filtered"
      class="log-body"
      :overscan="10"
      @scroll="handleVirtualScroll"
    >
      <template #default="{ item: e }">
        <div class="log-entry" :data-level="e.level">
          <pre>{{ e.text }}</pre>
        </div>
      </template>
    </VList>

    <!-- Standard scroller -->
    <div v-else ref="scroller" class="log-body" @scroll="handleScroll">
      <div
        v-for="e in filtered"
        :key="e.id"
        class="log-entry"
        :data-level="e.level"
      >
        <pre>{{ e.text }}</pre>
      </div>
    </div>

    <!-- Aria-live mute, dichiara cambio errori per screen reader -->
    <div class="sr-only" aria-live="polite">
      {{ errorCount > 0 ? `${errorCount} errori nel log` : '' }}
    </div>
  </div>
</template>

<style scoped>
.log-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.log-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid hsl(var(--border));
  flex-shrink: 0;
}
.search {
  flex: 1;
  position: relative;
  max-width: 320px;
}
.search-icon {
  position: absolute;
  left: 8px;
  top: 50%;
  transform: translateY(-50%);
  color: hsl(var(--muted));
  pointer-events: none;
}
.search-input { padding-left: 26px; }
.counters {
  display: flex;
  gap: 4px;
}
.counter {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 5px;
  font-size: 10.5px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.counter.danger {
  background: hsl(var(--danger) / .15);
  color: hsl(var(--danger));
}
.counter.warn {
  background: hsl(var(--warning) / .15);
  color: hsl(var(--warning));
}
.actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}
.autoscroll-hint {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10.5px;
  color: hsl(var(--accent));
  padding: 4px 8px;
  border-radius: 5px;
  background: hsl(var(--accent) / .12);
  border: 1px solid hsl(var(--accent) / .25);
  cursor: pointer;
  user-select: none;
  margin-right: 4px;
}
.autoscroll-hint:hover { background: hsl(var(--accent) / .2); }
.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: hsl(var(--muted));
  padding: 40px;
}
.empty-title {
  font-size: 13px;
  font-weight: 500;
  color: hsl(var(--text));
  margin-top: 10px;
}
.empty-desc { font-size: 12px; text-align: center; }
.log-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  background: hsl(var(--surface));
  font-family: 'SF Mono', 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 11.5px;
  line-height: 1.55;
}
.log-entry {
  padding: 2px 16px;
  border-left: 2px solid transparent;
}
.log-entry pre {
  white-space: pre-wrap;
  word-break: break-word;
  color: hsl(var(--text));
  font-family: inherit;
}
.log-entry[data-level="error"]   { border-left-color: hsl(var(--danger)); background: hsl(var(--danger) / .04); }
.log-entry[data-level="error"] pre { color: hsl(var(--danger)); }
.log-entry[data-level="warn"]    { border-left-color: hsl(var(--warning)); }
.log-entry[data-level="warn"] pre { color: hsl(var(--warning)); }
.log-entry[data-level="dupe"]    { border-left-color: hsl(var(--muted)); opacity: .8; }
.log-entry[data-level="nodate"]  { border-left-color: hsl(220 30% 50%); }
.log-entry[data-level="info"] pre   { color: hsl(var(--accent)); }
.log-entry[data-level="summary"] {
  border-top: 1px solid hsl(var(--border));
  margin-top: 6px;
  padding-top: 8px;
  background: hsl(var(--subtle));
}
.log-entry[data-level="summary"] pre { color: hsl(var(--text)); font-weight: 500; }
.log-entry[data-level="ok"] pre  { color: hsl(var(--text)); }
.log-entry[data-level="plain"] pre { color: hsl(var(--muted)); }

.sr-only {
  position: absolute;
  width: 1px; height: 1px;
  padding: 0; margin: -1px;
  overflow: hidden; clip: rect(0,0,0,0);
  white-space: nowrap; border: 0;
}
</style>

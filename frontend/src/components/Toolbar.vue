<script setup>
import { ref, computed } from 'vue'
import {
  Play, Square, ArrowRight, ChevronDown, X, History, Trash2,
  Sun, Moon, Monitor, MoreHorizontal, Info, Copy,
} from '@lucide/vue'
import { useStore } from '../composables/useStore'
import { useTheme } from '../composables/useTheme'
import PathDisplay from './PathDisplay.vue'
import Tooltip from './Tooltip.vue'
import { PopoverRoot, PopoverTrigger, PopoverPortal, PopoverContent } from 'reka-ui'
import { toast } from 'vue-sonner'

const mode = useTheme()
function cycleTheme() {
  const order = ['light', 'dark', 'auto']
  const i = order.indexOf(mode.value)
  mode.value = order[(i + 1) % order.length]
}

const emit = defineEmits(['start', 'confirm-unsafe'])

const {
  state, hasInputDir, canRun, isUnsafeOrganize,
  chooseInput, chooseOutput, clearOutput, stop, setRecent, clearRecentsList,
  showAbout,
} = useStore()

const recentsOpen = ref(false)
const menuOpen = ref(false)

async function copyPath(path) {
  if (!path) return
  try {
    await navigator.clipboard.writeText(path)
    toast.success('Path copiato negli appunti', { duration: 1800 })
  } catch {
    toast.error('Impossibile copiare')
  }
}

const startDisabledReason = computed(() => {
  if (state.running) return 'Operazione in corso'
  if (state.watchActive) return 'Watch attivo — disattivalo prima'
  if (!hasInputDir.value) return 'Seleziona una cartella di input'
  return ''
})

const stopDisabledReason = computed(() => {
  if (!state.running && !state.watchActive) return 'Nessuna operazione in corso'
  return ''
})

function handleStart() {
  if (isUnsafeOrganize.value && !state.prefs.confirmedUnsafeOnce) {
    emit('confirm-unsafe')
  } else {
    emit('start')
  }
}

function pickRecent(dir) {
  recentsOpen.value = false
  setRecent(dir)
}
</script>

<template>
  <div class="toolbar">
    <!-- Input dir + recents -->
    <div class="dir-block">
      <span class="dir-label">Input</span>
      <button
        class="dir-btn"
        @click="chooseInput"
        @contextmenu.prevent="copyPath(state.prefs.inputDir)"
        :title="state.prefs.inputDir ? `${state.prefs.inputDir}\n(right-click: copia)` : 'Scegli cartella'"
        :aria-label="state.prefs.inputDir ? `Input: ${state.prefs.inputDir}` : 'Scegli cartella di input'"
      >
        <PathDisplay :path="state.prefs.inputDir" placeholder="Scegli cartella…" />
      </button>
      <PopoverRoot v-if="state.prefs.recents?.length" v-model:open="recentsOpen">
        <PopoverTrigger as-child>
          <button class="btn btn-ghost btn-sm recents-btn" title="Cartelle recenti" aria-label="Cartelle recenti">
            <History :size="13" />
            <ChevronDown :size="11" />
          </button>
        </PopoverTrigger>
        <PopoverPortal>
          <PopoverContent :side-offset="6" align="end" class="recents-popover">
            <div class="recents-head">Cartelle recenti</div>
            <button
              v-for="r in state.prefs.recents"
              :key="r"
              class="recent-item"
              @click="pickRecent(r)"
              :title="r"
            >
              <PathDisplay :path="r" />
            </button>
            <button class="recent-clear" @click="clearRecentsList(); recentsOpen = false">
              <Trash2 :size="11" /> Svuota
            </button>
          </PopoverContent>
        </PopoverPortal>
      </PopoverRoot>
    </div>

    <ArrowRight :size="14" class="sep" />

    <!-- Output dir -->
    <div class="dir-block">
      <span class="dir-label">Output</span>
      <button
        class="dir-btn"
        @click="chooseOutput"
        @contextmenu.prevent="copyPath(state.prefs.outputDir || state.prefs.inputDir)"
        :title="state.prefs.outputDir ? `${state.prefs.outputDir}\n(right-click: copia)` : 'Stessa di input'"
        :aria-label="state.prefs.outputDir ? `Output: ${state.prefs.outputDir}` : 'Output stessa di input'"
      >
        <PathDisplay
          :path="state.prefs.outputDir"
          placeholder="Stessa di input"
          :variant="state.prefs.outputDir ? 'default' : 'accent'"
        />
      </button>
      <button v-if="state.prefs.outputDir" class="btn btn-ghost btn-sm" @click="clearOutput" title="Usa stessa di input" aria-label="Ripristina output a stessa di input">
        <X :size="12" />
      </button>
    </div>

    <div class="spacer" />

    <!-- Tema -->
    <Tooltip :content="`Tema: ${mode}`">
      <button class="btn btn-ghost btn-sm theme-toggle" @click="cycleTheme" :aria-label="`Cambia tema (corrente: ${mode})`">
        <Sun v-if="mode === 'light'" :size="13" />
        <Moon v-else-if="mode === 'dark'" :size="13" />
        <Monitor v-else :size="13" />
      </button>
    </Tooltip>

    <!-- Menu altre azioni -->
    <PopoverRoot v-model:open="menuOpen">
      <PopoverTrigger as-child>
        <button class="btn btn-ghost btn-sm" aria-label="Altre azioni">
          <MoreHorizontal :size="14" />
        </button>
      </PopoverTrigger>
      <PopoverPortal>
        <PopoverContent :side-offset="6" align="end" class="recents-popover">
          <button class="recent-item" @click="menuOpen = false; showAbout()">
            <Info :size="12" /> Informazioni
          </button>
          <button class="recent-item" @click="menuOpen = false; copyPath(state.prefs.inputDir)" :disabled="!state.prefs.inputDir">
            <Copy :size="12" /> Copia path input
          </button>
        </PopoverContent>
      </PopoverPortal>
    </PopoverRoot>

    <!-- Stop -->
    <Tooltip :content="stopDisabledReason">
      <button
        class="btn btn-secondary btn-md"
        @click="stop"
        :disabled="!state.running && !state.watchActive"
      >
        <Square :size="12" /> Stop
      </button>
    </Tooltip>

    <!-- Avvia -->
    <Tooltip :content="startDisabledReason">
      <button
        class="btn btn-primary btn-md"
        @click="handleStart"
        :disabled="!canRun"
      >
        <Play :size="12" :fill="canRun ? 'currentColor' : 'none'" />
        Avvia
        <span class="kbd kbd-on-primary">⌘R</span>
      </button>
    </Tooltip>
  </div>
</template>

<style scoped>
.toolbar {
  height: 48px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border-bottom: 1px solid hsl(var(--border));
  background: hsl(var(--bg));
  /* Le aree vuote della toolbar trascinano la finestra. */
  --wails-draggable: drag;
}
/* Gli elementi interattivi non devono trascinare (la proprietà eredita). */
.toolbar button,
.toolbar input,
.toolbar .dir-btn {
  --wails-draggable: no-drag;
}
.dir-block {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.dir-label {
  font-size: 11px;
  color: hsl(var(--muted));
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: .04em;
  padding-right: 4px;
}
.dir-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 8px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: hsl(var(--text));
  font-size: 12.5px;
  max-width: 260px;
  min-width: 0;
  transition: background .12s, border-color .12s;
}
.dir-btn:hover {
  background: hsl(var(--subtle));
  border-color: hsl(var(--border));
}
.sep { color: hsl(var(--muted)); flex-shrink: 0; }
.spacer { flex: 1; }
.recents-btn { padding: 0 4px; gap: 2px; }

.kbd-on-primary {
  background: rgba(255,255,255,.18);
  border-color: rgba(255,255,255,.22);
  color: rgba(255,255,255,.9);
}
</style>

<style>
.recents-popover {
  z-index: 200;
  background: hsl(var(--elevated));
  border: 1px solid hsl(var(--border));
  border-radius: 8px;
  box-shadow: 0 8px 28px rgba(0,0,0,.18);
  padding: 4px;
  min-width: 240px;
  max-height: 320px;
  overflow-y: auto;
  animation: tooltip-in .12s ease-out;
}
.recents-head {
  font-size: 10.5px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .05em;
  color: hsl(var(--muted));
  padding: 8px 8px 4px;
}
.recent-item {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 28px;
  padding: 0 8px;
  border-radius: 5px;
  background: transparent;
  border: none;
  color: hsl(var(--text));
  text-align: left;
  font-size: 12px;
}
.recent-item:hover { background: hsl(var(--subtle)); }
.recent-clear {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 24px;
  margin-top: 4px;
  padding: 0 8px;
  border-radius: 5px;
  background: transparent;
  border: none;
  color: hsl(var(--muted));
  font-size: 11.5px;
  border-top: 1px solid hsl(var(--border));
}
.recent-clear:hover { color: hsl(var(--danger)); background: hsl(var(--subtle)); }
</style>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Toaster, toast } from 'vue-sonner'
import 'vue-sonner/style.css'
import { useStore } from './composables/useStore'
import { useShortcuts } from './composables/useShortcuts'
import { useTheme } from './composables/useTheme'

import Titlebar from './components/Titlebar.vue'
import Toolbar from './components/Toolbar.vue'
import TabsNav from './components/TabsNav.vue'
import StatusBar from './components/StatusBar.vue'
import DropOverlay from './components/DropOverlay.vue'
import EmptyState from './components/EmptyState.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import DryRunBanner from './components/DryRunBanner.vue'

import OptionsTab from './components/tabs/OptionsTab.vue'
import LogTab from './components/tabs/LogTab.vue'
import ResultsTab from './components/tabs/ResultsTab.vue'
import DedupeTab from './components/tabs/DedupeTab.vue'
import WatchTab from './components/tabs/WatchTab.vue'

const {
  state, init, unbindEvents, persist,
  chooseInput, chooseOutput, start, stop,
} = useStore()

const mode = useTheme()
const unsafeConfirmOpen = ref(false)

useShortcuts({
  onChooseInput: chooseInput,
  onChooseOutput: chooseOutput,
  onStart: () => {
    if (state.running || state.watchActive) return
    handleStart()
  },
  onStop: stop,
  onClearLog: () => { state.logText = '' },
  onTab: (id) => { state.activeTab = id },
})

function handleStart() {
  const isUnsafe = !state.prefs.dryRun && !state.prefs.copyMode && !state.prefs.renameOnly
  if (isUnsafe && !state.prefs.confirmedUnsafeOnce) {
    unsafeConfirmOpen.value = true
  } else {
    start()
    state.activeTab = 'log'
  }
}

function onConfirmUnsafe({ remember }) {
  if (remember) {
    state.prefs.confirmedUnsafeOnce = true
    persist()
  }
  start()
  state.activeTab = 'log'
}

// Drag & drop
function onDragEnter(e) {
  if (e.dataTransfer?.types?.includes('Files')) {
    state.isDragOver = true
  }
}
function onDragLeave(e) {
  if (e.target === e.currentTarget) state.isDragOver = false
}

onMounted(init)
onUnmounted(unbindEvents)
</script>

<template>
  <div
    class="app-root"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent
    @dragleave.self="state.isDragOver = false"
    @drop.prevent="state.isDragOver = false"
  >
    <DropOverlay :visible="state.isDragOver" />

    <Titlebar />
    <Toolbar @start="handleStart" @confirm-unsafe="unsafeConfirmOpen = true" />
    <DryRunBanner />

    <template v-if="state.prefs.inputDir">
      <TabsNav />

      <main class="main" role="main" :aria-label="`Tab ${state.activeTab}`">
        <KeepAlive>
          <OptionsTab v-if="state.activeTab === 'options'" />
          <LogTab     v-else-if="state.activeTab === 'log'" />
          <ResultsTab v-else-if="state.activeTab === 'results'" />
          <DedupeTab  v-else-if="state.activeTab === 'dedupe'" />
          <WatchTab   v-else-if="state.activeTab === 'watch'" />
        </KeepAlive>
      </main>
    </template>

    <EmptyState v-else />

    <StatusBar />

    <ConfirmDialog
      v-model:open="unsafeConfirmOpen"
      title="Spostare i file?"
      description="L'operazione sposterà i file dalla cartella di input verso la destinazione. Per provare senza modificare i file abilita Dry-run o Copia."
      confirm-text="Sposta"
      :destructive="true"
      :show-remember="true"
      @confirm="onConfirmUnsafe"
    />

    <Toaster
      :theme="mode === 'auto' ? 'system' : mode"
      position="bottom-right"
      :duration="3500"
      close-button
      rich-colors
    />
  </div>
</template>

<style>
.app-root {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
  background: hsl(var(--bg));
  color: hsl(var(--text));
}
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
[data-wails-drag] { -webkit-app-region: drag; }
[data-wails-no-drag] { -webkit-app-region: no-drag; }
</style>

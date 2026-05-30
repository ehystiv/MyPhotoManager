<script setup>
import { FolderPlus, MousePointer2, Sparkles } from '@lucide/vue'
import { useStore } from '../composables/useStore'
import PathDisplay from './PathDisplay.vue'

const { state, chooseInput, setRecent } = useStore()
</script>

<template>
  <div class="empty-root">
    <div class="hero">
      <div class="icon-wrap">
        <Sparkles :size="28" />
      </div>
      <h1>Benvenuto in MyPhotoManager</h1>
      <p>Organizza, rinomina e deduplica le tue foto in pochi click.</p>

      <div class="cta-row">
        <button class="btn btn-primary btn-lg" @click="chooseInput">
          <FolderPlus :size="14" /> Scegli cartella
        </button>
        <div class="or">o</div>
        <div class="drag-hint">
          <MousePointer2 :size="14" />
          <span>Trascina una cartella sulla finestra</span>
        </div>
      </div>

      <div v-if="state.prefs.recents?.length" class="recents">
        <div class="recents-title">Cartelle recenti</div>
        <div class="recents-list">
          <button
            v-for="r in state.prefs.recents.slice(0, 5)"
            :key="r"
            class="recent-chip"
            @click="setRecent(r)"
          >
            <PathDisplay :path="r" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.empty-root {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  overflow-y: auto;
}
.hero {
  max-width: 460px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}
.icon-wrap {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  background: hsl(var(--accent) / .12);
  color: hsl(var(--accent));
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 14px;
}
h1 {
  font-size: 20px;
  font-weight: 600;
  color: hsl(var(--text));
  margin: 0;
}
p {
  color: hsl(var(--muted));
  font-size: 13px;
  margin: 0 0 20px 0;
}
.cta-row {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
}
.or {
  font-size: 11px;
  color: hsl(var(--muted));
  text-transform: uppercase;
  letter-spacing: .1em;
}
.drag-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: hsl(var(--muted));
  padding: 6px 12px;
  border: 1px dashed hsl(var(--border));
  border-radius: 8px;
}
.recents {
  margin-top: 8px;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.recents-title {
  font-size: 10.5px;
  text-transform: uppercase;
  letter-spacing: .08em;
  color: hsl(var(--muted));
  font-weight: 600;
}
.recents-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 100%;
  max-width: 360px;
}
.recent-chip {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 10px;
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 6px;
  color: hsl(var(--text));
  font-size: 12px;
  text-align: left;
}
.recent-chip:hover {
  background: hsl(var(--elevated));
  border-color: hsl(var(--accent) / .5);
}
</style>

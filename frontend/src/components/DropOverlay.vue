<script setup>
import { FolderPlus } from '@lucide/vue'
defineProps({ visible: Boolean })
</script>

<template>
  <Transition name="drop">
    <div v-if="visible" class="drop-overlay">
      <div class="drop-hint">
        <FolderPlus :size="44" />
        <div class="title">Rilascia la cartella qui</div>
        <div class="subtitle">Verrà impostata come input</div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.drop-overlay {
  position: fixed;
  inset: 8px;
  z-index: 300;
  background: hsl(var(--accent) / 0.12);
  border: 2px dashed hsl(var(--accent));
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: all;
  backdrop-filter: blur(6px);
}
.drop-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: hsl(var(--accent));
}
.title {
  font-size: 16px;
  font-weight: 600;
}
.subtitle {
  font-size: 12px;
  color: hsl(var(--muted));
}
.drop-enter-active, .drop-leave-active { transition: opacity .15s; }
.drop-enter-from, .drop-leave-to { opacity: 0; }
</style>

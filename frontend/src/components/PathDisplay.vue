<script setup>
import { computed } from 'vue'
import { Folder, FolderOpen } from '@lucide/vue'

const props = defineProps({
  path: { type: String, default: '' },
  placeholder: { type: String, default: 'Nessuna cartella' },
  variant: { type: String, default: 'default' }, // default | accent
})

const display = computed(() => {
  if (!props.path) return props.placeholder
  // Mostra solo le ultime 2 componenti del path per leggibilità.
  const parts = props.path.split('/').filter(Boolean)
  if (parts.length <= 2) return '/' + parts.join('/')
  return '…/' + parts.slice(-2).join('/')
})
</script>

<template>
  <span
    class="path-display"
    :class="{ 'is-empty': !path, 'is-accent': variant === 'accent' }"
    :title="path || placeholder"
  >
    <FolderOpen v-if="path" :size="13" class="icon" />
    <Folder v-else :size="13" class="icon" />
    <span class="text">{{ display }}</span>
  </span>
</template>

<style scoped>
.path-display {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 100%;
  min-width: 0;
  color: hsl(var(--text));
  font-size: 12.5px;
}
.path-display.is-empty {
  color: hsl(var(--muted));
  font-style: italic;
}
.path-display.is-accent {
  color: hsl(var(--accent));
}
.icon {
  flex-shrink: 0;
  opacity: .8;
}
.text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

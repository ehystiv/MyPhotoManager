<script setup>
import { computed, toRef } from 'vue'
import { SlidersHorizontal, Copy, Images } from '@lucide/vue'
import { useStore } from '../composables/useStore'
import { useLogParser } from '../composables/useLogParser'

const { state } = useStore()
const { errorCount } = useLogParser(toRef(state, 'logText'))

const cullingMarked = computed(() => state.cullingPhotos.filter(p => p.mark).length)

const tabs = computed(() => [
  { id: 'organizza', label: 'Organizza',          icon: SlidersHorizontal, kbd: '⌘1',
    badge: errorCount.value > 0 ? errorCount.value : (state.running ? '●' : (state.stats?.moved || null)),
    badgeColor: errorCount.value > 0 ? 'danger' : 'accent' },
  { id: 'dedupe',    label: 'Gestione duplicati', icon: Copy,              kbd: '⌘2' },
  { id: 'culling',   label: 'Revisiona',          icon: Images,            kbd: '⌘3',
    badge: cullingMarked.value > 0 ? cullingMarked.value : null },
])
</script>

<template>
  <div class="tabs-nav">
    <button
      v-for="t in tabs"
      :key="t.id"
      class="tab-trigger"
      :data-active="state.activeTab === t.id"
      @click="state.activeTab = t.id"
      :title="`${t.label} (${t.kbd})`"
    >
      <component :is="t.icon" :size="13" />
      <span>{{ t.label }}</span>
      <span
        v-if="t.badge != null"
        class="badge"
        :class="{
          'badge-success': t.badgeColor === 'success',
          'badge-danger':  t.badgeColor === 'danger',
        }"
      >{{ t.badge }}</span>
    </button>
  </div>
</template>

<style scoped>
.tabs-nav {
  height: 38px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 8px;
  border-bottom: 1px solid hsl(var(--border));
  background: hsl(var(--bg));
}
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 600;
  background: hsl(var(--accent));
  color: hsl(var(--accent-fg));
  margin-left: 2px;
}
.badge-success { background: hsl(var(--success)); }
.badge-danger  { background: hsl(var(--danger)); }
</style>

<script setup>
import { computed, ref, onMounted, watch } from 'vue'
import { useStore } from '../../composables/useStore'
import {
  FolderOpen, Image, Layers, SkipForward, Copy as CopyIcon, Trash2, BarChart3,
} from '@lucide/vue'
import { vAutoAnimate } from '@formkit/auto-animate/vue'

const { state, openOutputInFinder } = useStore()

const stats = computed(() => state.stats)

const totals = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'moved',   label: 'Organizzate', value: s.moved,   icon: Image,        color: 'accent' },
    { key: 'raw',     label: 'RAW',         value: s.raw,     icon: Layers,       color: 'text' },
    { key: 'others',  label: 'Altri',       value: s.others,  icon: Image,        color: 'text' },
    { key: 'skipped', label: 'Senza data',  value: s.skipped, icon: SkipForward,  color: 'warning' },
    { key: 'dupes',   label: 'Duplicati',   value: s.dupes,   icon: CopyIcon,     color: 'muted' },
    { key: 'cleaned', label: 'Cartelle vuote rimosse', value: s.cleaned, icon: Trash2, color: 'muted' },
  ].filter(t => t.value > 0)
})

const maxYear = computed(() => {
  if (!stats.value?.byYear?.length) return 1
  return Math.max(...stats.value.byYear.map(y => y.count))
})
</script>

<template>
  <div class="results-tab" v-if="stats">
    <header class="head">
      <h2>Risultato</h2>
      <button class="btn btn-secondary btn-md" @click="openOutputInFinder">
        <FolderOpen :size="13" /> Apri nel Finder
      </button>
    </header>

    <div class="totals" v-auto-animate>
      <div v-for="t in totals" :key="t.key" class="total-card" :data-color="t.color">
        <component :is="t.icon" :size="14" class="total-icon" />
        <div class="total-num">{{ t.value }}</div>
        <div class="total-label">{{ t.label }}</div>
      </div>
    </div>

    <div class="years" v-if="stats.byYear?.length">
      <div class="years-head">
        <BarChart3 :size="13" />
        <span>Per anno</span>
      </div>
      <div class="years-list" v-auto-animate>
        <div v-for="y in stats.byYear" :key="y.year" class="year-row">
          <span class="year">{{ y.year }}</span>
          <div class="bar-wrap">
            <div class="bar" :style="{ width: (y.count / maxYear * 100) + '%' }"></div>
          </div>
          <span class="count">{{ y.count }}</span>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="empty">
    <BarChart3 :size="32" />
    <div class="empty-title">Nessun risultato</div>
    <div class="empty-desc">I risultati dell'organizzazione compariranno qui.</div>
  </div>
</template>

<style scoped>
.results-tab {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 24px;
  max-width: 760px;
  width: 100%;
  margin: 0 auto;
  overflow-y: auto;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.head h2 {
  font-size: 16px;
  font-weight: 600;
  color: hsl(var(--text));
  margin: 0;
}
.totals {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
}
.total-card {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  position: relative;
}
.total-icon {
  color: hsl(var(--muted));
  margin-bottom: 4px;
}
.total-num {
  font-size: 22px;
  font-weight: 600;
  color: hsl(var(--text));
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}
.total-label {
  font-size: 11px;
  color: hsl(var(--muted));
}
.total-card[data-color="accent"] .total-icon,
.total-card[data-color="accent"] .total-num { color: hsl(var(--accent)); }
.total-card[data-color="warning"] .total-icon,
.total-card[data-color="warning"] .total-num { color: hsl(var(--warning)); }
.total-card[data-color="muted"] .total-num { color: hsl(var(--muted)); }

.years {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 14px 16px;
}
.years-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: hsl(var(--muted));
  margin-bottom: 10px;
}
.years-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.year-row {
  display: grid;
  grid-template-columns: 48px 1fr 60px;
  align-items: center;
  gap: 10px;
  font-size: 12.5px;
}
.year {
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--text));
}
.bar-wrap {
  height: 6px;
  background: hsl(var(--subtle));
  border-radius: 3px;
  overflow: hidden;
}
.bar {
  height: 100%;
  background: linear-gradient(90deg, hsl(var(--accent) / .6), hsl(var(--accent)));
  border-radius: 3px;
  transition: width .35s ease-out;
}
.count {
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: hsl(var(--muted));
  font-size: 12px;
}
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
</style>

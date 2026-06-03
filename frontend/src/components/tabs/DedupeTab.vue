<script setup>
import { ref, computed } from 'vue'
import { Copy, Search, Trash2, AlertTriangle } from '@lucide/vue'
import { useStore } from '../../composables/useStore'
import ConfirmDialog from '../ConfirmDialog.vue'
import { formatBytes } from '../../lib/utils'
import { vAutoAnimate } from '@formkit/auto-animate/vue'

const { state, dedupePreview, dedupeRemove } = useStore()

const confirmOpen = ref(false)

const result = computed(() => state.dedupeResult)

function onConfirm() {
  dedupeRemove()
}
</script>

<template>
  <div class="dedupe-tab">
    <div class="left-col">
      <header class="head">
        <h2>Gestione duplicati</h2>
        <p>Trova le foto con lo stesso identico contenuto, anche se hanno nomi diversi.</p>
      </header>

      <div class="actions">
        <button
          class="btn btn-secondary btn-md"
          @click="dedupePreview"
          :disabled="!state.prefs.inputDir || state.running || state.watchActive"
        >
          <Search :size="13" /> Cerca doppioni
        </button>
        <button
          class="btn btn-danger btn-md"
          @click="confirmOpen = true"
          :disabled="!state.prefs.inputDir || state.running || state.watchActive"
        >
          <Trash2 :size="13" /> Rimuovi doppioni
        </button>
      </div>
    </div>

    <div class="right-col">
      <div v-auto-animate class="result-wrap">
        <div v-if="result" class="result">
          <div class="result-head">
            <Copy :size="13" />
            <span>Risultato scansione</span>
          </div>
          <div class="row">
            <span class="label">Scansionate</span>
            <span class="val">{{ result.scanned }} foto</span>
          </div>
          <div class="row">
            <span class="label">Gruppi di doppioni</span>
            <span class="val">{{ result.groups }}</span>
          </div>
          <div class="row" v-if="result.removed > 0">
            <span class="label">{{ result.dryRun ? 'Da rimuovere' : 'Rimossi' }}</span>
            <span class="val accent">{{ result.removed }} file · {{ formatBytes(result.freed) }}</span>
          </div>
          <div class="row" v-else>
            <span class="label muted">Nessun doppione trovato</span>
          </div>
          <div v-if="result.dryRun && result.removed > 0" class="hint">
            <AlertTriangle :size="12" />
            Risultato di anteprima — premi "Rimuovi doppioni" per cancellare definitivamente.
          </div>
        </div>

        <div v-else class="empty">
          <Copy :size="32" />
          <div class="empty-title">Nessuna scansione</div>
          <div class="empty-desc">Premi «Cerca doppioni» per scansionare la cartella.</div>
        </div>
      </div>
    </div>
  </div>

  <ConfirmDialog
    v-model:open="confirmOpen"
    title="Eliminare definitivamente i doppioni?"
    description="Per ogni gruppo di file identici verrà mantenuto il primo (ordine alfabetico) e gli altri saranno rimossi. L'operazione non è reversibile."
    confirm-text="Sì, rimuovi"
    :destructive="true"
    @confirm="onConfirm"
  />
</template>

<style scoped>
.dedupe-tab {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 0;
  overflow: hidden;
}

.left-col {
  border-right: 1px solid hsl(var(--border));
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 24px;
  overflow-y: auto;
}

.right-col {
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow-y: auto;
}

.result-wrap {
  display: flex;
  flex-direction: column;
  flex: 1;
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
.actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result {
  background: hsl(var(--surface));
  border: 1px solid hsl(var(--border));
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.result-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: hsl(var(--muted));
  padding-bottom: 4px;
  border-bottom: 1px solid hsl(var(--border));
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
}
.label {
  color: hsl(var(--muted));
  font-size: 12px;
}
.val {
  color: hsl(var(--text));
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}
.val.accent { color: hsl(var(--accent)); }
.muted { color: hsl(var(--muted)); }
.hint {
  margin-top: 4px;
  padding: 8px 10px;
  background: hsl(var(--warning) / .1);
  border: 1px solid hsl(var(--warning) / .3);
  border-radius: 6px;
  font-size: 11.5px;
  color: hsl(var(--warning));
  display: flex;
  align-items: center;
  gap: 6px;
}

.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: hsl(var(--muted));
  padding: 40px;
  text-align: center;
}
.empty-title {
  font-size: 13px;
  font-weight: 500;
  color: hsl(var(--text));
  margin-top: 8px;
}
.empty-desc { font-size: 12px; }
</style>

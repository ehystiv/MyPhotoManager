<script setup>
import { Eye, X } from '@lucide/vue'
import { useStore } from '../composables/useStore'

const { state, persist } = useStore()

function disable() {
  state.prefs.dryRun = false
  persist()
}
</script>

<template>
  <Transition name="banner">
    <div v-if="state.prefs.dryRun" class="dry-banner" role="status">
      <Eye :size="14" />
      <span>
        <strong>Modalità anteprima</strong> attiva — nessun file verrà modificato.
      </span>
      <button class="btn-disable" @click="disable" title="Disattiva dry-run">
        Disattiva
      </button>
      <button class="btn-close" @click="disable" aria-label="Chiudi">
        <X :size="13" />
      </button>
    </div>
  </Transition>
</template>

<style scoped>
.dry-banner {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 14px;
  background: hsl(var(--warning) / .12);
  border-bottom: 1px solid hsl(var(--warning) / .25);
  color: hsl(var(--warning));
  font-size: 12px;
  line-height: 1.4;
}
.dry-banner > svg { flex-shrink: 0; }
.dry-banner > span { flex: 1; min-width: 0; }
strong { font-weight: 600; }
.btn-disable {
  background: hsl(var(--warning) / .18);
  color: hsl(var(--warning));
  border: 1px solid hsl(var(--warning) / .35);
  border-radius: 5px;
  height: 24px;
  padding: 0 10px;
  font-size: 11.5px;
  font-weight: 500;
}
.btn-disable:hover { background: hsl(var(--warning) / .28); }
.btn-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  background: transparent;
  color: hsl(var(--warning));
  border: none;
  opacity: .7;
}
.btn-close:hover { opacity: 1; background: hsl(var(--warning) / .12); }

.banner-enter-active, .banner-leave-active {
  transition: max-height .2s ease-out, opacity .15s, padding .2s ease-out;
  overflow: hidden;
}
.banner-enter-from, .banner-leave-to {
  max-height: 0;
  opacity: 0;
  padding-top: 0;
  padding-bottom: 0;
}
.banner-enter-to, .banner-leave-from {
  max-height: 50px;
}
</style>

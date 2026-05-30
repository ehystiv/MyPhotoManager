<script setup>
import { ref } from 'vue'
import {
  AlertDialogRoot, AlertDialogTrigger, AlertDialogPortal, AlertDialogOverlay,
  AlertDialogContent, AlertDialogTitle, AlertDialogDescription,
  AlertDialogCancel, AlertDialogAction,
} from 'reka-ui'
import { AlertTriangle } from '@lucide/vue'

defineProps({
  open: Boolean,
  title: { type: String, default: 'Conferma' },
  description: { type: String, default: '' },
  confirmText: { type: String, default: 'Conferma' },
  cancelText: { type: String, default: 'Annulla' },
  destructive: { type: Boolean, default: true },
  showRemember: { type: Boolean, default: false },
})
const emit = defineEmits(['update:open', 'confirm'])

const rememberChoice = ref(false)

function onConfirm() {
  emit('confirm', { remember: rememberChoice.value })
  emit('update:open', false)
}
</script>

<template>
  <AlertDialogRoot :open="open" @update:open="$emit('update:open', $event)">
    <AlertDialogPortal>
      <AlertDialogOverlay class="dialog-overlay" />
      <AlertDialogContent class="dialog-content">
        <div class="dialog-icon" :class="{ destructive }">
          <AlertTriangle :size="20" />
        </div>
        <div class="dialog-body">
          <AlertDialogTitle class="dialog-title">{{ title }}</AlertDialogTitle>
          <AlertDialogDescription class="dialog-desc">
            {{ description }}
            <slot name="description" />
          </AlertDialogDescription>
          <label v-if="showRemember" class="remember">
            <input type="checkbox" v-model="rememberChoice" />
            <span>Non chiedere più</span>
          </label>
        </div>
        <div class="dialog-actions">
          <AlertDialogCancel as-child>
            <button class="btn btn-secondary btn-md">{{ cancelText }}</button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <button
              class="btn btn-md"
              :class="destructive ? 'btn-danger' : 'btn-primary'"
              @click="onConfirm"
            >{{ confirmText }}</button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>

<style>
.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 400;
  background: rgba(0,0,0,.45);
  backdrop-filter: blur(2px);
  animation: fade-in .15s ease-out;
}
.dialog-content {
  position: fixed;
  z-index: 401;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: hsl(var(--elevated));
  border: 1px solid hsl(var(--border));
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0,0,0,.4);
  padding: 20px;
  width: 420px;
  max-width: 90vw;
  display: grid;
  grid-template-columns: auto 1fr;
  grid-template-areas:
    "icon body"
    "icon body"
    "actions actions";
  gap: 14px;
  animation: dialog-in .18s ease-out;
}
@keyframes dialog-in {
  from { opacity: 0; transform: translate(-50%, -48%) scale(.98); }
  to   { opacity: 1; transform: translate(-50%, -50%) scale(1); }
}
@keyframes fade-in {
  from { opacity: 0; } to { opacity: 1; }
}
.dialog-icon {
  grid-area: icon;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: hsl(var(--subtle));
  color: hsl(var(--warning));
  display: flex;
  align-items: center;
  justify-content: center;
}
.dialog-icon.destructive {
  color: hsl(var(--danger));
  background: hsl(var(--danger) / .12);
}
.dialog-body { grid-area: body; }
.dialog-title {
  font-size: 14px;
  font-weight: 600;
  color: hsl(var(--text));
  margin-bottom: 6px;
}
.dialog-desc {
  font-size: 12.5px;
  color: hsl(var(--muted));
  line-height: 1.5;
}
.dialog-actions {
  grid-area: actions;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
.remember {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  font-size: 12px;
  color: hsl(var(--muted));
  cursor: pointer;
}
.remember input { accent-color: hsl(var(--accent)); }
</style>

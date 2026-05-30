<script setup>
import { Check } from '@lucide/vue'
import Tooltip from './Tooltip.vue'

const props = defineProps({
  modelValue: Boolean,
  label: String,
  description: String,
  disabled: Boolean,
  disabledReason: String,
})
defineEmits(['update:modelValue'])
</script>

<template>
  <Tooltip :content="disabled ? (disabledReason || '') : ''">
    <label class="check" :class="{ disabled }">
      <span class="box" :class="{ checked: modelValue, disabled }">
        <Check v-if="modelValue" :size="11" :stroke-width="3" />
      </span>
      <input
        type="checkbox"
        :checked="modelValue"
        :disabled="disabled"
        @change="$emit('update:modelValue', $event.target.checked)"
      />
      <span class="content">
        <span class="label">{{ label }}</span>
        <span v-if="description" class="desc">{{ description }}</span>
      </span>
    </label>
  </Tooltip>
</template>

<style scoped>
.check {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
  padding: 4px 0;
  user-select: none;
}
.check.disabled {
  opacity: .45;
  cursor: not-allowed;
}
.box {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: 1px solid hsl(var(--border));
  background: hsl(var(--bg));
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: hsl(var(--accent-fg));
  transition: all .12s;
  margin-top: 1px;
}
.check:hover .box:not(.disabled) {
  border-color: hsl(var(--accent) / .6);
}
.box.checked {
  background: hsl(var(--accent));
  border-color: hsl(var(--accent));
}
input { display: none; }
.content {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.label {
  font-size: 12.5px;
  color: hsl(var(--text));
  line-height: 1.3;
}
.desc {
  font-size: 11px;
  color: hsl(var(--muted));
  line-height: 1.35;
}
</style>

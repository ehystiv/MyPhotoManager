<script setup>
import { TooltipRoot, TooltipTrigger, TooltipPortal, TooltipContent, TooltipProvider } from 'reka-ui'

defineProps({
  content: { type: String, default: '' },
  side: { type: String, default: 'top' },
  delay: { type: Number, default: 350 },
})
</script>

<template>
  <TooltipProvider :delay-duration="delay">
    <TooltipRoot>
      <TooltipTrigger as-child>
        <slot />
      </TooltipTrigger>
      <TooltipPortal>
        <TooltipContent
          v-if="content"
          :side="side"
          :side-offset="6"
          class="tooltip-content"
        >
          {{ content }}
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>

<style>
.tooltip-content {
  z-index: 200;
  background: hsl(var(--elevated));
  border: 1px solid hsl(var(--border));
  color: hsl(var(--text));
  padding: 5px 9px;
  font-size: 11.5px;
  border-radius: 6px;
  box-shadow: 0 4px 14px rgba(0,0,0,.12);
  max-width: 280px;
  line-height: 1.45;
  animation: tooltip-in .12s ease-out;
}
@keyframes tooltip-in {
  from { opacity: 0; transform: translateY(2px); }
  to   { opacity: 1; transform: translateY(0); }
}
</style>

<script setup>
// Drag area sottile per la finestra macOS — i traffic lights nativi
// vengono renderizzati sopra (TitleBarHiddenInset).
// Tutti i controlli sono nella Toolbar sottostante.
</script>

<template>
  <div class="titlebar"></div>
</template>

<style scoped>
.titlebar {
  height: 28px;
  flex-shrink: 0;
  background: hsl(var(--bg));
  /* Wails v2 riconosce questa proprietà come drag region (default CSSDragProperty). */
  --wails-draggable: drag;
}
</style>

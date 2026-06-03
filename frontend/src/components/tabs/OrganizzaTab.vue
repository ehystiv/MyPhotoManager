<script setup>
import OptionsTab from './OptionsTab.vue'
import LogTab from './LogTab.vue'
import ResultsTab from './ResultsTab.vue'
</script>

<template>
  <div class="organizza-tab">
    <div class="left-col">
      <OptionsTab />
    </div>
    <div class="right-col">
      <div class="right-top">
        <ResultsTab />
      </div>
      <div class="right-bottom">
        <LogTab />
      </div>
    </div>
  </div>
</template>

<style scoped>
.organizza-tab {
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
  min-height: 0;
  overflow: hidden;
}
.left-col :deep(.options-tab) {
  max-width: none;
  margin: 0;
}

.right-col {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.right-top {
  flex: 2;
  min-height: 0;
  border-bottom: 1px solid hsl(var(--border));
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.right-top :deep(.results-tab),
.right-top :deep(.empty) {
  max-width: none;
  margin: 0;
}

.right-bottom {
  flex: 3;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>

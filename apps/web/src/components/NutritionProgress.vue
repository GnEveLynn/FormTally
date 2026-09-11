<script setup lang="ts">
import type { MetricProgress } from '@formtally/api-contract/days'
defineProps<{ label: string; unit: string; progress: MetricProgress }>()
</script>
<template><article class="metric"><div><strong>{{ label }}</strong><span>{{ progress.consumed }} / {{ progress.target }} {{ unit }}</span></div><progress :value="Math.min(progress.percent, 100)" max="100"/><small v-if="progress.overBy">已超出 {{ progress.overBy }} {{ unit }}</small><small v-else>还可摄入 {{ progress.remaining }} {{ unit }}</small></article></template>
<style scoped>.metric{display:grid;gap:.45rem;padding:.9rem;border:1px solid #e0e8e4;border-radius:1rem;background:#fff}.metric div{display:flex;justify-content:space-between;gap:1rem}.metric span,small{color:var(--ft-color-text-muted)}progress{width:100%;accent-color:#55b98d}</style>

<script setup lang="ts">
import type { GoalCalculation } from '@formtally/api-contract/goals'
import { ref } from 'vue'
defineProps<{ calculation: GoalCalculation }>()
const open = ref(false)
</script>
<template>
  <section class="calculation">
    <button type="button" :aria-expanded="open" @click="open = !open">{{ open ? '收起计算说明' : '如何计算' }}</button>
    <div v-if="open">
      <h2>{{ calculation.method.displayName }}</h2>
      <p>{{ calculation.calculationVersion }} · {{ calculation.method.id }}</p>
      <p>{{ calculation.method.formulaExpression }}</p>
      <dl><template v-for="step in calculation.steps" :key="step.key"><dt>{{ step.label }}</dt><dd>{{ step.value }} {{ step.unit }}</dd></template></dl>
      <p>活动系数 {{ calculation.inputs.activityMultiplier }}，目标调整 {{ calculation.inputs.goalAdjustmentPercent }}%</p>
      <p>热量取整：{{ calculation.rounding.energy }}；营养素取整：{{ calculation.rounding.macros }}</p>
      <a :href="calculation.method.sourceUrl" target="_blank" rel="noreferrer">参考来源</a>
      <p>{{ calculation.disclaimer }}</p>
    </div>
  </section>
</template>
<style scoped>
.calculation { margin-top: 1rem; padding: 1rem; border-radius: 1rem; background: #fff; }
button { min-height: 44px; border: 0; background: transparent; font: inherit; font-weight: 700; }
dl { display: grid; grid-template-columns: 1fr auto; gap: .5rem; } dd { margin: 0; font-weight: 700; }
</style>

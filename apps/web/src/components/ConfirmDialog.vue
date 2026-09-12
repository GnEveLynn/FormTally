<script setup lang="ts">
defineProps<{ open: boolean; title: string; confirmLabel: string; busy?: boolean }>()
defineEmits<{ cancel: []; confirm: [] }>()
</script>

<template>
  <div v-if="open" class="backdrop" role="presentation">
    <section class="dialog" role="dialog" aria-modal="true" :aria-label="title">
      <h2>{{ title }}</h2>
      <slot />
      <div class="actions">
        <button type="button" data-action="cancel" :disabled="busy" @click="$emit('cancel')">取消</button>
        <button class="danger" type="button" data-action="confirm" :disabled="busy" @click="$emit('confirm')">{{ busy ? '处理中…' : confirmLabel }}</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.backdrop{position:fixed;inset:0;z-index:10;display:grid;place-items:center;padding:1rem;background:#17342d88}.dialog{width:min(100%,24rem);padding:1.25rem;border-radius:1rem;background:#fff;box-shadow:0 1rem 3rem #10251f44}.dialog h2{margin:0 0 .75rem}.actions{display:flex;justify-content:flex-end;gap:.75rem;margin-top:1.25rem}.actions button{min-height:44px;padding:0 1rem;border:0;border-radius:.75rem;font:inherit;font-weight:700}.danger{color:#fff;background:#a22b2b}
</style>

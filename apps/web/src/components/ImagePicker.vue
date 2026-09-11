<script setup lang="ts">
import { ref } from 'vue'
import { processImage } from '../domain/image'

defineProps<{ occurredAt: string; mealType: string }>()
const emit = defineEmits<{ selected: [file: File]; error: [message: string] }>()
const preview = ref('')

async function choose(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  try {
    const processed = await processImage(file)
    if (preview.value) URL.revokeObjectURL(preview.value)
    preview.value = URL.createObjectURL(processed)
    emit('selected', processed)
  } catch (error) {
    emit('error', error instanceof Error ? error.message : '图片处理失败')
  }
}
</script>

<template>
  <section>
    <p>{{ occurredAt }} · {{ mealType }}</p>
    <input type="file" accept="image/jpeg,image/png,image/webp" capture="environment" @change="choose" />
    <img v-if="preview" :src="preview" alt="餐食图片预览" />
  </section>
</template>

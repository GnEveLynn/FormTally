<script setup lang="ts">
import { ref } from 'vue'
import { processImage } from '../domain/image'

const props = defineProps<{ occurredAt: string; mealType: string }>()
const emit = defineEmits<{ selected: [file: File]; error: [message: string] }>()
const preview = ref('')
const mealLabels: Record<string, string> = { breakfast: '早餐', lunch: '午餐', dinner: '晚餐', snack: '加餐' }

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
    <p>{{ props.occurredAt }} · {{ mealLabels[props.mealType] ?? '未知餐别' }}</p>
    <label class="picker"><span class="camera">▣</span><strong>拍照或选择食物照片</strong><small>支持 JPEG、PNG、WebP</small><input type="file" accept="image/jpeg,image/png,image/webp" capture="environment" @change="choose" /></label>
    <img v-if="preview" :src="preview" alt="餐食图片预览" />
  </section>
</template>

<style scoped>
section{display:grid;gap:.7rem}p{margin:0;color:#71807a;font-size:.82rem}.picker{display:grid;justify-items:center;gap:.35rem;padding:1.5rem;border:1px dashed #cbd9d2;border-radius:1rem;color:#53635d;cursor:pointer}.picker input{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}.camera{font-size:2rem;color:#6f7c77}.picker small{color:#89948f;font-weight:400}img{width:100%;max-height:18rem;object-fit:cover;border-radius:1rem}
</style>

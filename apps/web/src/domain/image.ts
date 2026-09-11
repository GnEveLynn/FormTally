const supportedTypes = new Set(['image/jpeg', 'image/png', 'image/webp'])
const maxInputBytes = 10 * 1024 * 1024

export function validateImage(file: File): string | null {
  if (!supportedTypes.has(file.type)) return '请选择 JPEG、PNG 或 WebP 图片'
  if (file.size > maxInputBytes) return '图片不能超过 10 MB'
  return null
}

export async function processImage(file: File, maxDimension = 1600): Promise<File> {
  const validation = validateImage(file)
  if (validation) throw new Error(validation)
  const bitmap = await createImageBitmap(file, { imageOrientation: 'from-image' })
  const scale = Math.min(1, maxDimension / Math.max(bitmap.width, bitmap.height))
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(bitmap.width * scale))
  canvas.height = Math.max(1, Math.round(bitmap.height * scale))
  canvas.getContext('2d')?.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
  bitmap.close()
  const blob = await new Promise<Blob>((resolve, reject) =>
    canvas.toBlob((value) => (value ? resolve(value) : reject(new Error('图片处理失败'))), 'image/jpeg', 0.86),
  )
  return new File([blob], 'meal.jpg', { type: 'image/jpeg' })
}

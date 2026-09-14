const maxImageBytes = 10 * 1024 * 1024
const maxImageDimension = 1600

export type MediaErrorCode = 'cancelled' | 'permission_denied' | 'too_large' | 'invalid_image' | 'processing_failed'

export class MediaError extends Error {
  constructor(readonly code: MediaErrorCode, message: string) {
    super(message)
    this.name = 'MediaError'
  }
}

export interface ProcessedImage {
  path: string
  size: number
  width: number
  height: number
  mimeType: 'image/jpeg'
}

function chooseImage(source: 'camera' | 'album'): Promise<WechatMiniprogram.MediaFile> {
  return new Promise((resolve, reject) => wx.chooseMedia({
    count: 1,
    mediaType: ['image'],
    sourceType: [source],
    sizeType: ['original'],
    camera: 'back',
    success({ tempFiles }) {
      const file = tempFiles[0]
      if (tempFiles.length !== 1 || !file?.tempFilePath) {
        reject(new MediaError('invalid_image', '无法读取这张图片，请重新选择'))
        return
      }
      resolve(file)
    },
    fail({ errMsg }) {
      if (errMsg.includes('cancel')) reject(new MediaError('cancelled', '已取消选择图片'))
      else if (errMsg.includes('auth') || errMsg.includes('deny') || errMsg.includes('permission')) reject(new MediaError('permission_denied', '没有获得权限，你可以改从相册选择或重新授权'))
      else reject(new MediaError('processing_failed', '选择图片失败，请重试'))
    },
  }))
}

function imageInfo(path: string): Promise<WechatMiniprogram.GetImageInfoSuccessCallbackResult> {
  return new Promise((resolve, reject) => wx.getImageInfo({
    src: path,
    success: resolve,
    fail: () => reject(new MediaError('invalid_image', '无法读取这张图片，请重新选择')),
  }))
}

function fileSize(path: string): Promise<number> {
  return new Promise((resolve, reject) => wx.getFileSystemManager().getFileInfo({
    filePath: path,
    success: ({ size }) => resolve(size),
    fail: () => reject(new MediaError('processing_failed', '图片处理失败，请重试')),
  }))
}

function exportJpeg(path: string, width: number, height: number): Promise<string> {
  const canvas = wx.createOffscreenCanvas({ type: '2d' } as WechatMiniprogram.CreateOffscreenCanvasOption)
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext('2d')
  const image = canvas.createImage()

  return new Promise((resolve, reject) => {
    image.onload = () => {
      try { context.drawImage(image, 0, 0, width, height) } catch {
        reject(new MediaError('processing_failed', '图片处理失败，请重试'))
        return
      }
      wx.canvasToTempFilePath({
        canvas,
        width,
        height,
        destWidth: width,
        destHeight: height,
        fileType: 'jpg',
        quality: 0.86,
        success: ({ tempFilePath }) => resolve(tempFilePath),
        fail: () => reject(new MediaError('processing_failed', '图片处理失败，请重试')),
      })
    }
    image.onerror = () => reject(new MediaError('invalid_image', '无法读取这张图片，请重新选择'))
    image.src = path
  })
}

export async function chooseAndProcessMealImage(source: 'camera' | 'album'): Promise<ProcessedImage> {
  const selected = await chooseImage(source)
  if (selected.size > maxImageBytes) throw new MediaError('too_large', '图片不能超过 10 MB')

  const info = await imageInfo(selected.tempFilePath)
  if (info.width < 1 || info.height < 1 || info.type === 'unknown') {
    throw new MediaError('invalid_image', '无法读取这张图片，请重新选择')
  }
  const scale = Math.min(1, maxImageDimension / Math.max(info.width, info.height))
  const width = Math.max(1, Math.round(info.width * scale))
  const height = Math.max(1, Math.round(info.height * scale))
  const path = await exportJpeg(selected.tempFilePath, width, height)
  const size = await fileSize(path)
  if (size > maxImageBytes) throw new MediaError('too_large', '处理后的图片仍超过 10 MB，请重新选择')
  return { path, size, width, height, mimeType: 'image/jpeg' }
}

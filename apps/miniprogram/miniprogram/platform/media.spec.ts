import { beforeEach, describe, expect, it, vi } from 'vitest'
import { chooseAndProcessMealImage, MediaError } from './media'

describe('meal image processing', () => {
  let chooseOptions: Record<string, any>
  let exportOptions: Record<string, any>
  const getImageInfo = vi.fn()
  const drawImage = vi.fn()

  beforeEach(() => {
    chooseOptions = {}
    exportOptions = {}
    getImageInfo.mockReset()
    drawImage.mockReset()
    Object.assign(globalThis, {
      wx: {
        chooseMedia: vi.fn((options: Record<string, any>) => {
          chooseOptions = options
          options.success({
            tempFiles: [{ tempFilePath: '/tmp/source.png', size: 2 * 1024 * 1024, fileType: 'image' }],
            type: 'image',
          })
        }),
        getImageInfo,
        createOffscreenCanvas: vi.fn(() => {
          const image: Record<string, any> = { width: 3200, height: 1600, onload: () => undefined, onerror: () => undefined }
          Object.defineProperty(image, 'src', { set: () => queueMicrotask(() => image.onload()) })
          return { width: 0, height: 0, createImage: () => image, getContext: () => ({ drawImage }) }
        }),
        canvasToTempFilePath: vi.fn((options: Record<string, any>) => {
          exportOptions = options
          options.success({ tempFilePath: '/tmp/meal.jpg' })
        }),
        getFileSystemManager: vi.fn(() => ({
          getFileInfo: vi.fn((options: Record<string, any>) => options.success({ size: 350_000, digest: '' })),
        })),
      },
    })
    getImageInfo.mockImplementation((options: Record<string, any>) => options.success({
      path: '/tmp/source.png', width: 3200, height: 1600, type: 'png', orientation: 'up',
    }))
  })

  it('selects one image from the requested source and exports a 1600px JPEG', async () => {
    await expect(chooseAndProcessMealImage('camera')).resolves.toEqual({
      path: '/tmp/meal.jpg', size: 350_000, width: 1600, height: 800, mimeType: 'image/jpeg',
    })
    expect(chooseOptions).toMatchObject({ count: 1, mediaType: ['image'], sourceType: ['camera'], sizeType: ['original'], camera: 'back' })
    expect(drawImage).toHaveBeenCalledWith(expect.any(Object), 0, 0, 1600, 800)
    expect(exportOptions).toMatchObject({ width: 1600, height: 800, destWidth: 1600, destHeight: 800, fileType: 'jpg', quality: 0.86 })
  })

  it('rejects an input over 10 MB before decoding it', async () => {
    globalThis.wx.chooseMedia = vi.fn((options: any) => options.success({
      tempFiles: [{ tempFilePath: '/tmp/huge.jpg', size: 10 * 1024 * 1024 + 1, fileType: 'image' }], type: 'image',
    })) as typeof wx.chooseMedia
    await expect(chooseAndProcessMealImage('album')).rejects.toMatchObject({ code: 'too_large' })
    expect(getImageInfo).not.toHaveBeenCalled()
  })

  it('classifies undecodable images and denied permissions for display', async () => {
    getImageInfo.mockImplementationOnce((options: any) => options.fail({ errMsg: 'getImageInfo:fail decode' }))
    await expect(chooseAndProcessMealImage('album')).rejects.toMatchObject({ code: 'invalid_image', message: '无法读取这张图片，请重新选择' })

    globalThis.wx.chooseMedia = vi.fn((options: any) => options.fail({ errMsg: 'chooseMedia:fail auth deny' })) as typeof wx.chooseMedia
    const denied = await chooseAndProcessMealImage('camera').catch((error) => error)
    expect(denied).toBeInstanceOf(MediaError)
    expect(denied).toMatchObject({ code: 'permission_denied' })
  })
})

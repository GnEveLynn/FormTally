import { MediaError } from '../../platform/media'

export function mediaFailureView(error: unknown): { message: string; offerAlbum: boolean } {
  return {
    message: error instanceof Error ? error.message : '选择图片失败，请重试',
    offerAlbum: error instanceof MediaError && error.code === 'permission_denied',
  }
}

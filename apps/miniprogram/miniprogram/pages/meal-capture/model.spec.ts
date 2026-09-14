import { describe, expect, it } from 'vitest'
import { MediaError } from '../../platform/media'
import { mediaFailureView } from './model'

describe('meal capture presentation', () => {
  it('offers the album after camera permission is denied', () => {
    expect(mediaFailureView(new MediaError('permission_denied', '没有获得权限'))).toEqual({
      message: '没有获得权限', offerAlbum: true,
    })
    expect(mediaFailureView(new Error('图片处理失败'))).toEqual({ message: '图片处理失败', offerAlbum: false })
  })
})

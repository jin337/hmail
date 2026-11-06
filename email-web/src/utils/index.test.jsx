import dayjs from 'dayjs'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { flatTree, formatMailTime, getFileType, isSvg, throttle, transHtmlAttrs } from './index'

describe('getFileType', () => {
  it('按扩展名归类文件类型', () => {
    expect(getFileType('PNG')).toBe('image')
    expect(getFileType('mp4')).toBe('video')
    expect(getFileType('mp3')).toBe('audio')
    expect(getFileType('rar')).toBe('zip')
    expect(getFileType('xlsx')).toBe('excel')
    expect(getFileType('docx')).toBe('word')
    expect(getFileType('pptx')).toBe('ppt')
    expect(getFileType('pdf')).toBe('pdf')
  })

  it('未知类型默认为文本', () => {
    expect(getFileType('txt')).toBe('text')
    expect(getFileType(undefined)).toBe('text')
  })
})

describe('flatTree', () => {
  it('递归展开树结构', () => {
    const tree = [
      { value: 'a', children: [{ value: 'a1' }, { value: 'a2', children: [{ value: 'a21' }] }] },
      { value: 'b' },
    ]
    expect(flatTree(tree).map((item) => item.value)).toEqual(['a', 'a1', 'a2', 'a21', 'b'])
  })
})

describe('formatMailTime', () => {
  it('空值返回空串', () => {
    expect(formatMailTime('')).toBe('')
  })

  it('一周内显示相对时间', () => {
    expect(formatMailTime(dayjs().subtract(1, 'hour').format())).toBeTruthy()
  })

  it('超过一周显示月日，跨年显示年月日', () => {
    const lastYear = dayjs().subtract(1, 'year').format('YYYY-MM-DD HH:mm:ss')
    expect(formatMailTime(lastYear)).toBe(dayjs(lastYear).format('YYYY/MM/DD'))
  })
})

describe('transHtmlAttrs', () => {
  it('互换 img 上两个属性的值', () => {
    const html = '<div><img src="a.png" data-href="http://x/a.png"></div>'
    const result = transHtmlAttrs(html, 'src', 'data-href')
    expect(result).toContain('src="http://x/a.png"')
    expect(result).toContain('data-href="a.png"')
  })

  it('空内容原样返回', () => {
    expect(transHtmlAttrs('', 'src', 'data-href')).toBe('')
  })
})

describe('isSvg', () => {
  it('识别 svg 标签', () => {
    const svg = document.createElement('svg')
    expect(isSvg({ target: svg })).toBe(true)
    expect(isSvg({ target: document.createElement('div') })).toBe(false)
    expect(isSvg({ target: { type: 'path' } })).toBe(true)
    expect(isSvg(null)).toBe(false)
  })
})

describe('throttle', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('首次立即执行，随后合并为一次尾部调用', () => {
    const fn = vi.fn()
    const throttled = throttle(fn, 100)

    throttled('a')
    throttled('b')
    throttled('c')

    expect(fn).toHaveBeenCalledTimes(1)
    expect(fn).toHaveBeenLastCalledWith('a')

    vi.advanceTimersByTime(100)
    expect(fn).toHaveBeenCalledTimes(2)
    expect(fn).toHaveBeenLastCalledWith('c')
  })

  it('cancel 后不再触发尾部调用', () => {
    const fn = vi.fn()
    const throttled = throttle(fn, 100)

    throttled('a')
    throttled('b')
    throttled.cancel()

    vi.advanceTimersByTime(100)
    expect(fn).toHaveBeenCalledTimes(1)
  })
})

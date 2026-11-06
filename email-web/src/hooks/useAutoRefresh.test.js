import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useAutoRefresh } from 'src/hooks/useAutoRefresh'

// 控制页面可见性，验证后台标签页不发起刷新请求
function setHidden(hidden) {
  Object.defineProperty(document, 'hidden', {
    configurable: true,
    get: () => hidden,
  })
}

describe('useAutoRefresh', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setHidden(false)
  })

  afterEach(() => {
    vi.useRealTimers()
    setHidden(false)
  })

  it('到点后调用刷新函数', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoRefresh(refresh))

    act(() => result.current.restart())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60 * 1000)
    })

    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('页面不可见时跳过刷新，恢复可见后重新刷新', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoRefresh(refresh))

    setHidden(true)
    act(() => result.current.restart())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60 * 1000)
    })
    expect(refresh).not.toHaveBeenCalled()

    setHidden(false)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60 * 1000)
    })
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('clear 后不再触发刷新', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined)
    const { result } = renderHook(() => useAutoRefresh(refresh))

    act(() => result.current.restart())
    act(() => result.current.clear())
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10 * 60 * 1000)
    })

    expect(refresh).not.toHaveBeenCalled()
  })
})

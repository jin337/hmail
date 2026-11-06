import { useEffect, useRef } from 'react'

// 自动刷新节奏：前 20 次 1 分钟，之后 5 分钟，最多 50 次
const FAST_COUNT = 20
const MAX_COUNT = 50
const FAST_INTERVAL = 60 * 1000
const SLOW_INTERVAL = 5 * 60 * 1000

/**
 * 邮件列表自动刷新。
 * refresh 为「拉取最新一页」的函数，每次调用后重新计时。
 */
export function useAutoRefresh(refresh) {
  const timerRef = useRef(null)
  const countRef = useRef(0)
  const refreshRef = useRef(refresh)

  useEffect(() => {
    refreshRef.current = refresh
  }, [refresh])

  const clear = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  const schedule = () => {
    clear()
    if (countRef.current >= MAX_COUNT) return

    const delay = countRef.current >= FAST_COUNT ? SLOW_INTERVAL : FAST_INTERVAL
    timerRef.current = setTimeout(async () => {
      try {
        // 后台标签页不刷新：避免页面不可见时仍每分钟请求列表与未读数
        if (typeof document !== 'undefined' && document.hidden) {
          return
        }
        await refreshRef.current()
        countRef.current += 1
      } finally {
        schedule()
      }
    }, delay)
  }

  // 切换文件夹 / 重新计时
  const restart = () => {
    countRef.current = 0
    schedule()
  }

  useEffect(() => clear, [])

  return { restart, clear, schedule }
}

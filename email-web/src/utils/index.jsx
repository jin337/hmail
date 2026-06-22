/**
 * 节流函数
 * @param {Function} fn 需要被节流的函数
 * @param {Number} delay 延迟的时间（毫秒）
 * @returns {Function} 返回一个带有 cancel 方法的函数
 */
export function throttle(fn, delay) {
  let timer = null // 用于存储定时器
  let lastTime = 0 // 记录上一次执行的时间

  // 返回被节流包裹后的函数
  const throttledFn = function (...args) {
    const now = Date.now() // 获取当前时间
    const remaining = delay - (now - lastTime) // 计算距离上次执行还需要等待的时间

    // 如果已经超过了设定的延迟时间，或者这是第一次触发
    if (remaining <= 0) {
      // 清除可能存在的旧定时器（防止尾部执行）
      if (timer) {
        clearTimeout(timer)
        timer = null
      }
      // 执行函数
      fn.apply(this, args)
      // 更新上次执行时间
      lastTime = now
    } else if (!timer) {
      // 如果还没到时间，且没有设置尾部定时器，则设置一个定时器
      // 保证在停止滚动后，最后一次操作也能被执行
      timer = setTimeout(() => {
        fn.apply(this, args)
        lastTime = Date.now()
        timer = null
      }, remaining)
    }
  }

  // 提供一个 cancel 方法，用于在组件卸载时清理定时器
  throttledFn.cancel = function () {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    lastTime = 0
  }

  return throttledFn
}

// 获取文件类型
export function getFileType(contentType) {
  const type = contentType?.toLowerCase()

  // 图片
  const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp']
  // 视频
  const videoExts = ['mp4', 'webm', 'ogg', 'mov']
  // 音频
  const audioExts = ['mp3', 'wav', 'aac', 'flac', 'm4a']
  // 压缩文件
  const zipExts = ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz']
  // excel
  const excelExts = ['xls', 'xlsx', 'csv']
  // word
  const wordExts = ['doc', 'docx']
  // ppt
  const pptExts = ['ppt', 'pptx']
  // pdf
  const pdfExts = ['pdf']


  if (imageExts.includes(type)) return 'image'
  if (videoExts.includes(type)) return 'video'
  if (audioExts.includes(type)) return 'audio'
  if (zipExts.includes(type)) return 'zip'
  if (excelExts.includes(type)) return 'excel'
  if (wordExts.includes(type)) return 'word'
  if (pptExts.includes(type)) return 'ppt'
  if (pdfExts.includes(type)) return 'pdf'

  // 默认为文本
  return 'text'
}

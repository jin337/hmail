import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router'

import { createViewer } from 'jit-viewer'

import request from 'src/api/request'

const Preview = () => {
  const containerRef = useRef()
  const [searchParams] = useSearchParams()
  const [viewer, setViewer] = useState(null)

  // 获取文件类型
  const getFileType = (contentType) => {
    const type = contentType?.toLowerCase()

    // 图片
    const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp']
    // 视频
    const videoExts = ['mp4', 'webm', 'ogg', 'mov']
    // 音频
    const audioExts = ['mp3', 'wav', 'aac', 'flac', 'm4a']

    if (imageExts.includes(type)) return 'image'
    if (videoExts.includes(type)) return 'video'
    if (audioExts.includes(type)) return 'audio'

    return type
  }

  const init = async (data) => {
    const base64Str = decodeURIComponent(data)
    const jsonString = decodeURIComponent(atob(base64Str))
    const params = JSON.parse(jsonString)

    const result = await request.post('/api/mail/download', params, {
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(result)

    // 类型
    const type = getFileType(params.content_type)

    if (containerRef?.current) {
      const instance = createViewer({
        target: containerRef.current,
        theme: 'light',
        toolbar: true,
        file: url,
        filename: params.file_name,
        type: type,
      })
      instance.mount()
      setViewer(instance)
    }
  }

  useEffect(() => {
    const encodedData = searchParams.get('data')
    encodedData && init(encodedData)

    return () => viewer && viewer.destroy()
  }, [searchParams])

  return (
    <div className='relative z-10 h-[calc(100%-56px)] w-full'>
      <div ref={containerRef} className='h-full w-full' />
      <div className='absolute bottom-0 z-20 h-9 w-full border-t border-gray-300 bg-white'></div>
    </div>
  )
}
export default Preview

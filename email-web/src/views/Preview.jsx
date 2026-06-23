import { useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router'

import { createViewer } from 'jit-viewer'

import request from 'src/api/request'

const Preview = () => {
  const containerRef = useRef()
  const [searchParams] = useSearchParams()
  const [viewer, setViewer] = useState(null)

  const init = async (data) => {
    const base64Str = decodeURIComponent(data)
    const jsonString = decodeURIComponent(atob(base64Str))
    const params = JSON.parse(jsonString)

    const result = await request.post('/api/mail/download', params, {
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(result)

    // 类型

    if (containerRef?.current) {
      const instance = createViewer({
        target: containerRef.current,
        theme: 'light',
        toolbar: true,
        file: url,
        filename: params.file_name,
        type: params.type,
      })
      instance.mount()
      setViewer(instance)
    }
  }

  useEffect(() => {
    const encodedData = searchParams.get('preview')
    encodedData && init(encodedData)

    return () => viewer && viewer.destroy()
  }, [searchParams])

  return (
    <div className='relative z-10 h-screen w-full'>
      <div ref={containerRef} className='h-full w-full' />
      <div className='absolute bottom-0 z-20 h-9 w-full border-t border-gray-300 bg-white'></div>
    </div>
  )
}
export default Preview

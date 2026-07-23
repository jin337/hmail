import { Avatar } from '@arco-design/web-react'
import { useEffect, useState } from 'react'

import dayjs from 'dayjs'

const ImgUrl =
  import.meta.env.MODE === 'development'
    ? import.meta.env.VITE_BASE_URL
    : 'http://' + window.location.hostname + ':' + window.location.port

const AvatarImage = ({ email, name }) => {
  const [imgError, setImgError] = useState(false)
  const [time] = useState(dayjs().unix())

  useEffect(() => {
    const init = async () => {
      setImgError(false)
    }
    init()
  }, [email])

  if (!email) {
    return name?.slice(0, 1).toUpperCase() || '?'
  }

  const url = ImgUrl + `api/viewfile?url=static/avatars/${email}.webp?v=${time}`

  if (imgError) {
    return (
      <Avatar className={'min-w-10!'} style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
        {name?.slice(0, 1).toUpperCase() || '?'}
      </Avatar>
    )
  }

  return (
    <Avatar className={'min-w-10!'} style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
      <img
        className='h-full w-full object-cover'
        src={url}
        alt={name}
        onError={() => setImgError(true)}
        onLoad={() => setImgError(false)}
      />
    </Avatar>
  )
}

export default AvatarImage

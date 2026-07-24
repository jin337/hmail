import { Avatar } from '@arco-design/web-react'
import { useEffect, useState } from 'react'

import dayjs from 'dayjs'

const AvatarImage = ({ email, name, baseUrl }) => {
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

  const url = baseUrl + `/api/viewfile?url=static/avatars/${email}.webp?v=${time}`

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

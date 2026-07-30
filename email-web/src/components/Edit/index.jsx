import {} from 'react'

import './index.scss'
const Edit = (props) => {
  const { value, onChange } = props

  return (
    <div className='y-mail-wrap'>
      <div
        className='y-mail-content'
        contenteditable='true'
        onInput={(e) => onChange(e.target.innerHTML)}
        dangerouslySetInnerHTML={{ __html: value }}></div>
    </div>
  )
}
export default Edit

import { useState } from 'react'

import Edit from 'src/components/Edit'

const Demo = () => {
  const [value, setValue] = useState('<div><span>Hello World</span></div><div><b>Hello World</b></div>')
  const onChange = (value) => {
    console.log(value)
  }

  return (
    <div className='h-full w-full bg-white'>
      <Edit value={value} onChange={onChange} />
    </div>
  )
}
export default Demo

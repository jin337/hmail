import { useState } from 'react'

import Edit from 'src/components/Edit'

const Demo = () => {
  const [value, setValue] = useState('<div>Hello World</div><div>Hello World</div>')
  const onChange = (value) => {
    console.log(value)
  }

  return (
    <div className='h-full w-full bg-white p-4'>
      <Edit value={value} onChange={onChange} />
    </div>
  )
}
export default Demo

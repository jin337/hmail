import { } from 'react'

import { Layout } from '@arco-design/web-react'

// 组件
import Content from './Content'
import Menu from './Menu'

const MailLayout = () => {
  return (
    <Layout className='pr-4!'>
      <Layout.Sider width={220} theme='light' className='bg-transparent!'>
        <Menu />
      </Layout.Sider>

      <Content />
    </Layout>
  )
}
export default MailLayout

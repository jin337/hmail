import { Button, Layout } from '@arco-design/web-react'
import { IconLayout, IconMenu } from '@arco-design/web-react/icon'

// 组件
import Detail from './Detail'
import List from './List'

import { useMailContext } from './MailContext'

const Content = () => {
  const { isTable, setIsTable } = useMailContext()

  return (
    <Layout className='relative rounded-t-xl bg-white'>
      <Layout.Sider width={isTable ? '100%' : '360px'} className={`box-shadow-none z-10 flex-1`}>
        <List />
      </Layout.Sider>

      <Layout.Content>
        <div className='absolute top-4 right-4 z-20'>
          <Button size='small' onClick={() => setIsTable(!isTable)} icon={isTable ? <IconLayout /> : <IconMenu />}></Button>
        </div>
        <Detail />
      </Layout.Content>
    </Layout>
  )
}
export default Content

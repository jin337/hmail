import { Button, Layout } from '@arco-design/web-react'
import { IconLayout, IconMenu } from '@arco-design/web-react/icon'

// 组件
import Detail from './Detail'
import List from './List'

import { useMailContext } from './MailContext'

const Content = () => {
  const { isTable, setIsTable, currentMail, setCurrentMail } = useMailContext()

  // 切换模式
  const cutTable = () => {
    setCurrentMail(null)
    setIsTable(!isTable)
    localStorage.setItem('isTable', !isTable)
  }

  return (
    <Layout className='relative rounded-t-xl bg-white'>
      <Layout.Sider width={isTable ? (currentMail ? 0 : '100%') : '360px'} className={`box-shadow-none z-10 flex-1`}>
        {/* 邮件列表 */}
        <List />
      </Layout.Sider>

      <Layout.Content>
        {/* 切换模式按钮 */}
        {!(isTable && currentMail) && (
          <div className='absolute top-4 right-4 z-20'>
            <Button size='small' onClick={cutTable} icon={isTable ? <IconLayout /> : <IconMenu />}></Button>
          </div>
        )}
        {/* 邮件详情 */}
        <Detail />
      </Layout.Content>
    </Layout>
  )
}
export default Content

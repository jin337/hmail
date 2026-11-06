import { Layout } from '@arco-design/web-react'
import { useOutletContext } from 'react-router'

import WriteMail from 'src/components/WriteMail'
import { useMailbox } from 'src/hooks/useMailbox'
import Content from './Content'
import { MailProvider } from './MailContext'
import Menu from './Menu'

const MailLayout = () => {
  const { currentAccountId, baseUrl, userInfo, searchWord, setSearchWord, searchSeq } = useOutletContext()
  const { mailContext, compose } = useMailbox({
    currentAccountId,
    baseUrl,
    userInfo,
    searchWord,
    setSearchWord,
    searchSeq,
  })

  return (
    <MailProvider value={mailContext}>
      <Layout className='pr-4!'>
        <Layout.Sider width={220} className='bg-transparent!'>
          {/* 左列：文件夹导航 */}
          <Menu />
        </Layout.Sider>
        {mailContext.currentFolder?.key === 'compose' ? (
          // 写信
          <WriteMail key={compose.detail?.uid || '0'} {...compose} />
        ) : (
          // 右列：邮件列表 + 邮件详情
          <Content />
        )}
      </Layout>
    </MailProvider>
  )
}

export default MailLayout

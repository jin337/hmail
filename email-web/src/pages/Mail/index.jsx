import { useEffect, useMemo, useState } from 'react'
import { useOutletContext } from 'react-router'

import { Layout, Message } from '@arco-design/web-react'
import { IconDelete, IconEmail, IconFile, IconSend, IconStar } from '@arco-design/web-react/icon'

// 组件
import Content from './Content'
import Menu from './Menu'

import { MailProvider } from './MailContext'

import request from 'src/api/request'

import { getFileType, transHtml } from 'src/utils/index'

// 目录菜单
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'star', folder: 'Star', title: '星标邮件', icon: <IconStar /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

// 移动文件夹
const moveList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

// 筛选
const filterList = [
  {
    label: '筛选',
    children: [
      {
        label: '全部',
        value: 'all',
        key: 0,
      },
      {
        label: '未读',
        value: 'unread',
        key: 0,
      },
    ],
  },
  {
    label: '排序方式',
    children: [
      {
        label: '按日期',
        children: [
          {
            label: '由新到旧',
            value: 'date_desc',
            key: 1,
          },
          {
            label: '由旧到新',
            value: 'date_asc',
            key: 1,
          },
        ],
      },
      {
        label: '按大小',
        children: [
          {
            label: '由大到小',
            value: 'size_desc',
            key: 1,
          },
          {
            label: '由小到大',
            value: 'size_asc',
            key: 1,
          },
        ],
      },
    ],
  },
]

const MailLayout = () => {
  const { searchWord } = useOutletContext()

  const [folderList, setFolderList] = useState(menuList) // 目录菜单
  const [currentFolder, setCurrentFolder] = useState({}) // 当前文件夹

  const [mailList, setMailList] = useState({}) // 邮件列表
  const [filterKeys, setFilterKeys] = useState(['all', 'date_desc']) // 已筛选参数

  const [isTable, setIsTable] = useState(false)

  const [currentMail, setCurrentMail] = useState(null)

  // 获取当前邮件
  const getCurrentMail = async (item) => {
    const params = {
      uid: item.uid,
      folder: item.folder,
    }
    const { code, data, msg } = await request.post('/api/mail/detail', params, {
      headers: { 'X-Client-Host': window.location.host },
    })

    if (code === 200) {
      const newData = {
        ...data,
        attachments: data?.attachments?.map((e) => ({
          ...e,
          file_type: getFileType(e.file_type),
        })),
      }

      newData.content = transHtml(newData.content)
      setCurrentMail({ ...item, detail: newData })
    } else {
      Message.error(msg)
    }
  }

  // 获取邮件列表
  const getMailList = async (item) => {
    setCurrentMail(null) // 清空当前邮件

    let url = '/api/mail/list'
    let params = {
      ...item,
    }
    if (item.folder === 'Star') {
      url = '/api/mail/star-list'
      params = {
        filter: item.filter,
        keyword: item.keyword,
      }
    }

    let { code, data, msg } = await request.post(url, params)
    if (code === 200) {
      const list = (data?.list || []).map((e) => {
        const to_reply = e?.to_info?.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ')
        const cc_reply = e.cc ? e?.cc_info?.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ') : ''

        return {
          ...e,
          to_reply,
          cc_reply,
        }
      })

      setMailList({
        ...data,
        list,
      })
    } else {
      Message.error(msg)
    }
  }

  // 写信
  const onEdit = (record) => {
    let compose = { key: 'compose', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> }
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      return Message.warning('写邮件页已打开，请先关闭')
    }

    // 编辑-草稿
    if (record?.uid) {
      compose.title = record.subject
    }

    setCurrentFolder(compose)
    setFolderList((prev) => [compose, ...prev])
  }

  // 关闭写信
  const onCloseEdit = (record) => {
    setCurrentFolder(record || menuList[0])
    setFolderList((prev) => prev.filter((item) => item.key !== 'compose'))
  }

  // 筛选事件
  const onSelectFilter = (filter) => {
    setFilterKeys(filter)
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      page: 1,
      filter: filter,
    })
  }

  // 监听选中邮件
  useEffect(() => {
    const init = async () => {
      if (!currentMail?.uid) return
      await getCurrentMail({
        uid: currentMail.uid,
        folder: currentMail.folder,
      })
    }
    init()
  }, [currentMail?.uid])

  // 监听切换目录 / 搜索 / 筛选变化，重新加载
  useEffect(() => {
    const init = async () => {
      if (!currentFolder?.folder) return
      getMailList({
        folder: currentFolder.folder,
        keyword: searchWord,
        filter: filterKeys,
        page: 1,
        size: 25,
      })
    }
    init()
  }, [currentFolder?.folder, searchWord])

  // 立即加载第一页邮件
  useEffect(() => {
    const init = () => {
      setCurrentFolder(menuList[0])
    }
    init()
  }, [])

  // 统一上下文数据
  const mailContent = useMemo(
    () => ({
      folderList, // 目录菜单
      setFolderList, // 设置目录菜单
      currentFolder, // 当前文件夹
      setCurrentFolder, // 设置当前文件夹
      onEdit, // 写信
      onCloseEdit, // 关闭写信

      mailList, // 邮件列表
      filterList, // 筛选列表
      filterKeys, // 筛选参数
      onSelectFilter, // 筛选事件
      isTable,  // 是否表格模式
      setIsTable, // 设置表格模式

      currentMail, // 当前邮件
      setCurrentMail, // 设置当前邮件

      moveList, // 移动邮件
    }),
    [folderList, currentFolder, onEdit, onCloseEdit, onSelectFilter, mailList, filterKeys]
  )

  return (
    <MailProvider value={mailContent}>
      <Layout className='pr-4!'>
        <Layout.Sider width={220} className='bg-transparent!'>
          <Menu />
        </Layout.Sider>
        <Content />
      </Layout>
    </MailProvider>
  )
}
export default MailLayout

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

import { Avatar, Button, Card, Divider, Dropdown, Input, Layout, Menu, Message, Space, Spin, Table } from '@arco-design/web-react'
import {
  IconArrowLeft,
  IconAttachment,
  IconClose,
  IconDelete,
  IconDown,
  IconEdit,
  IconEmail,
  IconEye,
  IconFile,
  IconLayout,
  IconLeft,
  IconMenu,
  IconRedo,
  IconReply,
  IconRight,
  IconSearch,
  IconSend,
  IconStar,
  IconToBottom,
} from '@arco-design/web-react/icon'

import request from 'src/api/request'

import WriteMail from 'src/components/WriteMail'

import IconAudio from 'src/assets/file_aduio.svg'
import IconExcel from 'src/assets/file_excel.svg'
import IconImage from 'src/assets/file_image.svg'
import IconPdf from 'src/assets/file_pdf.svg'
import IconPpt from 'src/assets/file_ppt.svg'
import IconText from 'src/assets/file_text.svg'
import IconVideo from 'src/assets/file_video.svg'
import IconWord from 'src/assets/file_word.svg'
import IconZip from 'src/assets/file_zip.svg'

import IconMail from 'src/assets/mail.svg'
import IconMailOpen from 'src/assets/mail_open.svg'
import IconSent from 'src/assets/mail_sent.svg'
import IconStarUnselect from 'src/assets/mail_star.svg'
import IconStarSelect from 'src/assets/mail_star_open.svg'

import { getFileType, throttle } from 'src/utils/index'

// 左侧文件夹
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail className='text-lg!' /> },
  { key: 'star', folder: 'Star', title: '星标邮件', icon: <IconStar className='text-lg!' /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend className='text-lg!' /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile className='text-lg!' /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete className='text-lg!' /> },
]

const MailLayout = () => {
  const [userList, setUserList] = useState({})

  const [folderList, setFolderList] = useState(menuList)
  const [currentFolder, setCurrentFolder] = useState({})
  const [searchWord, setSearchWord] = useState('')

  const [loading, setLoading] = useState(false)
  const [mailList, setMailList] = useState([])
  const [total, setTotal] = useState(0)
  const [selectedRowKeys, setSelectedRowKeys] = useState([])

  const [currentLoading, setCurrentLoading] = useState(false)
  const [currentMail, setCurrentMail] = useState(null)
  const [writeMail, setWriteMail] = useState(null)
  const [newWriteMail, setNewWriteMail] = useState(null)
  const [isTable, setIsTable] = useState(false)

  const [_, setRefreshCount] = useState(0)
  const timerRef = useRef(null)
  const tableRef = useRef(null)
  const pageSize = 20
  const totalPages = Math.ceil(total / pageSize)

  // 切换选中邮件
  const onCutMail = (record, key) => {
    const index = mailList.findIndex((e) => e.uid === record.uid)
    if (key === 'prev') {
      onSelectMail(mailList[index - 1])
    } else if (key === 'next') {
      onSelectMail(mailList[index + 1])
    }
  }

  // 关闭写邮件页，返回收件箱
  const onClickCompose = (key) => {
    setNewWriteMail(null)
    setWriteMail(null)

    setFolderList((prev) => prev.filter((item) => item.key !== 'compose'))
    setTimeout(() => {
      const item = folderList.find((item) => item.key === key)
      loadMailList(item.key)
    }, 300)
  }

  // 写邮件
  const onWriteMail = (key, mailData) => {
    setWriteMail(null)
    setNewWriteMail(null)
    // 草稿页已打开
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      setCurrentFolder(folderList.find((item) => item.key === 'compose'))
      return Message.warning('写邮件页已打开，请先关闭')
    }

    let composeItem = { key: 'compose', folder: 'DRAFTS', title: '草稿', icon: <IconFile className='text-lg!' /> }
    if (key !== 'new') {
      composeItem.title = mailData?.subject
      setWriteMail(mailData)
    }
    setFolderList((prev) => [composeItem, ...prev])
    setCurrentFolder(composeItem)

    setSelectedRowKeys([])
    setMailList([])
    setSearchWord('')
  }

  // 点击导航栏
  const loadMailList = async (key) => {
    setRefreshCount(0)
    setSelectedRowKeys([])
    setMailList([])
    setTotal(0)
    setCurrentMail(null)
    setSearchWord('')

    // 当前文件夹
    const item = folderList.find((item) => item.key === key)
    setCurrentFolder(item)

    // 草稿页
    if (key === 'compose') {
      setWriteMail(newWriteMail || writeMail)
      return
    }

    // 加载邮件列表
    getMailData(item.folder)
    InboxRefresh()
  }

  // 自动刷新，获取收件箱邮件
  const InboxRefresh = () => {
    setRefreshCount((prevCount) => {
      // 1分钟刷新一次，10分钟以后，10分钟刷新一次
      let count = prevCount >= 10 ? 60 * 1000 : 10 * 60 * 1000 // 60秒和10分钟

      // 最大刷新次数
      if (prevCount >= 30) {
        if (timerRef.current) clearTimeout(timerRef.current)
        return
      }
      // 先清除已有定时器
      if (timerRef.current) clearTimeout(timerRef.current)

      timerRef.current = setTimeout(async () => {
        await getMailData('INBOX', '', 1, 1) //收件箱，无关键字，第一页，刷新
        //本次请求完成，再开启下一轮计时
        setRefreshCount((prev) => prev + 1)
        InboxRefresh()
      }, count)
      return prevCount
    })
  }

  // 预览附件
  const onPreviewAttachment = (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
      file_name: item.file_name,
      file_type: item.file_type,
    }
    const jsonString = JSON.stringify(params)
    const base64Str = btoa(encodeURIComponent(jsonString))
    window.open(`/web/preview?preview=${base64Str}`, '_blank')
  }

  // 下载附件
  const onDownloadAttachment = async (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
    }
    const res = await request.post('/api/mail/download', params, {
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(res)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', item.file_name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // 获取邮件详情
  const onSelectMail = async (item, e) => {
    // 排除干扰点击
    const targetElement = e?.target

    const isCheckboxClick = targetElement
      ? targetElement?.classList.contains('arco-checkbox') ||
        targetElement?.classList.contains('arco-checkbox-input') ||
        targetElement?.closest('.arco-checkbox')
      : false

    // 排除复选框的点击
    if (isCheckboxClick) return

    setCurrentLoading(true)
    const params = {
      uid: item.uid,
      folder: item.folder,
    }

    const { code, data, msg } = await request.post('/api/mail/detail', params)
    if (code === 200) {
      const newData = {
        ...data,
        attachments: data?.attachments?.map((e) => ({
          ...e,
          file_type: getFileType(e.file_type),
        })),
      }
      setCurrentMail({ ...item, detail: newData })
      if (item.key === 'drafts') {
        const newItem = {
          ...item,
          detail: newData,
          to_info: item.to_info.map((e) => ({ label: e.name, value: e.email })),
          cc_info: item?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
        }
        onWriteMail('rewrite', newItem)
      }
    } else {
      Message.error(msg)
    }
    setCurrentLoading(false)
  }

  // 搜索邮件
  const onSearch = async (val) => {
    setCurrentMail(null)
    setSelectedRowKeys([])

    getMailData(currentFolder.folder, val)

    scrollToTop()
  }

  // 获取邮件数据
  const getMailData = async (folder, keyword = '', page = 1, isRefresh) => {
    // 加载邮件列表
    setLoading(true)
    const url = folder === 'Star' ? '/api/mail/star-list' : '/api/mail/list'
    const params = { page, size: pageSize, folder, keyword }
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

      if (folder === 'INBOX') {
        let inbox = list.filter((item) => !item?.flags?.includes('Seen'))?.length || 0
        setFolderList((prev) =>
          prev.map((item) => {
            if (item.folder === folder) {
              return { ...item, total: inbox }
            }
            return item
          })
        )
      }
      // 刷新
      if (!isRefresh) {
        if (page === 1) {
          setMailList(list)
        } else {
          setMailList([...mailList, ...list])
        }
        setTotal(data?.total || 0)
      }
    } else {
      Message.error(msg)
    }

    setLoading(false)
  }

  // 删除邮件
  const onDelMail = async (ids) => {
    setSelectedRowKeys([])

    if (currentFolder.folder === 'Deleted') {
      const { code } = await request.post('/api/mail/delete', { folder: 'Deleted', uids: ids })
      if (code === 200) {
        Message.success('邮件已彻底删除')
      }
    } else {
      // 其他文件夹：移动到垃圾箱
      const { code } = await request.post('/api/mail/move', {
        uids: ids,
        from_folder: currentFolder.folder,
        to_folder: 'Deleted',
      })
      if (code === 200) {
        Message.success('已移入垃圾箱')
      }
    }
    setCurrentMail(null)
    loadMailList(currentFolder.key)
  }

  // 移动邮件
  const confirmMoveMail = async (e) => {
    const { code } = await request.post('/api/mail/move', {
      from_folder: currentFolder.folder,
      to_folder: e,
      uids: [currentMail.uid],
    })
    if (code !== 200) {
      Message.error('移动失败')
      return
    }
    Message.success('移动成功')
    loadMailList(currentFolder.key)
    setCurrentMail(null)
  }

  // 邮件原内容
  const FormContent = `<p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><br></p>
  <p style="line-height: 1;">
  <span style="color: rgb(140, 140, 140);">— </span>
  <span style="font-size: 12px; color: rgb(140, 140, 140);">原始邮件</span>
  <span style="color: rgb(140, 140, 140);"> ————————————</span>
  </p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 12px;">发件人：</span>
  <span style="font-size: 12px;">${currentMail?.from_info?.name} &lt;${currentMail?.from}&gt; </span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>发件时间：</span>
  <span style="font-size: 12px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>收件人：</span>
  <span style="font-size: 12px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>主题：</span>
  <span style="font-size: 12px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  const FormContentCc = `<p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><br></p>
  <p style="line-height: 1;">
  <span style="color: rgb(140, 140, 140);">— </span>
  <span style="font-size: 12px; color: rgb(140, 140, 140);">原始邮件</span>
  <span style="color: rgb(140, 140, 140);"> ————————————</span>
  </p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 12px;">发件人：</span>
  <span style="font-size: 12px;">${currentMail?.from_info?.name} &lt;${currentMail?.from}&gt;</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>发件时间：</span>
  <span style="font-size: 12px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>收件人：</span>
  <span style="font-size: 12px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>抄送：</span>
  <span style="font-size: 12px;">${currentMail?.cc_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>主题：</span>
  <span style="font-size: 12px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  // 回复邮件
  const onReply = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `回复: ${currentMail.subject}`,
      to_info: currentMail.to_info.map((e) => ({ label: e.name, value: e.email })),
      cc_info: currentMail?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
      is_reply: true,
    }

    if (currentFolder.key === 'inbox') {
      newMail.to_info = [{ label: currentMail.from_info.name, value: currentMail.from }]
    }
    onWriteMail('reply', newMail)
  }

  // 转发邮件
  const onForward = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `转发: ${currentMail.subject}`,
      to_info: [],
      cc_info: [],
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
      is_forward: true,
    }

    onWriteMail('forward', newMail)
  }

  // 星标邮件
  const onStar = async (item) => {
    const type = item?.flags?.includes('Flagged') ? 2 : 1 // 1:添加星标 2:取消星标
    const params = {
      uid: item.uid,
      folder: currentFolder.folder,
      status: 'Flagged',
      type,
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      setMailList((prev) => {
        let newList = [...prev]
        const index = newList.findIndex((item) => item.uid === params.uid)
        let flags = newList[index]?.flags || []
        if (type === 1) {
          flags.push('Flagged')
        } else {
          flags = flags?.filter((item) => item !== 'Flagged')
        }
        newList[index].flags = flags
        return newList
      })

      setCurrentMail((prev) => {
        let newItem = { ...prev }
        let flags = newItem?.flags || []
        if (type === 1) {
          flags.push('Flagged')
        } else {
          flags = flags?.filter((item) => item !== 'Flagged')
        }
        newItem.flags = flags

        return newItem
      })
    }
  }

  // 标记已读
  const onRead = async (item) => {
    const isRead = item?.flags?.includes('Seen')
    if (isRead) return
    const params = {
      uid: item.uid,
      folder: currentFolder.folder,
      status: 'Seen',
      type: 1,
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      setMailList((prev) => {
        const newList = [...prev]
        const index = newList.findIndex((item) => item.uid === params.uid)
        let flags = newList[index]?.flags || []
        flags.push('Seen')
        newList[index].flags = flags
        return newList
      })

      setFolderList((prev) =>
        prev.map((item) => {
          if (item.folder === 'INBOX') {
            return { ...item, total: item.total - 1 }
          }
          return item
        })
      )
    }
  }

  // 发送邮件&草稿
  const onSend = async (type, form, html, fileList, detail, setLoading) => {
    const values = form.getFieldsValue()
    if (!values.to_info || !values.subject) {
      Message.warning('请填写收件人和主题')
      return
    }

    const formData = new FormData()
    const to = values.to_info.map((e) => e.value)
    const cc = values?.cc_info?.map((e) => e.value) || ''

    formData.append('to', to)
    formData.append('cc', cc)
    formData.append('subject', values.subject)
    formData.append('content', html)
    if (detail?.uid) {
      formData.append('uid', detail.uid)
    }

    // 回复
    if (detail?.is_reply) {
      const references = detail.references + ' ' + detail?.message_id
      formData.append('in-reply-to', detail?.message_id)
      formData.append('references', references)
    }

    // 转发
    if (detail?.is_forward) {
      const references = detail.references + ' ' + detail?.message_id
      formData.append('references', references)
    }

    const partIds = []
    fileList.forEach((file) => {
      if (file?.part_id) {
        partIds.push(file.part_id)
      } else {
        formData.append('files', file?.originFile)
      }
    })
    if (partIds.length > 0) {
      formData.append('part_ids', partIds.join(','))
    }

    setLoading(true)
    let url = '/api/mail/send'
    if (type === 'Drafts') {
      url = '/api/mail/save-draft'
    }
    const { code, msg } = await request.post(url, formData, { headers: { 'Content-Type': 'multipart/form-data' } })
    if (code === 200) {
      Message.success(msg)
      onClickCompose(type === 'Sent' ? 'sent' : 'drafts')
    } else {
      Message.error(msg)
    }
    setLoading(false)
  }

  // 获取用户列表
  const getUserList = async () => {
    const { code, data, msg } = await request.post('/api/user/list')
    if (code === 200) {
      setUserList(data)
    } else {
      Message.error(msg)
    }

    loadMailList('inbox')
  }
  // 初始加载邮件列表
  useEffect(() => {
    getUserList()
  }, [])

  //   滚动到顶部
  const scrollToTop = () => {
    const scrollContainer = tableRef?.current?.querySelector('.arco-table-body')
    if (scrollContainer) {
      // 使用平滑滚动回到顶部
      scrollContainer.scrollTo({
        top: 0,
        behavior: 'smooth',
      })
    }
  }

  // 滚动加载
  const throttledScrollHandler = useMemo(
    () =>
      throttle((e) => {
        const { scrollTop, scrollHeight, clientHeight } = e.target
        const distanceToBottom = scrollHeight - scrollTop - clientHeight

        if (distanceToBottom <= 300 && !loading) {
          let currentPage = Math.ceil(mailList.length / pageSize)
          if (currentPage < totalPages) {
            getMailData(currentFolder.folder, searchWord, currentPage + 1)
          }
        }
      }, 500),
    [totalPages, mailList.length, searchWord]
  )

  const onScroll = useCallback(
    (e) => {
      throttledScrollHandler(e)
    },
    [throttledScrollHandler]
  )

  // 监听滚动事件
  useEffect(() => {
    const scrollContainer = tableRef?.current?.querySelector('.arco-table-body')

    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', onScroll)
    }

    return () => {
      if (scrollContainer) {
        scrollContainer.removeEventListener('scroll', onScroll)
      }
      throttledScrollHandler.cancel()
    }
  }, [onScroll, throttledScrollHandler])

  // 销毁定时器
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  return (
    <Layout className='flex-1'>
      {/* 左列：文件夹导航 */}
      <Layout.Sider width={220} theme='light' className='mail-menu box-shadow-none bg-transparent!'>
        <div className='p-4'>
          <Button type='primary' icon={<IconEdit />} long onClick={() => onWriteMail('new')}>
            写信
          </Button>
        </div>
        <Menu
          className={'bg-transparent! px-2'}
          selectedKeys={[currentFolder?.key || '']}
          onClickMenuItem={(key) => loadMailList(key)}>
          {folderList.map((item) => (
            <Menu.Item key={item.key} className='leading-8!'>
              <div className='flex items-center'>
                {item.icon}
                <span className='inline-block w-27 overflow-hidden align-middle text-ellipsis whitespace-nowrap'>
                  {item.title}
                </span>
              </div>
              {item?.key === 'inbox' && item?.total > 0 && <span className='font-medium text-blue-600'>{item.total}</span>}
              {item?.key === 'compose' && <IconClose className='m-0!' onClick={() => onClickCompose('inbox')} />}
            </Menu.Item>
          ))}
        </Menu>
      </Layout.Sider>

      {/* 写信 */}
      {currentFolder?.key === 'compose' && (
        <Spin className={'mr-4 w-full rounded-t-xl bg-white'} block loading={currentLoading}>
          <WriteMail
            key={writeMail?.uid || '0'}
            detail={writeMail}
            userList={userList?.list || []}
            onChange={setNewWriteMail}
            onClose={onClickCompose}
            onSend={onSend} // 传递邮件发送函数
          />
        </Spin>
      )}

      {currentFolder?.key !== 'compose' && (
        <Layout className='relative mr-4! rounded-t-xl'>
          {/* 切换模式按钮 */}
          {!(isTable && currentMail) && (
            <div className={`absolute top-4 right-4 z-20`}>
              <Button
                size='small'
                onClick={() => {
                  setIsTable(!isTable)
                  setCurrentMail(null)
                }}
                icon={isTable ? <IconLayout /> : <IconMenu />}></Button>
            </div>
          )}
          {/* 搜索框 */}
          <div className='fixed top-0 z-10 w-125 py-3'>
            <Input.Search
              prefix={<IconSearch />}
              placeholder='搜索主题/发件人'
              searchButton
              allowClear
              value={searchWord}
              onChange={setSearchWord}
              onSearch={onSearch}
              onClear={onSearch}
            />
          </div>
          {/* 中列：邮件列表 */}
          <Layout.Sider
            ref={tableRef}
            width={isTable ? (currentMail ? 0 : '100%') : 360}
            className={`box-shadow-none z-10 flex-1`}>
            <Table
              size='middle'
              loading={loading}
              scroll={{ y: 'calc(100vh - 116px)' }}
              className={`email-list h-full ${isTable ? 'email-table' : ''}`}
              rowKey='uid'
              pagination={false}
              border={{
                wrapper: false,
              }}
              rowClassName={(record) => {
                const selectClass = currentMail?.uid === record?.uid ? ' select-row' : ''
                return 'cursor-pointer' + selectClass
              }}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys: selectedRowKeys,
                onChange: (selectedRowKeys) => setSelectedRowKeys(selectedRowKeys),
              }}
              onRow={(record) => {
                return {
                  onClick: (e) => {
                    onSelectMail(record, e)
                  },
                }
              }}
              columns={[
                {
                  title: (
                    <div className='flex items-center justify-between'>
                      <span className={'text-base font-bold'}>{currentFolder?.title}</span>
                      <Space>
                        {selectedRowKeys.length > 0 && (
                          <Button size='mini' icon={<IconDelete />} onClick={() => onDelMail(selectedRowKeys)}>
                            {currentFolder.key === 'delete' ? '清空' : '删除'}
                          </Button>
                        )}
                        <span className={`${isTable ? 'mr-9' : ''}`}>共 {total} 封</span>
                      </Space>
                    </div>
                  ),
                  dataIndex: 'date',
                  render: (text, record) => (
                    <div className={record?.flags?.includes('Seen') ? '' : 'font-bold'} onClick={() => onRead(record)}>
                      <div className={`flex items-center justify-between gap-2 ${!isTable ? 'mb-1' : ''}`}>
                        <div className={` ${isTable ? 'flex' : ''}`}>
                          <div className={`flex items-center gap-1.5 ${isTable ? 'w-60!' : ''}`}>
                            {record?.flags?.includes('Seen') ? <IconMailOpen /> : <IconMail />}
                            {currentFolder?.key === 'sent' ? (
                              <>
                                <IconSent />
                                {record?.to_info?.map((t) => t.name).join(', ') || record?.to}
                                {record?.cc_info?.length > 0 ? ',  ' : ''}
                                {record?.cc_info?.map((t) => t.name).join(', ') || record?.cc}
                              </>
                            ) : (
                              record?.from_info.name || record?.from
                            )}

                            {record.has_attach ? <IconAttachment className='text-gray-400!' /> : ''}
                          </div>
                          {isTable && (
                            <div className='flex gap-1'>
                              <div className={'max-w-50 truncate'}>{record?.subject || ''}</div>
                              <div className={'max-w-50 truncate font-light text-gray-400'}>{record?.text || ''}</div>
                            </div>
                          )}
                        </div>
                        <div className='flex gap-4 text-right'>
                          {isTable && <div className='w-16'>{record.size}</div>}
                          <div className='w-16'>
                            {dayjs(record?.send_time).isBefore(dayjs().subtract(1, 'week'))
                              ? dayjs(record?.send_time).format('MM/DD')
                              : dayjs(record?.send_time).fromNow()}
                          </div>
                          {isTable && (
                            <>
                              {record?.flags?.includes('Flagged') ? (
                                <IconStarSelect data-no-click className='cursor-pointer text-xl!' />
                              ) : (
                                <IconStarUnselect data-no-click className='cursor-pointer text-xl!' />
                              )}
                            </>
                          )}
                        </div>
                      </div>
                      {!isTable && (
                        <>
                          <div className={'truncate'}>{record?.subject || ''}</div>
                          <div className={'flex items-center justify-between gap-2'}>
                            <div className={'flex-1 truncate font-light text-gray-400'}>{record?.text || ''}</div>
                            {record?.flags?.includes('Flagged') && <IconStarSelect className='cursor-pointer text-xl!' />}
                          </div>
                        </>
                      )}
                    </div>
                  ),
                },
              ]}
              data={mailList}
            />
          </Layout.Sider>

          {/* 右列：邮件详情 + 顶部操作按钮栏 */}
          <Layout.Content className={`relative h-full min-w-130 flex-1 bg-white ${isTable && currentMail ? ' z-10 w-full' : ''}`}>
            {currentMail && (
              <Spin block loading={currentLoading}>
                {/* 邮件操作工具栏 */}
                <div className='flex items-center justify-between gap-2 border-b border-gray-200 p-4'>
                  <div className='flex items-center gap-2'>
                    {isTable && currentMail && (
                      <Button size='small' icon={<IconArrowLeft />} onClick={() => setCurrentMail()}>
                        返回
                      </Button>
                    )}
                    <Button size='small' icon={<IconDelete />} onClick={() => onDelMail([currentMail.uid])}>
                      {currentFolder.key === 'delete' ? '彻底删除' : '删除'}
                    </Button>
                    <Button size='small' icon={<IconReply />} onClick={onReply}>
                      回复
                    </Button>
                    <Button size='small' icon={<IconRedo />} onClick={onForward}>
                      转发
                    </Button>
                    <Dropdown
                      trigger='click'
                      droplist={
                        <Menu onClickMenuItem={confirmMoveMail}>
                          {folderList
                            .filter((e) => ![currentFolder.folder].includes(e.folder))
                            .map((e) => (
                              <Menu.Item key={e.folder}>{e.title}</Menu.Item>
                            ))}
                        </Menu>
                      }>
                      <Button size='small'>
                        移动
                        <IconDown />
                      </Button>
                    </Dropdown>
                  </div>

                  {isTable && (
                    <Button.Group type='text'>
                      <Button
                        size='small'
                        icon={<IconLeft />}
                        disabled={currentMail?.uid === mailList[0]?.uid}
                        onClick={() => onCutMail(currentMail, 'prev')}>
                        上一封
                      </Button>
                      <Button
                        size='small'
                        disabled={currentMail?.uid === mailList[mailList?.length - 1]?.uid}
                        onClick={() => onCutMail(currentMail, 'next')}>
                        下一封
                        <IconRight />
                      </Button>
                    </Button.Group>
                  )}
                </div>
                <div className='h-[calc(100vh-117px)] flex-1 overflow-y-auto p-4'>
                  {/* 邮件头部信息 */}
                  <div className='mb-4 flex items-center gap-2'>
                    <span className='text-lg font-bold'>{currentMail.subject}</span>
                    {currentMail?.flags?.includes('Flagged') ? (
                      <IconStarSelect className='cursor-pointer text-xl!' onClick={() => onStar(currentMail)} />
                    ) : (
                      <IconStarUnselect className='cursor-pointer text-xl!' onClick={() => onStar(currentMail)} />
                    )}
                  </div>
                  <div className='mb-4 flex items-start gap-3'>
                    <Avatar className={'min-w-10!'} style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
                      {currentMail?.from_info?.name?.slice(0, 1).toUpperCase()}
                    </Avatar>
                    <div className='flex-1 text-sm'>
                      <div className='mb-1'>
                        <strong>{currentMail?.from_info?.name}</strong>
                        <span className='text-gray-400'>&nbsp;&lt;{currentMail.from}&gt;</span>
                      </div>
                      <div className='flex items-start justify-between gap-2'>
                        <div className='flex-1'>
                          <div className='mb-1 flex'>
                            <div className='whitespace-nowrap text-gray-400'>收件人</div>
                            <div className='flex flex-wrap'>
                              {currentMail?.to_info?.map((e, index) => (
                                <div key={index}>
                                  <span className='mr-1 ml-3'>{e.name}</span>
                                  <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                                  {index !== currentMail?.to_info?.length - 1 && <span className='text-gray-400'>,</span>}
                                </div>
                              ))}
                            </div>
                          </div>
                          {currentMail?.cc && (
                            <div className='flex items-center'>
                              <div className='text-gray-400'>抄送</div>
                              <div className='flex flex-wrap'>
                                {currentMail?.cc_info?.map((e, index) => (
                                  <div key={index}>
                                    <span className='mr-1 ml-3'>{e.name}</span>
                                    <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                                    {index !== currentMail?.to_info?.length - 1 && <span className='text-gray-400'>,</span>}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                        <div className='w-45 text-right text-gray-400'>
                          {dayjs(currentMail?.send_time).format('YYYY年MM月DD日 HH:mm:ss') || ''}
                        </div>
                      </div>
                    </div>
                  </div>
                  <Divider />
                  {/* 邮件正文 */}
                  <div
                    className='mail-detail'
                    dangerouslySetInnerHTML={{
                      __html: currentMail.detail?.content || '<div class="text-gray-500">暂无邮件内容</div>',
                    }}
                  />

                  {/* 附件 */}
                  {currentMail?.has_attach && (
                    <Card
                      className='mt-10'
                      title={
                        <>
                          <IconAttachment className='mr-1' />
                          {currentMail?.detail?.attachments?.length}个 附件 {currentMail?.detail?.attach_size}
                        </>
                      }>
                      <div className='flex flex-col gap-2'>
                        {currentMail.detail.attachments.map((item, index) => (
                          <div key={index} className='flex items-center justify-between gap-2 bg-gray-100 p-2 hover:bg-gray-200'>
                            <div className='flex flex-1 items-center'>
                              <span className='mr-2'>
                                {item?.file_type === 'video' && <IconVideo />}
                                {item?.file_type === 'audio' && <IconAudio />}
                                {item?.file_type === 'zip' && <IconZip />}
                                {item?.file_type === 'image' && <IconImage />}

                                {item?.file_type === 'ppt' && <IconPpt />}
                                {item?.file_type === 'pdf' && <IconPdf />}
                                {item?.file_type === 'excel' && <IconExcel />}
                                {item?.file_type === 'word' && <IconWord />}

                                {item?.file_type === 'text' && <IconText />}
                              </span>
                              {item.file_name}
                              <span className='text-gray-400'>（{item.size}）</span>
                            </div>
                            <Space>
                              <Button type='text' size='small' onClick={() => onPreviewAttachment(item)}>
                                <IconEye />
                                预览
                              </Button>
                              <Button type='text' size='small' onClick={() => onDownloadAttachment(item)}>
                                <IconToBottom />
                                下载
                              </Button>
                            </Space>
                          </div>
                        ))}
                      </div>
                    </Card>
                  )}
                </div>
              </Spin>
            )}
            {!currentMail && (
              <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
            )}
          </Layout.Content>
        </Layout>
      )}
    </Layout>
  )
}

export default MailLayout

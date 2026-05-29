import { useEffect, useState } from 'react'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

import { Avatar, Button, Card, Divider, Dropdown, Input, Layout, Menu, Message, Space, Spin, Table } from '@arco-design/web-react'
import {
  IconAttachment,
  IconClose,
  IconDelete,
  IconDown,
  IconEdit,
  IconEmail,
  IconFile,
  IconImage,
  IconRedo,
  IconReply,
  IconSend,
} from '@arco-design/web-react/icon'

import request from 'src/api/request'

import WriteMail from 'src/components/WriteMail'

import IconMailOpen from 'src/assets/mail-open.svg'
import IconMail from 'src/assets/mail.svg'
import IconSent from 'src/assets/sent.svg'

const { Sider, Content } = Layout

// 左侧文件夹
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

const MailLayout = () => {
  const [userList, setUserList] = useState({})
  const [folderList, setFolderList] = useState(menuList)
  const [currentFolder, setCurrentFolder] = useState({})
  const [searchWord, setSearchWord] = useState('')
  const [loading, setLoading] = useState(false)
  const [mailList, setMailList] = useState([])
  const [total, setTotal] = useState(0)
  const [currentLoading, setCurrentLoading] = useState(false)
  const [currentMail, setCurrentMail] = useState(null)
  const [writeMail, setWriteMail] = useState(null)
  const [newWriteMail, setNewWriteMail] = useState(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState([])

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
    // 草稿页已打开
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      setCurrentFolder(folderList.find((item) => item.key === 'compose'))
      return Message.warning('写邮件页已打开，请先关闭')
    }

    let composeItem = { key: 'compose', folder: 'DRAFTS', title: '草稿', icon: <IconFile /> }
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
  }

  // 选中邮件查看详情
  const handleSelectMail = async (item, e) => {
    // 排除干扰点击
    const targetElement = e?.target

    const isCheckboxClick = targetElement
      ? targetElement?.classList.contains('arco-checkbox') ||
        targetElement?.classList.contains('arco-checkbox-input') ||
        targetElement?.closest('.arco-checkbox')
      : false

    // 排除复选框的点击
    if (isCheckboxClick) return

    setCurrentMail()
    setCurrentLoading(true)
    const params = {
      uid: item.uid,
      folder: currentFolder.folder,
    }

    const { code, data, msg } = await request.post('/api/mail/detail', params)
    if (code === 200) {
      setCurrentMail({ ...item, detail: data })
      if (currentFolder.key === 'drafts') {
        const newItem = { ...item, detail: data, to_email: item?.to.split(', '), cc_email: item?.cc ? item?.cc.split(', ') : [] }
        onWriteMail('rewrite', newItem)
      }
    } else {
      Message.error(msg)
    }
    setCurrentLoading(false)
  }

  // 搜索邮件
  const handleSearch = async (val) => {
    if (!val) {
      Message.warning('请输入搜索内容')
      return
    }

    setCurrentMail(null)
    setSelectedRowKeys([])

    getMailData(currentFolder.folder, val)
  }

  // 获取邮件数据
  const getMailData = async (folder, keyword = '') => {
    // 加载邮件列表
    setLoading(true)
    const params = { page: 1, size: 1000, folder, keyword }
    let { code, data, msg } = await request.post('/api/mail/list', params)
    if (code === 200) {
      const list = (data?.list || []).map((e) => {
        const from_name = e.from.split('@')[0]
        const to_info = e.to.split(', ').map((t) => t && { email: t, name: t.split('@')[0] })
        const to_name = to_info.map((t) => t.name).join(', ')
        const to_reply = to_info.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ')

        const cc_info = e.cc ? e.cc.split(', ').map((t) => t && { email: t, name: t.split('@')[0] }) : []
        const cc_name = cc_info ? cc_info.map((t) => t.name).join(', ') : ''
        const cc_reply = cc_info ? cc_info?.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ') : ''

        return {
          ...e,
          from_name,
          to_info,
          to_name,
          to_reply,
          cc_info,
          cc_name,
          cc_reply,
        }
      })
      setMailList(list)
      setTotal(data?.total || 0)
    } else {
      Message.error(msg)
    }
    setLoading(false)
  }

  // 下载附件
  const handleDownloadAttachment = async (item) => {
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

  // 删除邮件
  const handleDelMail = async (ids) => {
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
  <p style="line-height: 1;"><span style="font-size: 13px;">原始邮件</span>——————</p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 13px;">发件人：</span>
  <span style="font-size: 13px;">${currentMail?.from_name} &lt;${currentMail?.from}&gt; </span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>发件时间：</span>
  <span style="font-size: 13px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>收件人：</span>
  <span style="font-size: 13px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>主题：</span>
  <span style="font-size: 13px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  const FormContentCc = `<p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><span style="font-size: 13px;">原始邮件</span>——————</p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 13px;">发件人：</span>
  <span style="font-size: 13px;">${currentMail?.from_name} &lt;${currentMail?.from}&gt;</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>发件时间：</span>
  <span style="font-size: 13px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>收件人：</span>
  <span style="font-size: 13px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>抄送：</span>
  <span style="font-size: 13px;">${currentMail?.cc_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 13px;"><br>主题：</span>
  <span style="font-size: 13px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  // 回复邮件
  const handleReply = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `回复: ${currentMail.subject}`,
      to_email: currentMail.to.split(', '),
      cc: currentMail.cc ? currentMail.cc.split(', ') : [],
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
    }
    if (currentFolder.key === 'inbox') {
      newMail.to_email = [currentMail.from]
    }
    onWriteMail('reply', newMail)
  }

  // 转发邮件
  const handleForward = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `转发: ${currentMail.subject}`,
      to_email: null,
      cc_email: null,
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
    }

    onWriteMail('forward', newMail)
  }

  // 标记已读
  const onRead = async (item) => {
    if (item?.is_read) return
    const params = {
      uid: item.uid,
      folder: currentFolder.folder,
      status: '\\Seen',
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      setMailList((prev) => {
        const newList = [...prev]
        const index = newList.findIndex((item) => item.uid === params.uid)
        newList[index].is_read = true
        return newList
      })
    }
  }

  // 发送邮件&草稿
  const handleSend = async (type, form, html, fileList, detail, setLoading) => {
    const values = form.getFieldsValue()
    if (!values.to || !values.subject) {
      Message.warning('请填写收件人和主题')
      return
    }
    const formData = new FormData()
    formData.append('to', values.to)
    formData.append('cc', values.cc || '')
    formData.append('subject', values.subject)
    formData.append('content', html)
    if (detail?.uid) {
      formData.append('uid', detail.uid)
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
      onClickCompose(type === 'Sent' ? 'inbox' : 'drafts')
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
  }

  // 初始加载邮件列表
  useEffect(() => {
    loadMailList('inbox')
    getUserList()
  }, [])

  return (
    <Layout className='flex-1'>
      {/* 左列：文件夹导航 */}
      <Sider width={220} theme='light' className='mail-menu border-r border-gray-200 bg-white'>
        <div className='p-4'>
          <Button type='primary' icon={<IconEdit />} long onClick={() => onWriteMail('new')}>
            写信
          </Button>
        </div>
        <Menu selectedKeys={[currentFolder?.key || '']} onClickMenuItem={(key) => loadMailList(key)}>
          {folderList.map((item) => (
            <Menu.Item key={item.key}>
              <span>
                {item.icon}
                <span className='inline-block w-27 overflow-hidden align-middle text-ellipsis whitespace-nowrap'>
                  {item.title}
                </span>
              </span>
              {item?.key === 'compose' && <IconClose className='m-0!' onClick={() => onClickCompose('inbox')} />}
            </Menu.Item>
          ))}
        </Menu>
      </Sider>

      {currentFolder?.key === 'compose' ? (
        <Spin className={'w-full'} block loading={currentLoading}>
          <WriteMail
            key={writeMail?.uid || '0'}
            detail={writeMail}
            userList={userList?.list || []}
            onChange={setNewWriteMail}
            onClose={onClickCompose}
            onSend={handleSend} // 传递邮件发送函数
          />
        </Spin>
      ) : (
        <>
          {/* 中列：邮件列表 */}
          <div className='max-w-90 min-w-90 flex-1 border-r border-gray-200'>
            {/* 搜索框 */}
            <div className='border-b border-gray-200 p-2'>
              <Input.Search
                placeholder='搜索主题/发件人'
                searchButton
                value={searchWord}
                onChange={setSearchWord}
                onSearch={handleSearch}
              />
            </div>
            <Table
              loading={loading}
              scroll={{ y: 'calc(100vh - 157px)' }}
              className='email-list h-full'
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
                    handleSelectMail(record, e)
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
                          <Button size='mini' icon={<IconDelete />} onClick={() => handleDelMail(selectedRowKeys)}>
                            {currentFolder.key === 'delete' ? '清空' : '删除'}
                          </Button>
                        )}
                        <span>共 {total} 封</span>
                      </Space>
                    </div>
                  ),
                  dataIndex: 'date',
                  render: (text, record) => (
                    <div className={record.is_read ? '' : 'font-bold'} onClick={() => onRead(record)}>
                      <div className='mb-1 flex items-center justify-between gap-2'>
                        <div className={'flex items-center gap-1.5'}>
                          {record.is_read ? <IconMailOpen /> : <IconMail />}
                          {currentFolder?.key === 'sent' ? (
                            <>
                              <IconSent />
                              {record?.to_name}
                              {record?.cc_name ? ', ' + record?.cc_name : ''}
                            </>
                          ) : (
                            record?.from_name
                          )}

                          {record.has_attach ? <IconAttachment className='text-gray-400!' /> : ''}
                        </div>
                        <span>{dayjs(record?.send_time).fromNow()}</span>
                      </div>
                      <div className={'truncate'}>{record?.subject || ''}</div>
                      <div className={'truncate font-light text-gray-400'}>{record?.text || ''}</div>
                    </div>
                  ),
                },
              ]}
              data={mailList}
            />
          </div>

          {/* 右列：邮件详情 + 顶部操作按钮栏 */}
          <Content className='min-w-130 flex-1'>
            {currentMail ? (
              <>
                {/* 邮件操作工具栏 */}
                <div className='flex items-center gap-2 border-b border-gray-200 p-4'>
                  <Button size='small' icon={<IconDelete />} onClick={() => handleDelMail([currentMail.uid])}>
                    {currentFolder.key === 'delete' ? '彻底删除' : '删除'}
                  </Button>
                  <Button size='small' icon={<IconReply />} onClick={handleReply}>
                    回复
                  </Button>
                  <Button size='small' icon={<IconRedo />} onClick={handleForward}>
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
                <Spin className='h-[calc(100vh-180px)] flex-1 overflow-y-auto p-4' block loading={currentLoading}>
                  {/* 邮件头部信息 */}
                  <div className='mb-4 text-lg font-bold'>{currentMail.subject}</div>
                  <div className='mb-4 flex items-start gap-3'>
                    <Avatar className={'min-w-10!'} style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
                      {currentMail?.from_name?.slice(0, 1).toUpperCase()}
                    </Avatar>
                    <div className='flex-1 text-sm'>
                      <div className='mb-1'>
                        <strong>{currentMail.from_name}</strong>
                        <span className='text-gray-400'>&nbsp;&lt;{currentMail.from}&gt;</span>
                      </div>
                      <div className='flex items-center justify-between gap-2'>
                        <div className='flex-1 whitespace-nowrap'>
                          <div className='mb-1 flex items-center'>
                            <div className='text-gray-400'>收件人</div>
                            {currentMail?.to_info?.map((e, index) => (
                              <div key={index}>
                                {index !== 0 && <span className='text-gray-400'>,</span>}
                                <span className='mr-1 ml-3'>{e.name}</span>
                                <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                              </div>
                            ))}
                          </div>
                          {currentMail?.cc && (
                            <div className='flex items-center'>
                              <div className='text-gray-400'>抄送</div>
                              {currentMail?.cc_info?.map((e, index) => (
                                <div key={index}>
                                  {index !== 0 && <span className='text-gray-400'>,</span>}
                                  <span className='mr-1 ml-3'>{e.name}</span>
                                  <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                        <div className='w-45 text-right text-gray-400'>
                          {dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss') || ''}
                        </div>
                      </div>
                    </div>
                  </div>
                  <Divider />

                  {/* 邮件正文 */}
                  <div className="mail-detail"
                    dangerouslySetInnerHTML={{
                      __html: currentMail.detail?.content || '<div class="text-gray-500">暂无邮件内容</div>',
                    }}
                  />
                  {currentMail.detail?.attachments?.length > 0 && (
                    <Card
                      className='mt-10'
                      title={
                        <>
                          <IconAttachment className='mr-1' />
                          {currentMail?.detail?.attachments?.length}个 附件
                        </>
                      }>
                      <div className='flex flex-col gap-2'>
                        {currentMail.detail.attachments.map((item, index) => (
                          <div key={index} className='flex items-center justify-between gap-2 bg-gray-100 p-2'>
                            <div className='flex-1'>
                              <Avatar
                                size={28}
                                shape='square'
                                className={`mr-2 ${item?.content_type?.includes('image') ? 'bg-[#0BB5B5]!' : 'bg-[#5252CC]!'}`}>
                                {item?.content_type?.includes('image') ? (
                                  <IconImage className={'text-xl'} />
                                ) : (
                                  <IconFile className={'text-xl'} />
                                )}
                              </Avatar>
                              {item.file_name}
                            </div>
                            <Button type='text' size='small' onClick={() => handleDownloadAttachment(item)}>
                              下载
                            </Button>
                          </div>
                        ))}
                      </div>
                    </Card>
                  )}
                </Spin>
              </>
            ) : (
              <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
            )}
          </Content>
        </>
      )}
    </Layout>
  )
}

export default MailLayout

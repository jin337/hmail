import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

import WriteMail from './WriteMail'

import IconMailOpen from 'src/assets/mail-open.svg'
import IconMail from 'src/assets/mail.svg'
import IconSent from 'src/assets/sent.svg'

import {
  Avatar,
  Button,
  Divider,
  Dropdown,
  Form,
  Input,
  Layout,
  List,
  Menu,
  Message,
  Modal,
  Space,
  Spin,
  Table,
} from '@arco-design/web-react'
import {
  IconAttachment,
  IconClose,
  IconDelete,
  IconDown,
  IconEdit,
  IconEmail,
  IconFile,
  IconRedo,
  IconReply,
  IconSend,
  IconSettings,
} from '@arco-design/web-react/icon'

import request from 'src/api/request'

const { Sider, Content } = Layout

// 左侧文件夹
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> },
  { key: 'trash', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

const MailApp = () => {
  const [formPwd] = Form.useForm()
  const navigate = useNavigate()
  const [folderList, setFolderList] = useState(menuList)
  const [currentFolder, setCurrentFolder] = useState({})
  const [mailList, setMailList] = useState([])
  const [currentMail, setCurrentMail] = useState(null)
  const [writeMail, setWriteMail] = useState(null)
  const [currentLoading, setCurrentLoading] = useState(false)
  const [loading, setLoading] = useState(false)
  const [searchWord, setSearchWord] = useState('')

  const [selectedRowKeys, setSelectedRowKeys] = useState([])
  const [visible, setVisible] = useState(false)

  // 本地登录信息
  const userEmail = localStorage.getItem('mail_email')

  // 写信也没是否打开
  const isOpenWriteMail = () => {
    if (folderList.some((item) => item.key === 'compose')) {
      return Message.warning('写邮件页已打开，请先关闭')
    }
  }

  // 关闭写邮件页，返回收件箱
  const onClickCompose = (key) => {
    setWriteMail(null)
    setFolderList((prev) => prev.filter((item) => item.key !== 'compose'))
    setTimeout(() => {
      const item = folderList.find((item) => item.key === key)
      loadMailList(item.key)
    }, 300)
  }
  // 写邮件
  const onWriteMail = (key, mailData) => {
    isOpenWriteMail()

    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      setCurrentFolder(folderList.find((item) => item.key === 'compose'))
      return
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

  // 加载邮件列表
  const loadMailList = async (key) => {
    setSelectedRowKeys([])
    setMailList([])
    setCurrentMail(null)
    setSearchWord('')

    const item = folderList.find((item) => item.key === key)
    setCurrentFolder(item)

    if (key === 'compose') {
      return
    }
    setLoading(true)
    const params = { folder: item.folder, page: 1, size: 20, keyword: '' }
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
    } else {
      Message.error(msg)
    }
    setLoading(false)
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
        const newItem = { ...item, detail: data, to_email: item?.to.split(', '), cc_email: item?.cc.split(', ') }
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

    setLoading(true)
    const params = { folder: currentFolder.folder, page: 1, size: 20, keyword: val }
    const { code, data, msg } = await request.post('/api/mail/list', params)
    if (code === 200) {
      setMailList(data?.list || [])
      if (data.length === 0) Message.info('无搜索结果')
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
  const FormContent = `<p><br></p><p><br></p><p>原始邮件——————</p><pre><code>发件人：${currentMail?.from_name} &lt;${currentMail?.from}&gt;
日期：${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}
收件人：${currentMail?.to_reply}
${currentMail?.cc && `抄送：${currentMail?.cc_reply}`}
主题：${currentMail?.subject}</code></pre>${currentMail?.detail?.content || ''}`
  // 回复邮件
  const handleReply = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `回复: ${currentMail.subject}`,
      to_email: currentMail.to.split(', '),
      cc_email: currentMail.cc ? currentMail.cc.split(', ') : [],
    }
    if (currentFolder.key === 'inbox') {
      newMail.to_email = [currentMail.from]
    }
    newMail.detail.content = FormContent

    onWriteMail('reply', newMail)
  }
  // 转发邮件
  const handleForward = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `转发: ${currentMail.subject}`,
    }

    newMail.detail.content = FormContent

    onWriteMail('forward', { ...newMail, to_email: [] })
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

  // 修改密码
  const handleRepasword = () => {
    setVisible(true)
    formPwd.resetFields()
  }

  // 修改密码
  const submitPwd = () => {
    formPwd.validate().then(async (values) => {
      const params = {
        old_password: values.oldpwd,
        new_password: values.newpwd,
      }
      const { code, message } = await request.post('/api/mail/chgpwd', params)
      if (code === 200) {
        Message.success('密码修改成功，请重新登录')
        handleLogout()
      } else {
        Message.error(message)
      }
    })
    setVisible(false)
  }
  // 自定义密码验证规则
  const checkPasswordRules = (value, callback) => {
    if (!value) {
      return callback('请输入密码')
    }

    if (value.length < 6 || value.length > 20) {
      return callback('密码长度必须在6到20个字符之间')
    }

    if (value.includes('^')) {
      return callback('密码不能包含^字符')
    }

    return Promise.resolve()
  }
  const checkNewPassword = (value, callback) => {
    if (!value) {
      return callback('请输入新密码')
    }

    if (value.length < 6 || value.length > 20) {
      return callback('密码长度必须在6到20个字符之间')
    }

    if (value.includes('^')) {
      return callback('密码不能包含^字符')
    }

    const oldPassword = formPwd.getFieldValue('oldpwd')
    if (oldPassword && value === oldPassword) {
      return callback('新密码不能与旧密码相同')
    }

    return Promise.resolve()
  }
  const checkConfirmPassword = (value, callback) => {
    if (!value) {
      return callback('请确认新密码')
    }

    const newPassword = formPwd.getFieldValue('newpwd')
    if (newPassword && value !== newPassword) {
      return callback('两次输入的密码不一致')
    }

    return Promise.resolve()
  }
  // 退出
  const handleLogout = () => {
    localStorage.removeItem('mail_token')
    localStorage.removeItem('mail_email')
    navigate('/login')
  }

  // 初始加载邮件列表，如果没有登录信息则跳转到登录页
  useEffect(() => {
    if (!userEmail) {
      navigate('/login')
      return
    } else {
      loadMailList('inbox')
    }
  }, [userEmail])

  return (
    <Layout className='h-screen w-full overflow-hidden'>
      <Layout.Header className='z-10 flex h-14 items-center justify-between bg-white px-6 shadow-sm'>
        {/* Logo */}
        <div className='flex items-center gap-3'>
          <div className='flex h-8 w-8 items-center justify-center rounded bg-blue-500 text-lg font-bold text-white'>H</div>
          <span className='text-lg font-bold text-gray-700'>华盛邮件</span>
        </div>
        {/* 搜索框 */}
        <div className='mx-8 flex max-w-xl flex-1'>
          <Input.Search
            placeholder='搜索主题/发件人'
            searchButton
            value={searchWord}
            onChange={setSearchWord}
            onSearch={handleSearch}
          />
        </div>
        {/* 头像和退出登录按钮 */}
        <Space>
          {userEmail}
          <Dropdown
            position='br'
            droplist={
              <Menu>
                <Menu.Item key='1' onClick={handleRepasword}>
                  修改密码
                </Menu.Item>
                <Menu.Item key='2' onClick={handleLogout}>
                  退出登录
                </Menu.Item>
              </Menu>
            }>
            <Button size='small' type='text'>
              <IconSettings />
            </Button>
          </Dropdown>
        </Space>
      </Layout.Header>

      <Layout>
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
            <WriteMail detail={writeMail} onClose={onClickCompose} />
          </Spin>
        ) : (
          <>
            {/* 中列：邮件列表 */}
            <div className='max-w-100 flex-1 border-r border-gray-200'>
              <Table
                loading={loading}
                scroll={{ y: 'calc(100vh - 108px)' }}
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
                              删除
                            </Button>
                          )}
                          <span>共 {mailList?.length || 0} 封</span>
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

                            {record.has_attach ? <IconAttachment className='ml-1 text-gray-400!' /> : ''}
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
            <Content>
              {currentMail ? (
                <>
                  {/* 邮件操作工具栏 */}
                  <div className='flex items-center gap-2 border-b border-gray-200 p-4'>
                    <Button size='small' icon={<IconDelete />} onClick={() => handleDelMail([currentMail.uid])}>
                      {currentFolder.key === 'trash' ? '彻底删除' : '删除'}
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
                  <Spin className='h-[calc(100vh-120px)] flex-1 overflow-y-auto p-4' block loading={currentLoading}>
                    {/* 邮件头部信息 */}
                    <div className='mb-4 text-lg font-bold'>{currentMail.subject}</div>
                    <div className='mb-4 flex items-start gap-3'>
                      <Avatar style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
                        {currentMail?.from_name?.slice(0, 1).toUpperCase()}
                      </Avatar>
                      <div className='flex-1 text-sm'>
                        <div className='mb-1'>
                          <strong>{currentMail.from_name}</strong>
                          <span className='text-gray-400'>&nbsp;&lt;{currentMail.from}&gt;</span>
                        </div>
                        <div className='flex items-center justify-between gap-2'>
                          <div className='flex-1'>
                            <div className='mb-1 flex items-center'>
                              <div className='text-gray-400'>收件人</div>
                              {currentMail?.to_info.map((e, index) => (
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
                                {currentMail?.cc_info.map((e, index) => (
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
                    {currentMail.detail?.attachments?.length > 0 && (
                      <>
                        <div style={{ marginBottom: 16 }}>
                          <div style={{ fontWeight: 500, marginBottom: 8 }}>附件：</div>
                          <List
                            size='small'
                            dataSource={currentMail.detail.attachments}
                            render={(item, index) => (
                              <List.Item
                                key={index}
                                extra={
                                  <Button type='text' size='small' onClick={() => handleDownloadAttachment(item)}>
                                    下载
                                  </Button>
                                }>
                                <List.Item.Meta title={item.file_name} />
                              </List.Item>
                            )}
                          />
                        </div>
                        <Divider />
                      </>
                    )}
                    <div
                      className='mail-detail'
                      style={{ lineHeight: 1.8 }}
                      dangerouslySetInnerHTML={{
                        __html: currentMail.detail?.content || '<div class="text-gray-500">暂无邮件内容</div>',
                      }}
                    />
                  </Spin>
                </>
              ) : (
                <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
              )}
            </Content>
          </>
        )}
      </Layout>

      {/* 修改密码 */}
      <Modal
        title='修改密码'
        visible={visible}
        onOk={() => submitPwd()}
        onCancel={() => setVisible(false)}
        autoFocus={false}
        focusLock={true}>
        <Form
          form={formPwd}
          autoComplete='off'
          layout='vertical'
          validateMessages={{ required: (_, { label }) => `${label}是必填项` }}>
          <Form.Item rules={[{ required: true }, { validator: checkPasswordRules }]} label='旧密码' field='oldpwd'>
            <Input.Password placeholder='请输入...' />
          </Form.Item>
          <Form.Item rules={[{ required: true }, { validator: checkNewPassword }]} label='新密码' field='newpwd'>
            <Input.Password placeholder='请输入...' />
          </Form.Item>
          <Form.Item rules={[{ required: true }, { validator: checkConfirmPassword }]} label='确认新密码' field='confirmpwd'>
            <Input.Password placeholder='请输入...' />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}

export default MailApp

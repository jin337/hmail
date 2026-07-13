import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import {
  Avatar,
  Button,
  Card,
  DatePicker,
  Dropdown,
  Form,
  Input,
  InputTag,
  Layout,
  Menu,
  Modal,
  Popover,
  Space,
  TimePicker,
  Tree,
  Typography,
  Upload,
} from '@arco-design/web-react'
import {
  IconCalendar,
  IconClose,
  IconDelete,
  IconDown,
  IconEdit,
  IconFile,
  IconPlus,
  IconSend,
  IconSettings,
  IconUpload,
} from '@arco-design/web-react/icon'

// 引入 wangEditor
import { Editor, Toolbar } from '@wangeditor/editor-for-react'

// 时间处理
import dayjs from 'dayjs'

// 公共事件
import { isSvg } from 'src/utils'

export default function WriteMail({
  detail,
  userList = [],
  recentlyList = [],
  onClose,
  onChange,
  onSend,
  onEditRecently,
  onDeleteRecently,
  onClearRecently,
}) {
  const separator = '#_#'
  const [form] = Form.useForm()
  const [formContact] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const [addCC, setAddCC] = useState(false)

  const [fileList, setFileList] = useState([])
  const [editor, setEditor] = useState(null)
  const [html, setHtml] = useState('')

  const [customTimeVisible, setCustomTimeVisible] = useState(false)
  const [customTime, setCustomTime] = useState(null)

  const toRef = useRef(null)
  const ccRef = useRef(null)
  const [lastFocus, setLastFocus] = useState(null) // 缓存最后一次焦点

  const timeList = [
    {
      key: '9',
      title: '明天上午9:00发送',
    },
    {
      key: 'custom',
      title: '自定义时间发送...',
    },
  ]

  // 格式化时间
  const onFormatTime = () => {
    setCustomTime(customTimeVisible)
    setCustomTimeVisible(false)
  }
  // 时间选择
  const onTimeCustom = (e) => {
    if (!customTime) {
      let target = dayjs().add(1, 'day').hour(9).minute(0).second(0)
      if (e === 'custom') {
        target = dayjs()
        setCustomTimeVisible({
          date: target.format('YYYY-MM-DD ddd'),
          time: target.format('HH:mm'),
        })
      } else {
        setCustomTime({
          date: target.format('YYYY-MM-DD ddd'),
          time: target.format('HH:mm'),
        })
      }
    } else {
      setCustomTimeVisible({ ...customTime })
    }
  }

  // 选择用户
  const onSelectUser = (ids, extra) => {
    const { e } = extra
    // 忽略svg点击
    if (isSvg(e)) {
      e.preventDefault()
      e.stopPropagation()
      return
    }

    const key = ids[0]?.split(separator).pop()
    const item = [...recentlyList, ...userList].find((e) => e?.email === key)
    if (!item) return

    const targetEmail = {
      label: item?.full_name,
      value: item?.email,
    }

    // 优先使用缓存的最后焦点
    if (lastFocus === 'to_info') {
      toRef.current.focus()
    }
    if (lastFocus === 'cc_info') {
      ccRef.current.focus()
    }
    if (!lastFocus) return

    const valueslist = form.getFieldValue(lastFocus) || []
    const list = valueslist.filter((e) => e.value !== targetEmail.value)
    if (!list.includes(targetEmail)) {
      form.setFieldValue(lastFocus, [...list, targetEmail])
    }
  }

  // 打开CC
  const openCC = () => {
    setAddCC(true)
    // 延迟聚焦，等待DOM渲染完成
    setTimeout(() => {
      ccRef.current?.focus()
    }, 0)
  }

  // 自动回填
  useEffect(() => {
    if (!detail?.uid) return

    const init = async () => {
      form.setFieldsValue(detail)
      setAddCC(detail?.cc_email?.length > 0)

      const list = (detail?.detail?.attachments || []).map((e) => ({
        ...e,
        name: e.file_name,
        uid: e.part_id,
      }))
      setFileList(list)

      if (editor) {
        editor?.setHtml(`${detail?.detail?.content || ''}`)
        editor?.focus() //获取焦点
      }
    }
    init()
  }, [detail, form, editor])

  // 销毁编辑器
  useEffect(() => {
    return () => {
      if (editor) editor.destroy()
    }
  }, [])

  // 写信，默认获取焦点
  useEffect(() => {
    if (!detail?.uid) {
      toRef.current.focus() //获取焦点
    }
  }, [detail, toRef, ccRef])

  // 提取验证函数
  const validateEmails = (value, callback) => {
    const EMAIL_REGEX = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

    if (!Array.isArray(value) || value.length === 0) {
      return callback()
    }

    for (const item of value) {
      const emailStr = item.value || ''
      // 标准邮箱格式
      if (/\s/.test(emailStr) || !EMAIL_REGEX.test(emailStr)) {
        return callback(`【${emailStr}】邮箱格式不正确`)
      }
    }

    callback()
  }

  // 发送邮件&草稿
  const handleSend = async (type) => {
    // 调用父组件传递的发送函数
    if (onSend) {
      let time = ''
      if (customTime) {
        let str = customTime?.date + ' ' + customTime?.time
        const pureTime = str.replace(/\s.+?(\d{2}:\d{2})$/, ' $1')
        time = dayjs(pureTime, 'YYYY-MM-DD HH:mm').format('YYYY-MM-DD HH:mm:ss')
      }

      onSend(type, form, html, fileList, detail, time, setLoading)
    }
  }

  function getEmailPrefix(str) {
    if (!str) return str
    const emailReg = /^[\w.-]+@[\w.-]+\.\w+$/
    if (emailReg.test(str)) {
      return str.split('@')[0]
    }
    return str
  }

  // 监控数据变化
  const onChangeMail = (_, values) => {
    const newToInfo = values?.to_info?.map((e) => ({ ...e, label: getEmailPrefix(e.label) }))
    if (newToInfo !== values?.to_info) {
      form.setFieldValue('to_info', newToInfo)
    }

    const newCcInfo = values?.cc_info?.map((e) => ({ ...e, label: getEmailPrefix(e.label) }))
    if (newCcInfo !== values?.cc_info) {
      form.setFieldValue('cc_info', newCcInfo)
    }

    const newValues = {
      ...values,
      to_info: newToInfo,
      cc_info: newCcInfo,
      detail: {
        content: html || detail?.detail?.content,
        attachments: values?.files || fileList?.length > 0 ? fileList : detail?.detail?.attachments,
      },
    }

    onChange(newValues)
  }

  // 清空联系人
  const onClearContact = useCallback(() => {
    Modal.confirm({
      title: '提示',
      okText: '清空',
      className: 'simpleModal',
      content: '确定要清空最近联系人吗？',
      onOk: async () => {
        onClearRecently()
      },
    })
  }, [onClearRecently,])

  // 联系人数据
  const treeData = useMemo(() => {
    const baseNodes = [
      {
        full_name: '邮箱联系人',
        key: '0-1',
        selectable: false,
        children: userList?.map((e) => ({ ...e, key: '0-1' + separator + e.email })),
      },
    ]

    // 有最近联系人则前置插入分组
    if (recentlyList.length > 0) {
      baseNodes.unshift({
        full_name: (
          <div className='group flex justify-between leading-6'>
            <span>最近联系人</span>
            <Button type='text' status='danger' size='mini' className='hidden! group-hover:block!' onClick={onClearContact}>
              清空
            </Button>
          </div>
        ),
        key: '0-0',
        selectable: false,
        children: recentlyList?.map((e) => ({ ...e, key: '0-0' + separator + e.email })),
      })
    }
    return baseNodes
  }, [recentlyList, userList, onClearContact])

  // 编辑联系人
  const editContact = (e) => {
    Modal.confirm({
      title: '编辑联系人',
      okText: '保存',
      icon: null,
      content: (
        <Form autoComplete='off' form={formContact} initialValues={e}>
          <Form.Item label='昵称' field='name' required>
            <Input placeholder='昵称' />
          </Form.Item>
          <Form.Item label='邮箱' field='email' disabled>
            <Input placeholder='邮箱' />
          </Form.Item>
        </Form>
      ),
      onOk: async () => {
        const values = await formContact.validate()
        onEditRecently(values)
      },
    })
  }

  // 联系人-生成树节点
  const generatorTreeNodes = (data = []) => {
    if (!Array.isArray(data) || data.length === 0) return []

    return data?.map((item) => {
      const { children, ...rest } = item

      rest.title = children ? (
        item.full_name
      ) : (
        <Popover
          position='lb'
          trigger='hover'
          key={item.email}
          triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
          content={
            <div className='flex gap-2'>
              <Avatar className={'min-w-10!'} style={{ backgroundColor: '#FFEDD8', color: '#FF8800' }}>
                {item?.full_name?.slice(0, 1).toUpperCase()}
              </Avatar>
              <div>
                <div className='flex items-center gap-2 font-bold'>
                  {item?.full_name}
                  {!item.id && <IconEdit className='cursor-pointer' onClick={() => editContact(item)} />}
                </div>
                <Typography.Text copyable>{item?.email}</Typography.Text>
              </div>
            </div>
          }>
          <div className='group flex items-center justify-between gap-2 leading-6'>
            {item.full_name}
            {!item.id && (
              <Button
                type='text'
                status='danger'
                size='mini'
                className='hidden! group-hover:block!'
                onClick={() => onDeleteRecently(item)}>
                <IconDelete />
              </Button>
            )}
          </div>
        </Popover>
      )
      return (
        <Tree.Node {...rest} key={item?.key} dataRef={item}>
          {children ? generatorTreeNodes(item.children) : null}
        </Tree.Node>
      )
    })
  }
  const treeNodes = useMemo(() => {
    return generatorTreeNodes(treeData)
  }, [treeData, generatorTreeNodes])

  return (
    <Layout className='h-full'>
      <Layout.Header className='flex h-15 items-center justify-between border-b border-gray-300 px-6'>
        <Space>
          <Button type='primary' icon={<IconSend />} loading={loading} onClick={() => handleSend('Sent')}>
            发送邮件
          </Button>
          <Button
            type='secondary'
            icon={<IconFile />}
            disabled={customTime?.date}
            loading={loading}
            onClick={() => handleSend('Drafts')}>
            存草稿
          </Button>
          <Dropdown
            trigger='click'
            droplist={
              <Menu onClickMenuItem={onTimeCustom}>
                {timeList.map((e) => (
                  <Menu.Item key={e.key}>{e.title}</Menu.Item>
                ))}
              </Menu>
            }>
            <Button type='secondary' icon={<IconSettings />} loading={loading}>
              发信设置
              <IconDown />
            </Button>
          </Dropdown>
        </Space>
        <Space>
          {!addCC && (
            <Button type='text' icon={<IconPlus />} onClick={() => openCC()}>
              添加抄送
            </Button>
          )}
          <Button
            type='text'
            size='mini'
            onClick={() => onClose('inbox')}
            icon={<IconClose className='text-gray-500!' />}></Button>
        </Space>
      </Layout.Header>
      <Layout.Content>
        <div className='flex h-[calc(100vh-116px)] items-start'>
          <Form
            className='h-full flex-1 overflow-y-auto p-6 pb-0'
            form={form}
            autoComplete='off'
            layout='vertical'
            onChange={onChangeMail}>
            <Form.Item className={'text-[13px] text-gray-500'} hidden={!customTime?.time}>
              发送后，邮件将于
              <span className='mx-2 cursor-pointer text-blue-500' onClick={() => onTimeCustom('custom')}>
                {customTime?.date}&nbsp;
                {customTime?.time}
              </span>
              发出
              <Button
                status='danger'
                type='text'
                size='mini'
                className='ml-2!'
                onClick={() => {
                  setCustomTime(null)
                }}>
                <IconClose />
              </Button>
            </Form.Item>
            <Form.Item field='to_info' rules={[{ required: true, message: '请输入收件人' }, { validator: validateEmails }]}>
              <InputTag
                labelInValue
                ref={toRef}
                prefix='收件人'
                placeholder='test@xxx.com'
                maxTagCount={5}
                saveOnBlur
                onFocus={() => setLastFocus('to_info')}
              />
            </Form.Item>
            <Form.Item field='cc_info' hidden={!addCC} rules={[{ validator: validateEmails }]}>
              <InputTag
                labelInValue
                ref={ccRef}
                prefix='抄送'
                placeholder='test@xxx.com'
                maxTagCount={5}
                saveOnBlur
                onFocus={() => setLastFocus('cc_info')}
              />
            </Form.Item>
            <Form.Item field='subject' rules={[{ required: true, message: '请输入主题' }]}>
              <Input prefix='主题' placeholder='邮件主题' />
            </Form.Item>
            {/* 富文本编辑器 */}
            <Form.Item>
              <div className='z-100 overflow-hidden rounded border border-gray-300'>
                <Toolbar
                  editor={editor}
                  defaultConfig={{
                    excludeKeys: ['group-video', 'group-image', 'insertTable', 'codeBlock', 'group-more-style'],
                    insertKeys: {
                      index: 30,
                      keys: ['clearStyle'],
                    },
                  }}
                  mode='default'
                  className='border-b border-gray-300'
                />
                <Editor
                  className='h-80 overflow-y-auto'
                  defaultConfig={{ placeholder: '请输入邮件正文...' }}
                  onCreated={setEditor}
                  onChange={(editor) => {
                    setHtml(editor.getHtml())
                    onChangeMail(null, detail)
                  }}
                  mode='default'
                />
              </div>
            </Form.Item>
            <Form.Item field='files'>
              <Upload
                autoUpload={false}
                action='/'
                multiple
                tip='（上传附件不得超过20M）'
                showUploadList={{ startIcon: null }}
                fileList={fileList}
                onChange={setFileList}>
                <Button icon={<IconUpload />}>上传附件</Button>
              </Upload>
            </Form.Item>
          </Form>
          <Card title='联系人' className='h-full w-60 border-t-0!' bodyStyle={{ overflowY: 'auto', height: 'calc(100% - 50px)' }}>
            <Tree
              blockNode
              autoExpandParent
              className='mail-contacts h-full'
              key={treeData.map((item) => item.key).join(',')}
              onSelect={(e, extra) => onSelectUser(e, extra)}>
              {treeNodes}
            </Tree>
          </Card>
        </div>
      </Layout.Content>

      <Modal
        title='自定义发送时间'
        className={'w-110!'}
        visible={customTimeVisible}
        onCancel={() => setCustomTimeVisible(false)}
        onOk={onFormatTime}>
        <div className='mb-2 flex items-center gap-2'>
          <Input prefix={<IconCalendar />} value={customTimeVisible?.date} />
          <TimePicker
            format='HH:mm'
            className={'w-40'}
            value={customTimeVisible?.time}
            onChange={(e) => setCustomTimeVisible((prev) => ({ ...prev, time: e }))}
          />
        </div>
        <DatePicker
          className={'no-footer w-69'}
          format='YYYY-MM-DD ddd'
          triggerElement={null}
          onChange={(_, date) => setCustomTimeVisible((prev) => ({ ...prev, date: date.format('YYYY-MM-DD ddd') }))}
        />
      </Modal>
    </Layout>
  )
}

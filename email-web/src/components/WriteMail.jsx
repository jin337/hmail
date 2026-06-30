import { useEffect, useRef, useState } from 'react'

import {
  Avatar,
  Button,
  Card,
  Form,
  Input,
  InputTag,
  Layout,
  Modal,
  Popover,
  Space,
  Tree,
  Typography,
  Upload,
} from '@arco-design/web-react'
import { IconClose, IconDelete, IconEdit, IconFile, IconPlus, IconSend, IconUpload } from '@arco-design/web-react/icon'

// 引入 wangEditor
import { Editor, Toolbar } from '@wangeditor/editor-for-react'

// 公共事件
import { isSvg } from 'src/utils'

// detail 邮件详情
// userList 用户列表
// recentlyContact 最近联系人
// onClose 关闭回调
// onChange 数据变化回调
// onSend 发送邮件回调
// onEditContact 编辑联系人回调
export default function WriteMail({
  detail,
  userList = [],
  recentlyContact = [],
  onClose,
  onChange,
  onSend,
  onEditContact,
  onDelete,
  onClear,
}) {
  const [form] = Form.useForm()
  const [formContact] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const [addCC, setAddCC] = useState(false)

  const [fileList, setFileList] = useState([])
  const [editor, setEditor] = useState(null)
  const [html, setHtml] = useState('')

  const toRef = useRef(null)
  const ccRef = useRef(null)
  const [lastFocus, setLastFocus] = useState(null) // 缓存最后一次焦点

  // 选择用户
  const onSelectUser = (ids, extra) => {
    const { e } = extra
    if (isSvg(e)) return

    const key = ids[0]
    const item = [...userList, ...recentlyContact].find((e) => e?.email === key)
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
    if (detail?.uid) {
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
      onSend(type, form, html, fileList, detail, setLoading)
    }
  }

  // 监控数据变化
  const onChangeMail = (_, values) => {
    const newValues = {
      ...values,
      detail: {
        content: html || detail?.detail?.content,
        attachments: values?.files || fileList?.length > 0 ? fileList : detail?.detail?.attachments,
      },
    }
    onChange(newValues)
  }

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
          <Form.Item label='邮箱' field='email'>
            <Input placeholder='邮箱' />
          </Form.Item>
        </Form>
      ),
      onOk: async () => {
        const values = await formContact.validate()
        onEditContact(values)
      },
    })
  }

  // 联系人-生成树节点
  const generatorTreeNodes = (data) => {
    return data.map((item) => {
      const { children, key, ...rest } = item
      rest.title = children ? (
        item.full_name
      ) : (
        <Popover
          position='tr'
          trigger='hover'
          key={key}
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
                onClick={() => onDelete(item)}>
                <IconDelete />
              </Button>
            )}
          </div>
        </Popover>
      )

      return (
        <Tree.Node key={rest?.email} {...rest} dataRef={item}>
          {children ? generatorTreeNodes(item.children) : null}
        </Tree.Node>
      )
    })
  }

  return (
    <Layout className='h-full'>
      <Layout.Header className='flex h-15 items-center justify-between border-b border-gray-300 px-6'>
        <Space>
          <Button type='primary' icon={<IconSend />} loading={loading} onClick={() => handleSend('Sent')}>
            发送邮件
          </Button>
          <Button type='secondary' icon={<IconFile />} loading={loading} onClick={() => handleSend('Drafts')}>
            存草稿
          </Button>
        </Space>
        <Space>
          {!addCC && (
            <Button type='text' icon={<IconPlus />} onClick={() => openCC()}>
              添加抄送
            </Button>
          )}
          <IconClose className='cursor-pointer' onClick={() => onClose('inbox')} />
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
            <Tree className='mail-contacts h-full' checkStrictly blockNode onSelect={(e, extra) => onSelectUser(e, extra)}>
              {generatorTreeNodes([
                {
                  full_name: (
                    <div className='group flex justify-between leading-6'>
                      <span>最近联系人</span>
                      <Button
                        type='text'
                        status='danger'
                        size='mini'
                        className='hidden! group-hover:block!'
                        onClick={() => onClear()}>
                        清空
                      </Button>
                    </div>
                  ),
                  email: '0-0',
                  selectable: false,
                  children: recentlyContact,
                },
                {
                  full_name: '邮箱联系人',
                  email: '0-1',
                  selectable: false,
                  children: userList,
                },
              ])}
            </Tree>
          </Card>
        </div>
      </Layout.Content>
    </Layout>
  )
}

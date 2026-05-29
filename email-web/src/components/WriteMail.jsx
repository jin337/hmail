import { Button, Card, Form, Input, InputTag, Layout, Space, Typography, Upload } from '@arco-design/web-react'
import { IconClose, IconFile, IconPlus, IconSend, IconUpload } from '@arco-design/web-react/icon'

import { useEffect, useState } from 'react'

// 引入 wangEditor
import { Editor, Toolbar } from '@wangeditor/editor-for-react'

export default function WriteMail({ detail, userList = [], onClose, onChange, onSend }) {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const [addCC, setAddCC] = useState(false)

  const [fileList, setFileList] = useState([])
  const [editor, setEditor] = useState(null)
  const [html, setHtml] = useState('')

  // 自动回填
  useEffect(() => {
    if (detail?.uid) {
      form.setFieldsValue({ ...detail, to: detail?.to_email, cc: detail?.cc_email })

      setAddCC(detail?.cc_email?.length > 0)

      const list = (detail?.detail?.attachments || []).map((e) => ({
        ...e,
        name: e.file_name,
        uid: e.part_id,
      }))
      setFileList(list)

      if (editor) {
        editor.setHtml(`${detail?.detail?.content || ''}`)
      }
    }
  }, [detail, form, editor])

  // 销毁编辑器
  useEffect(() => {
    return () => {
      if (editor) editor.destroy()
    }
  }, [])

  // 提取验证函数
  const validateEmails = (value, callback) => {
    const EMAIL_REGEX = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
    if (!value || value.length === 0) {
      return callback()
    }
    const invalid = value.filter((v) => !EMAIL_REGEX.test(v))
    if (invalid.length > 0) {
      callback(`存在无效的邮箱地址: ${invalid.join(', ')}`)
    } else {
      callback()
    }
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
        content: html,
        attachments: values?.files || fileList,
      },
    }

    onChange(newValues)
  }

  return (
    <Layout className='h-full'>
      <Layout.Header className='flex h-14 items-center justify-between border-b border-gray-300 px-6'>
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
            <Button type='text' icon={<IconPlus />} onClick={() => setAddCC(true)}>
              添加抄送
            </Button>
          )}
          <IconClose className='cursor-pointer' onClick={() => onClose('inbox')} />
        </Space>
      </Layout.Header>
      <Layout.Content>
        <div className='flex h-[calc(100vh-112px)] items-start'>
          <Form className='h-full flex-1 overflow-y-auto p-6 pb-0' form={form} layout='vertical' onChange={onChangeMail}>
            <Form.Item field='to' rules={[{ required: true, message: '请输入收件人' }, { validator: validateEmails }]}>
              <InputTag prefix='收件人' placeholder='test@xxx.com' saveOnBlur />
            </Form.Item>
            <Form.Item field='cc' hidden={!addCC} rules={[{ validator: validateEmails }]}>
              <InputTag prefix='抄送' placeholder='test@xxx.com' saveOnBlur />
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
                showUploadList={{ startIcon: null }}
                fileList={fileList}
                onChange={setFileList}>
                <Button icon={<IconUpload />}>上传附件</Button>
              </Upload>
            </Form.Item>
          </Form>
          <Card title='联系人' className='h-full w-50'>
            {userList?.map((item) => (
              <Typography.Paragraph copyable key={item?.id}>
                {item.email}
              </Typography.Paragraph>
            ))}
          </Card>
        </div>
      </Layout.Content>
    </Layout>
  )
}

import { Button, Form, Input, InputTag, Layout, Message, Space, Upload } from '@arco-design/web-react'
import { IconClose, IconFile, IconPlus, IconSend, IconUpload } from '@arco-design/web-react/icon'

import { useEffect, useState } from 'react'

// 引入 wangEditor
import { Editor, Toolbar } from '@wangeditor/editor-for-react'

import request from 'src/api/request'
export default function WriteMail({ detail, onClose }) {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [fileList, setFileList] = useState([])
  const [editor, setEditor] = useState(null)
  const [html, setHtml] = useState('')

  const [addCC, setAddCC] = useState(false)

  // 自动回填
  useEffect(() => {
    if (detail?.uid) {
      form.setFieldsValue({ ...detail, to: detail?.to_email, cc: detail?.cc_email })

      setAddCC(!!detail?.cc_email.length)

      const list = (detail?.detail?.attachments || []).map((e) => ({
        ...e,
        name: e.file_name,
        uid: e.part_id,
        originFile: null,
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

  const handleSend = async (type) => {
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
      onClose(type === 'Sent' ? 'sent' : 'drafts')
      if (editor) editor.destroy()
    } else {
      setLoading(false)
      Message.error(msg)
    }
    setLoading(false)
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
        <Form className='h-[calc(100vh-112px)] overflow-y-auto p-6 pb-0' form={form} layout='vertical'>
          <Form.Item field='to' rules={[{ required: true, message: '请输入收件人' }]}>
            <InputTag prefix='收件人' placeholder='test@xxx.com' />
          </Form.Item>
          <Form.Item field='cc' hidden={!addCC}>
            <InputTag prefix='抄送' />
          </Form.Item>
          <Form.Item field='subject' rules={[{ required: true, message: '请输入主题' }]}>
            <Input prefix='主题' placeholder='邮件主题' />
          </Form.Item>
          {/* 富文本编辑器 */}
          <Form.Item>
            <div className='z-100 overflow-hidden rounded border border-gray-300'>
              <Toolbar
                editor={editor}
                defaultConfig={{ excludeKeys: ['group-video', 'group-image', 'insertTable', 'codeBlock'] }}
                mode='default'
                className='border-b border-gray-300'
              />
              <Editor
                className='h-80 overflow-y-auto'
                defaultConfig={{ placeholder: '请输入邮件正文...' }}
                onCreated={setEditor}
                onChange={(editor) => setHtml(editor.getHtml())}
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
      </Layout.Content>
    </Layout>
  )
}

import { useState } from 'react'
import { useOutletContext } from 'react-router'

import { Button, Descriptions, Form, Input, Layout, Message, Progress, Tabs, Upload } from '@arco-design/web-react'
import { IconEdit } from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

import request from 'src/api/request'

const Personal = () => {
  const { currentAccountId, baseUrl, userInfo, userToken, setUserInfo, onLogout } = useOutletContext()
  const [formPwd] = Form.useForm()
  const [file, setFile] = useState(null)
  const [edit, setEdit] = useState(false)

  // 修改姓名
  const onEditName = async (e) => {
    const params = {
      id: userInfo.id,
      email: userInfo.email,
      is_admin: userInfo.is_admin,
      person_first_name: e.slice(0, 1),
      person_last_name: e.slice(1),
    }
    const { code, msg } = await request.post('/api/user/update', params)
    if (code === 200) {
      Message.success(msg)
      setEdit(false)
      const data = {
        ...userInfo,
        full_name: e,
      }
      localStorage.setItem(`USERINFO_${currentAccountId}`, JSON.stringify(data))
      setUserInfo(data)
    } else {
      Message.error(msg)
    }
  }

  // 修改密码
  const submitPwd = () => {
    formPwd.validate().then(async (values) => {
      const params = {
        old_password: values.oldpwd,
        new_password: values.newpwd,
      }
      const { code, message } = await request.post('/api/user/chgpwd', params)
      if (code === 200) {
        Message.success('密码修改成功，请重新登录')
        localStorage.removeItem('mail_remember')
        onLogout()
      } else {
        Message.error(message)
      }
    })
  }

  return (
    <Layout.Content className={'mx-4 rounded-xl bg-white p-4'}>
      <Tabs className={'h-full'} size='large' tabPosition='left' defaultActiveTab='1'>
        <Tabs.TabPane key='1' title='个人资料'>
          <Descriptions
            className='descriptions-wrap mx-auto w-90!'
            column={1}
            data={[
              {
                label: '头像',
                value: (
                  <Upload
                    action={baseUrl + '/api/user/uploadavatar'}
                    headers={{
                      Authorization: userToken,
                    }}
                    key={userInfo?.email}
                    fileList={file ? [file] : []}
                    showUploadList={false}
                    accept='image/*'
                    onChange={(_, currentFile) => {
                      const { response } = currentFile
                      if (response?.code === 200) {
                        setUserInfo((prev) => ({
                          ...prev,
                          avatar: baseUrl + `/api/viewfile?url=static/avatars/${userInfo?.email}.webp?v=${dayjs().unix()}`,
                        }))
                      }
                    }}
                    onProgress={(currentFile) => {
                      setFile(currentFile)
                    }}>
                    <div className={`arco-upload-list-item mt-0! pr-0! ${file && file?.status === 'error' ? ' is-error' : ''}`}>
                      <div className='h-10 w-10 border border-dashed border-(--color-neutral-3) bg-(--color-fill-2)'>
                        <img className='h-full w-full' src={userInfo?.avatar} />
                        <div className='arco-upload-list-item-picture-mask h-10 w-10 leading-10!'>
                          <IconEdit />
                        </div>
                        {file?.status === 'uploading' && file?.percent < 100 && (
                          <Progress
                            percent={file.percent}
                            type='circle'
                            size='mini'
                            style={{
                              position: 'absolute',
                              left: '50%',
                              top: '50%',
                              transform: 'translateX(-50%) translateY(-50%)',
                            }}
                          />
                        )}
                      </div>
                    </div>
                  </Upload>
                ),
              },
              {
                label: '姓名',
                value: edit ? (
                  <Input.Search
                    size='small'
                    className='w-50!'
                    defaultValue={userInfo?.full_name}
                    searchButton='保存'
                    onSearch={onEditName}
                  />
                ) : (
                  <>
                    {userInfo?.full_name}
                    <IconEdit className='ml-2 cursor-pointer' onClick={() => setEdit(true)} />
                  </>
                ),
              },
              {
                label: '账号',
                value: userInfo?.email,
              },
              {
                label: '权限',
                value: userInfo?.is_admin === 1 ? '管理员' : '用户',
              },
            ]}
            labelStyle={{ textAlign: 'right', paddingRight: 36 }}
          />
        </Tabs.TabPane>
        <Tabs.TabPane key='2' title='修改密码'>
          <Form
            className='mx-auto w-90!'
            form={formPwd}
            autoComplete='off'
            layout='vertical'
            validateMessages={{ required: (_, { label }) => `${label}是必填项` }}>
            <Form.Item
              rules={[
                { required: true, message: '请输入旧密码' },
                {
                  validator: (value, callback) => {
                    if (!value) {
                      return callback()
                    }
                    if (value.length < 6 || value.length > 20) {
                      return callback('密码长度必须在6到20个字符之间')
                    }
                    if (value.includes('^')) {
                      return callback('密码不能包含^字符')
                    }
                    return Promise.resolve()
                  },
                },
              ]}
              label='旧密码'
              field='oldpwd'>
              <Input.Password placeholder='请输入...' />
            </Form.Item>
            <Form.Item
              rules={[
                { required: true, message: '请输入新密码' },
                {
                  validator: (value, callback) => {
                    if (!value) {
                      return callback()
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
                  },
                },
              ]}
              label='新密码'
              field='newpwd'>
              <Input.Password placeholder='请输入...' />
            </Form.Item>
            <Form.Item
              rules={[
                { required: true, message: '请确认新密码' },
                {
                  validator: (value, callback) => {
                    if (!value) {
                      return callback()
                    }
                    const newPassword = formPwd.getFieldValue('newpwd')
                    if (newPassword && value !== newPassword) {
                      return callback('两次输入的密码不一致')
                    }
                    return Promise.resolve()
                  },
                },
              ]}
              label='确认新密码'
              field='confirmpwd'>
              <Input.Password placeholder='请输入...' />
            </Form.Item>
            <Form.Item className='text-right'>
              <Button type='primary' onClick={submitPwd}>
                修改密码
              </Button>
            </Form.Item>
          </Form>
        </Tabs.TabPane>
      </Tabs>
    </Layout.Content>
  )
}
export default Personal

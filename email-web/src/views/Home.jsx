import { useEffect, useState } from 'react'
import { Outlet, useNavigate } from 'react-router'

import { Button, Dropdown, Form, Input, Layout, Menu, Message, Modal, Space } from '@arco-design/web-react'
import { IconSettings } from '@arco-design/web-react/icon'

import request from 'src/api/request'

const Home = () => {
  const navigate = useNavigate()
  const [formPwd] = Form.useForm()
  const [visible, setVisible] = useState(false)

  // 本地登录信息
  const userToken = localStorage.getItem('mail_token')
  const userInfo = JSON.parse(localStorage.getItem('mail_info') || '{}')

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
      const { code, message } = await request.post('/api/user/chgpwd', params)
      if (code === 200) {
        Message.success('密码修改成功，请重新登录')
        localStorage.removeItem('mail_remember')
        handleLogout()
      } else {
        Message.error(message)
      }
    })
    setVisible(false)
  }

  // 退出
  const handleLogout = () => {
    localStorage.removeItem('mail_token')
    localStorage.removeItem('mail_info')
    navigate('/login')
  }

  // 初始加载，如果没有登录信息则跳转到登录页
  useEffect(() => {
    if (!userToken) {
      navigate('/login')
    }
  }, [userToken, navigate])

  return (
    <Layout className='h-screen w-full overflow-hidden'>
      <Layout.Header className='z-10 flex h-14 items-center justify-between bg-white px-6 shadow-sm'>
        {/* Logo */}
        <div className='flex cursor-pointer items-center gap-3' onClick={() => navigate('/')}>
          <div className='flex h-8 w-8 items-center justify-center rounded bg-blue-500 text-lg font-bold text-white'>H</div>
          <span className='text-lg font-bold text-gray-700'>华盛邮件</span>
        </div>
        {/* 头像和退出登录按钮 */}
        <Space>
          {userInfo?.full_name}
          {userInfo?.email}
          <Dropdown
            position='br'
            trigger='click'
            droplist={
              <Menu>
                {userInfo?.is_admin === 1 && (
                  <Menu.Item key='0' onClick={() => navigate('/user')}>
                    用户管理
                  </Menu.Item>
                )}
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

      <Outlet />

      {/* 修改密码 */}
      <Modal
        title='修改密码'
        visible={visible}
        onOk={() => submitPwd()}
        onCancel={() => setVisible(false)}
        autoFocus={false}
        focusLock={true}>
        <Form
          className='m-auto w-90!'
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
        </Form>
      </Modal>
    </Layout>
  )
}

export default Home

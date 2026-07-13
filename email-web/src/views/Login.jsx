import { useState } from 'react'
import { useNavigate } from 'react-router'

import { Button, Card, Checkbox, Form, Input, Message } from '@arco-design/web-react'

import request from 'src/api/request'

import loginImg from '../assets/img_login.gif'
const suffix = import.meta.env.VITE_SUFFIX
const pageTitle = import.meta.env.VITE_PAGE_TITLE

export default function Login() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  // 登录
  const onSubmit = async (values) => {
    setLoading(true)
    const params = {
      password: values.password,
      email: values.email + suffix,
    }
    const { code, data, msg } = await request.post('/api/login', params)
    if (code === 200) {
      if (values.remember) {
        localStorage.setItem('mail_remember', JSON.stringify(values))
      } else {
        localStorage.removeItem('mail_remember')
      }
      const accountId = data.email
      localStorage.setItem(`TOKEN_${accountId}`, data.token)
      localStorage.setItem(`USERINFO_${accountId}`, JSON.stringify(data))

      localStorage.setItem('current_account_id', accountId)
      Message.success('登录成功')
      navigate('/')
    } else {
      setLoading(false)
      Message.error(msg)
    }
    setLoading(false)
  }

  const remember = JSON.parse(localStorage.getItem('mail_remember') || '{}')

  return (
    <div className='min-h-screen bg-[#f2faff]'>
      {/* 主内容区域 */}
      <div className='flex items-center justify-center px-8 py-12'>
        <div className='flex w-full max-w-6xl items-center justify-center gap-16 pt-12'>
          {/* 左侧宣传文案 */}
          <div>
            <h1 className='mb-6 h-8 text-4xl font-bold'>
              <span className='text-[#00a4ff]'>{pageTitle}</span>
              <span className='text-black'>，常联系！</span>
            </h1>
            <div className='space-y-3 text-base text-gray-800'>
              <p>1987年9月14日21时07分</p>
              <p>中国第一封电子邮件</p>
              <p>从北京发往德国</p>
              <p>"越过长城，走向世界"</p>
            </div>
            <div className='h-95 w-95'>
              <img className='-ml-18 w-full' src={loginImg} alt='企业邮箱' />
            </div>
          </div>

          {/* 右侧登录卡片 */}

          <Card className={'w-87.5'} title='邮件系统登录'>
            <Form initialValues={remember} autoComplete='off' onSubmit={onSubmit} layout='vertical'>
              <Form.Item label='邮箱地址' field='email' rules={[{ required: true, message: '请输入邮箱' }]}>
                <Input suffix={suffix} placeholder='请输入用户名' />
              </Form.Item>
              <Form.Item label='邮箱密码' field='password' rules={[{ required: true, message: '请输入密码' }]}>
                <Input.Password placeholder='请输入密码' />
              </Form.Item>
              <Form.Item field='remember' triggerPropName='checked'>
                <Checkbox>记住密码</Checkbox>
              </Form.Item>
              <Button type='primary' htmlType='submit' loading={loading} long>
                登录
              </Button>
            </Form>
          </Card>
        </div>
      </div>
    </div>
  )
}

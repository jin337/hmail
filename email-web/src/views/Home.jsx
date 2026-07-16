import { useEffect, useState } from 'react'
import { Outlet, useNavigate } from 'react-router'

import { Avatar, Button, Dropdown, Layout, Menu, Space, Tag } from '@arco-design/web-react'
import { IconSettings } from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

const pageTitle = import.meta.env.VITE_PAGE_TITLE
const baseUrl = import.meta.env.VITE_BASE_URL

const Home = () => {
  const navigate = useNavigate()
  const [time, setTime] = useState(dayjs().unix())

  // 本地登录信息
  const currentAccountId = localStorage.getItem('current_account_id') || ''
  const userToken = currentAccountId ? localStorage.getItem(`TOKEN_${currentAccountId}`) : null
  const userInfo = currentAccountId ? JSON.parse(localStorage.getItem(`USERINFO_${currentAccountId}`) || '{}') : {}

  // 退出
  const handleLogout = () => {
    if (currentAccountId) {
      // 删除当前账号独立存储
      localStorage.removeItem(`TOKEN_${currentAccountId}`)
      localStorage.removeItem(`USERINFO_${currentAccountId}`)
      localStorage.removeItem(`current_account_id`)
    }
    // 清空活跃账号标记
    localStorage.removeItem('current_account_id')
    navigate('/login')
  }

  // 初始加载，如果没有登录信息则跳转到登录页
  useEffect(() => {
    if (!currentAccountId || !userToken) {
      navigate('/login')
    }
  }, [currentAccountId, userToken, navigate])

  return (
    <Layout className='h-screen w-screen overflow-hidden'>
      <Layout.Header className='z-10 flex h-14 items-center justify-between pr-6'>
        {/* Logo */}
        <div className='flex h-full w-55 cursor-pointer items-center justify-center'>
          <div className='flex items-center gap-3' onClick={() => navigate('/')}>
            <div className='flex h-8 w-8 items-center justify-center rounded bg-blue-500 text-lg font-bold text-white'>H</div>
            <span className='text-lg font-bold text-gray-700'>{pageTitle}</span>
          </div>
        </div>

        {/* 头像和退出登录按钮 */}
        <Space>
          <Avatar size={32}>
            <img alt='avatar' src={baseUrl + `static/avatars/${userInfo?.email}.webp?v=${time}`} />
          </Avatar>
          {userInfo?.full_name}&middot;
          {userInfo?.email}
          {userInfo.is_admin ? <Tag color='#165dff'>管理员</Tag> : <Tag color='#b8c8db'>用户</Tag>}
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
                <Menu.Item key='1' onClick={() => navigate('/personal')}>
                  个人中心
                </Menu.Item>
                <Menu.Item key='2' onClick={handleLogout}>
                  退出登录
                </Menu.Item>
              </Menu>
            }>
            <Button size='small' type='text' className='text-gray-500!' icon={<IconSettings />}>
              设置
            </Button>
          </Dropdown>
        </Space>
      </Layout.Header>

      <Outlet />
    </Layout>
  )
}

export default Home

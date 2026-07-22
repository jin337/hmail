import { useEffect, useRef, useState } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router'

import { Avatar, Button, Dropdown, Input, Layout, Menu, Space, Tag } from '@arco-design/web-react'
import { IconSearch, IconSettings } from '@arco-design/web-react/icon'

const pageTitle = import.meta.env.VITE_PAGE_TITLE
const baseUrl = import.meta.env.VITE_BASE_URL

const Home = () => {
  // 本地登录信息
  const currentAccountId = localStorage.getItem('current_account_id') || ''
  const userToken = currentAccountId ? localStorage.getItem(`TOKEN_${currentAccountId}`) : null

  const navigate = useNavigate()
  const location = useLocation()
  const [userInfo, setUserInfo] = useState(JSON.parse(localStorage.getItem(`USERINFO_${currentAccountId}`) || '{}'))
  const [searchWord, setSearchWord] = useState('') // 搜索

  // 退出
  const onLogout = () => {
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

  // 页面传递参数
  const outletCtx = {
    baseUrl,
    currentAccountId,
    userInfo,
    userToken,
    setUserInfo,
    onLogout,
    searchWord,
    setSearchWord,
    registerMethod: (name, fn) => {
      childMethods.current[name] = fn
    },
  }

  // 子页面事件
  const childMethods = useRef({})
  const onParentSearch = (val) => {
    if (childMethods.current.onSearch) {
      childMethods.current.onSearch(val)
    }
  }

  return (
    <Layout className='h-screen w-screen overflow-y-hidden'>
      <Layout.Header className='z-10 flex h-14 items-center justify-between pr-6'>
        <div className='flex h-full items-center'>
          {/* Logo */}
          <div className='flex w-55 cursor-pointer items-center justify-center gap-3' onClick={() => navigate('/')}>
            <div className='flex h-8 w-8 items-center justify-center rounded bg-blue-500 text-lg font-bold text-white'>H</div>
            <span className='text-lg font-bold text-gray-700'>{pageTitle}</span>
          </div>
          {/* 搜索框 */}
          {location.pathname === '/' && (
            <div className='w-98'>
              <Input.Search
                prefix={<IconSearch />}
                placeholder='搜索主题/发件人'
                searchButton
                allowClear
                value={searchWord}
                onChange={setSearchWord}
                onSearch={onParentSearch}
                onClear={onParentSearch}
              />
            </div>
          )}
        </div>

        {/* 头像和退出登录按钮 */}
        <Space>
          <Avatar size={32}>
            <img alt='avatar' src={userInfo?.avatar} />
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
                <Menu.Item key='2' onClick={onLogout}>
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

      <Outlet context={outletCtx} />
    </Layout>
  )
}

export default Home

import request from './request'

// 用户列表
export const fetchUserList = () => request.post('/api/user/list')

// 创建用户
export const createUser = (params) => request.post('/api/user/create', params)

// 更新用户
export const updateUser = (params) => request.post('/api/user/update', params)

// 删除用户
export const deleteUser = (params) => request.post('/api/user/delete', params)

// 修改密码
export const changePassword = (params) => request.post('/api/user/chgpwd', params)

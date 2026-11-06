import request from './request'

// 联系人列表（prefix: user_sent 最近联系人 / user_contact 我的联系人）
export const fetchContactList = (params) => request.post('/api/user/contact/list', params)

// 保存联系人
export const saveContact = (params) => request.post('/api/user/contact/save', params)

// 删除联系人
export const deleteContact = (params) => request.post('/api/user/contact/delete', params)

// 清空联系人
export const clearContact = (params) => request.post('/api/user/contact/clear', params)

import request from './request'

// 邮件列表
export const fetchMailList = (params) => request.post('/api/mail/list', params)

// 星标邮件列表
export const fetchStarMailList = (params) => request.post('/api/mail/star-list', params)

// 各文件夹未读数
export const fetchUnreadTotal = () => request.post('/api/mail/unread-total')

// 邮件详情（后端按 X-Client-Host 生成内联图片地址）
export const fetchMailDetail = (params) =>
  request.post('/api/mail/detail', params, {
    headers: { 'X-Client-Host': window.location.host },
  })

// 标记已读/星标等状态
export const updateMailStatus = (params) => request.post('/api/mail/status', params)

// 下载附件
export const downloadAttachment = (params) =>
  request.post('/api/mail/download', params, { responseType: 'blob' })

// 移动邮件
export const moveMail = (params) => request.post('/api/mail/move', params)

// 删除邮件
export const deleteMail = (params) => request.post('/api/mail/delete', params)

// 保存草稿
export const saveDraft = (formData) =>
  request.post('/api/mail/save-draft', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

// 发送邮件
export const sendMail = (formData) =>
  request.post('/api/mail/send', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

// 取消定时发送
export const unScheduleMail = (params) => request.post('/api/mail/un-schedule', params)

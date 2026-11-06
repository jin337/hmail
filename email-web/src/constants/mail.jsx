import { IconDelete, IconEmail, IconFile, IconSend, IconStar } from '@arco-design/web-react/icon'

// 标准文件夹
export const INBOX = 'INBOX'
export const STAR = 'Star'
export const SENT = 'Sent'
export const DRAFTS = 'Drafts'
export const DELETED = 'Deleted'

// 每页数量
export const PAGE_SIZE = 50

// 目录菜单
export const menuList = [
  { key: 'inbox', folder: INBOX, title: '收件箱', icon: <IconEmail /> },
  { key: 'star', folder: STAR, title: '星标邮件', icon: <IconStar /> },
  { key: 'sent', folder: SENT, title: '已发送', icon: <IconSend /> },
  { key: 'drafts', folder: DRAFTS, title: '草稿箱', icon: <IconFile /> },
  { key: 'delete', folder: DELETED, title: '垃圾箱', icon: <IconDelete /> },
]

// 移动文件夹
export const moveList = [
  { key: 'inbox', folder: INBOX, title: '收件箱', icon: <IconEmail /> },
  { key: 'sent', folder: SENT, title: '已发送', icon: <IconSend /> },
  { key: 'delete', folder: DELETED, title: '垃圾箱', icon: <IconDelete /> },
]

// 筛选
export const filterList = [
  {
    label: '筛选',
    children: [
      {
        label: '全部',
        value: 'all',
        key: 0,
      },
      {
        label: '未读',
        value: 'unread',
        key: 0,
      },
    ],
  },
  {
    label: '分割线',
    value: 'divider',
  },
  {
    label: '排序方式',
    children: [
      {
        label: '按日期',
        children: [
          {
            label: '由新到旧',
            value: 'date_desc',
            key: 1,
          },
          {
            label: '由旧到新',
            value: 'date_asc',
            key: 1,
          },
        ],
      },
      {
        label: '按大小',
        children: [
          {
            label: '由大到小',
            value: 'size_desc',
            key: 1,
          },
          {
            label: '由小到大',
            value: 'size_asc',
            key: 1,
          },
        ],
      },
    ],
  },
]

// 默认标记列表
export const baseFlags = [
  { flag: 'Seen', key: 1, title: '已读邮件' },
  { flag: 'Seen', key: 2, title: '未读邮件' },
  { title: '分割线', flag: 'divider' },
  { flag: 'Flagged', key: 1, title: '星标邮件' },
  { flag: 'Flagged', key: 2, title: '取消星标' },
]

// 默认筛选参数
export const defaultFilterKeys = ['all', 'date_desc']

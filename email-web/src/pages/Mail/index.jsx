import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router'

import { Layout, Message, Modal } from '@arco-design/web-react'
import { IconDelete, IconEmail, IconFile, IconSend, IconStar } from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

// 组件
import WriteMail from 'src/components/WriteMail'
import Content from './Content'
import Menu from './Menu'

import { MailProvider } from './MailContext'

import request from 'src/api/request'

import { getFileType, throttle, transHtmlAttrs } from 'src/utils/index'

// 目录菜单
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'star', folder: 'Star', title: '星标邮件', icon: <IconStar /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

// 移动文件夹
const moveList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete /> },
]

// 筛选
const filterList = [
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

// 标记
const flags = [
  { flag: 'Seen', key: 1, title: '已读邮件' },
  { flag: 'Seen', key: 2, title: '未读邮件' },
  { title: '分割线', flag: 'divider' },
  { flag: 'Flagged', key: 1, title: '星标邮件' },
  { flag: 'Flagged', key: 2, title: '取消星标' },
]

// 获取图片id
const getImageIds = (html) => {
  if (!html) return []

  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')

  const images = doc.querySelectorAll('img[data-href]')

  const ids = []
  images.forEach((img) => {
    const href = img.getAttribute('data-href')
    const match = href.match(/image_([\d.]+)\.(png|jpg|jpeg|gif|webp|bmp)$/i)
    if (match && match[1]) {
      ids.push(match[1])
    }
  })

  return ids
}

const MailLayout = () => {
  const { currentAccountId, baseUrl, userInfo, searchWord, setSearchWord, registerMethod } = useOutletContext()

  const tableRef = useRef()
  const [refreshCount, setRefreshCount] = useState(0) // 刷新次数
  const timerRef = useRef(null) // 定时器

  const [folderList, setFolderList] = useState(menuList) // 目录菜单
  const [currentFolder, setCurrentFolder] = useState({}) // 当前文件夹

  const [mailList, setMailList] = useState({}) // 邮件列表
  const [listLoading, setListLoading] = useState(false) // 列表加载中
  const [filterKeys, setFilterKeys] = useState(['all', 'date_desc']) // 已筛选参数
  const [selectedRowKeys, setSelectedRowKeys] = useState([]) // 已选择行

  const [isTable, setIsTable] = useState(() => localStorage.getItem('isTable') === 'true') // 表格模式
  const [isMove, setIsMove] = useState(false) // 移动模式

  const [currentMail, setCurrentMail] = useState(null) // 当前邮件
  const [mailLoading, setMailLoading] = useState(false) // 邮件加载中
  const [newMailInfo, setNewMailInfo] = useState(null) // 已改变的邮件内容

  const [userList, setUserList] = useState([]) // 用户列表
  const [recentlyList, setRecentlyList] = useState([]) // 最近联系人
  const [contactList, setContactList] = useState([]) // 联系人

  const [flagList, setFlagList] = useState(flags)

  const pageSize = 35 // 每页数量

  // 取消发送
  const onUnSchedule = async (item, type = 2) => {
    const params = {
      uid: item.uid,
      folder: item.folder,
      status: 'Draft',
      type,
    }
    const { code } = await request.post('/api/mail/un-schedule', params)
    if (code === 200) {
      setCurrentMail(null)
      getMailList({
        folder: item.folder,
        filter: filterKeys,
        keyword: searchWord,
        page: 1,
        size: pageSize,
      })
    }
  }

  // 发送邮件&草稿
  const onSend = async (type, detailProps, customTime) => {
    const { detail, ...rest } = detailProps

    if (!rest.to_info || !rest.subject) {
      Message.warning('请填写收件人和主题')
      return
    }

    const formData = new FormData()
    const to = rest.to_info.map((e) => e.value)
    const cc = rest?.cc_info?.map((e) => e.value) || ''

    formData.append('to', to)
    formData.append('cc', cc)
    formData.append('subject', rest.subject)

    if (rest?.is_reply || rest?.is_forward) {
      formData.append('folder', rest?.folder)
    } else {
      formData.append('folder', rest?.uid ? 'Drafts' : type)
    }

    if (rest?.uid) {
      formData.append('uid', rest.uid)
    }
    if (customTime) {
      formData.append('x-schedule-send', customTime)
    }

    // 回复
    if (rest?.is_reply) {
      const references = rest.references + ' ' + rest?.message_id
      formData.append('in_reply_to', rest?.message_id)
      formData.append('references', references)
    }

    // 转发
    if (rest?.is_forward) {
      const references = rest.references + ' ' + rest?.message_id
      formData.append('references', references)
    }

    // 邮件内容
    const content = transHtmlAttrs(detail.content, 'data-href', 'src')
    formData.append('content', content)

    // 提取 HTML 中的图片 ID
    const imageIds = getImageIds(content)

    // 附件
    const partIds = []
    detail?.attachments?.forEach((file) => {
      if (file?.part_id) {
        partIds.push(file.part_id)
      } else {
        formData.append('files', file?.originFile)
      }
    })

    const allPartIds = [...partIds, ...imageIds]
    if (allPartIds.length > 0) {
      formData.append('part_ids', allPartIds.join(','))
    }

    let url = ''
    if (type === 'Drafts') {
      url = '/api/mail/save-draft'
    } else {
      url = '/api/mail/send'

      // 标记已回复
      if (detail?.is_reply) {
        const params = {
          uid: detail?.uid,
          folder: detail?.folder,
          status: 'Answered',
          type: 1,
        }
        await request.post('/api/mail/status', params)
      }
    }

    if (url) {
      const { code, msg } = await request.post(url, formData, { headers: { 'Content-Type': 'multipart/form-data' } })
      if (code === 200) {
        Message.success(msg)
        onCloseEdit(type === 'Sent' ? 'sent' : 'drafts') // 关闭邮件，跳转到发导航
        getContactList({ prefix: 'user_sent' }) // 更新联系人
      } else {
        Message.error(msg)
      }
    }
  }

  // 回复、转发
  const onReplyForward = (key) => {
    if (!currentMail) return
    let newMail = null

    const to_reply = currentMail?.to_info
      ?.map(
        (t) =>
          `<span style="color: rgb(0, 0, 0);">${t.name}</span> <span style="color: rgb(149, 157, 166);">&lt;${t.email}&gt;</span>`
      )
      .join(', ')

    const cc_reply = currentMail?.cc
      ? currentMail?.cc_info
          ?.map(
            (t) =>
              `<span style="color: rgb(0, 0, 0);">${t.name}</span> <span style="color: rgb(149, 157, 166);">&lt;${t.email}&gt;</span>`
          )
          .join(', ')
      : ''

    // 邮件原内容
    const FormContent = `<div><br></div>
    <div><br></div>
    <article>
      <div style="display:flex;align-items:center;padding-top:8px">
        <div style="color:#959DA6;font-size:12px;line-height:30px">原始邮件</div>
        <hr style="border-width: medium; border-style: none; border-color: currentcolor; border-image: none;flex-grow:1;border-top:1px solid rgba(21, 46, 74, 0.07);margin-left:8px">
            </div>
      <div style="line-height: 20px; border-radius: 6px; background-color: rgba(20, 46, 77, 0.05); color: rgb(0, 0, 0); margin: 0px; padding: 8px; width: 100%;">
        <div style="line-height: 20px; font-size: 12px;">
          发件人：<span style="color: rgb(0, 0, 0);">${currentMail?.from_info?.name}</span><span style="color: rgb(149, 157, 166);">&lt;${currentMail?.from}&gt;</span>
        </div>
        <div style="line-height: 20px; font-size: 12px;">
          发件时间：<span style="color: rgb(0, 0, 0);">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
        </div>
        <div style="line-height: 20px; font-size: 12px;">收件人：${to_reply}</div>
        ${currentMail?.cc ? `<div style="line-height: 20px; font-size: 12px;">抄送：${cc_reply}</div>` : ''}
        <div style="line-height: 20px; font-size: 12px;">
          主题：<span style="color: rgb(0, 0, 0);">${currentMail?.subject}</span>
        </div>
      </div>
      <div><br></div>
      <div>${currentMail?.detail?.content}</div>
    </article>`

    // 回复
    if (key == 'is_reply') {
      newMail = {
        ...currentMail,
        subject: `回复: ${currentMail.subject}`,
        to_info: currentMail.to_info.map((e) => ({ label: e.name, value: e.email })),
        cc_info: currentMail?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
        detail: {
          ...currentMail?.detail,
          content: FormContent,
        },
        is_reply: true,
      }

      if (currentFolder.key === 'inbox') {
        newMail.to_info = [{ label: currentMail.from_info.name, value: currentMail.from }]
      }
    }

    // 转发
    if (key == 'is_forward') {
      newMail = {
        ...currentMail,
        subject: `转发: ${currentMail.subject}`,
        to_info: [],
        cc_info: [],
        detail: {
          ...currentMail?.detail,
          content: FormContent,
        },
        is_forward: true,
      }
    }
    if (newMail) {
      onEdit(newMail)
    }
  }

  // 删除邮件
  const onDelMail = async (items) => {
    const list = items.filter(Boolean)
    if (!list.length) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }
    const ids = list.map((e) => e.uid)
    const folder = list?.length === 1 ? list[0].folder : currentFolder.folder

    if (folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }

    Modal.confirm({
      title: '提示',
      content: '是否确定删除?',
      className: 'simpleModal',
      onOk: async () => {
        if (folder === 'Deleted') {
          const { code } = await request.post('/api/mail/delete', { folder: 'Deleted', uids: ids })
          if (code === 200) {
            Message.success('邮件已彻底删除')
          }
        } else {
          if (folder === 'Drafts') {
            // 判断是否是定时邮件
            const isSchedule = mailList?.list?.filter(
              (e) => ids.includes(e.uid) && Array.isArray(e.flags) && e.flags.includes('Draft')
            )
            if (isSchedule?.length > 0) {
              return Message.error('请先取消定时邮件后再进行删除操作')
            }
          }

          // 其他文件夹：移动到垃圾箱
          const { code } = await request.post('/api/mail/move', {
            uids: ids,
            from_folder: folder,
            to_folder: 'Deleted',
          })
          if (code === 200) {
            Message.success('已移入垃圾箱')
          }
        }
        // 刷新邮件列表
        getMailList({
          folder: currentFolder.folder,
          keyword: searchWord,
          filter: filterKeys,
          page: 1,
          size: pageSize,
        })
      },
    })
  }

  // 移动邮件
  const onMoveMail = async ({ uids, from, to }) => {
    if (from === 'Star') {
      return Message.warning('请选择非星标邮件')
    }
    const list = uids.filter(Boolean)
    if (!list.length) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }
    if (from === to) {
      return Message.warning({
        content: '请选择不同的文件夹',
        showIcon: true,
        position: 'bottom',
      })
    }
    const { code } = await request.post('/api/mail/move', {
      from_folder: from,
      to_folder: to,
      uids: list,
    })
    if (code !== 200) {
      Message.error('移动失败')
      return
    }
    Message.success('移动成功')
    // 刷新邮件列表
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      filter: filterKeys,
      page: 1,
      size: pageSize,
    })
  }

  // 标记为
  const onFlagMail = ({ uids, from, to }) => {
    const key = to.split('_')
    const item = {
      uids,
      folder: from,
      type: Number(key[1]),
    }

    if (key[0] === 'Seen') {
      onRead(item)
    }

    if (key[0] === 'Flagged') {
      onStar(item)
    }
  }

  // 标记为列表设置
  const onChangeMailFlag = (selected, list = mailList?.list) => {
    const selectedMails = (list || []).filter((item) => selected.includes(item.uid))

    // 完全没有选中，退出
    if (selected.length === 0 && selectedMails.length === 0) {
      return
    }

    const allHasSeen = selectedMails.every((mail) => Array.isArray(mail.flags) && mail.flags.includes('Seen'))
    const allHasFlagged = selectedMails.every((mail) => Array.isArray(mail.flags) && mail.flags.includes('Flagged'))
    const someHasSeen = selectedMails.some((mail) => Array.isArray(mail.flags) && mail.flags.includes('Seen'))
    const someHasFlagged = selectedMails.some((mail) => Array.isArray(mail.flags) && mail.flags.includes('Flagged'))
    let readList = []

    if (allHasSeen) {
      readList = [{ flag: 'Seen', key: 2, title: '未读邮件' }]
    } else {
      if (someHasSeen) {
        readList = [
          { flag: 'Seen', key: 1, title: '已读邮件' },
          { flag: 'Seen', key: 2, title: '未读邮件' },
        ]
      } else {
        readList = [{ flag: 'Seen', key: 1, title: '已读邮件' }]
      }
    }

    let starList = []
    if (allHasFlagged) {
      starList = [{ flag: 'Flagged', key: 2, title: '取消星标' }]
    } else {
      if (someHasFlagged) {
        starList = [
          { flag: 'Flagged', key: 1, title: '星标邮件' },
          { flag: 'Flagged', key: 2, title: '取消星标' },
        ]
      } else {
        starList = [{ flag: 'Flagged', key: 1, title: '星标邮件' }]
      }
    }

    const newFlags = [...readList, { title: '分割线', flag: 'divider' }, ...starList]
    setFlagList(newFlags)
  }

  // 标记已读
  const onRead = async (item) => {
    if (item.folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }

    const list = item.uids.filter(Boolean)
    if (list.length == 0) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }
    if (item.folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }
    const params = {
      uids: list,
      folder: item.folder,
      status: 'Seen',
      type: item.type, // 1:未读 2:已读
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      const newList = [...mailList.list]
      const { uids } = params

      newList.forEach((mailItem) => {
        if (!uids.includes(mailItem.uid)) return

        let flags = mailItem.flags || []
        if (item.type === 1) {
          if (!flags.includes('Seen')) {
            flags = [...flags, 'Seen']
          }
        }
        if (item.type === 2) {
          flags = flags.filter((f) => f !== 'Seen')
        }

        mailItem.flags = flags
      })
      setMailList((prev) => {
        if (!Array.isArray(uids) || uids.length === 0) return prev
        return {
          ...prev,
          list: newList,
        }
      })

      getUnReadTotal()

      onChangeMailFlag(params.uids, newList)
    }
  }

  // 标记星标
  const onStar = async (item) => {
    if (item.folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }

    const list = item.uids.filter(Boolean)
    if (list?.length == 0) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }

    const params = {
      uids: list,
      folder: item.folder,
      status: 'Flagged',
      type: item.type, // 1:添加 2:取消
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      const uids = params.uids
      let newList = [...mailList.list]
      const uidSet = new Set(uids)

      // 遍历更新flags
      newList = newList.map((mailItem) => {
        if (!uidSet.has(mailItem.uid)) return mailItem

        let flags = mailItem.flags || []
        if (item.type === 1) {
          if (!flags.includes('Flagged')) {
            flags = [...flags, 'Flagged']
          }
        } else {
          flags = flags.filter((item) => item !== 'Flagged')
        }
        return { ...mailItem, flags }
      })

      // 当前是星标文件夹，把本次操作的全部uid过滤移除
      if (currentFolder.folder === 'Star') {
        newList = newList.filter((item) => !uidSet.has(item.uid))
      }

      setMailList((prev) => {
        if (!Array.isArray(uids) || uids.length === 0) return prev

        return {
          ...prev,
          list: newList,
        }
      })

      if (currentMail) {
        setCurrentMail((prev) => {
          let newItem = { ...prev }
          let flags = newItem?.flags || []
          if (item.type === 1) {
            flags.push('Flagged')
          } else {
            flags = flags?.filter((item) => item !== 'Flagged')
          }
          newItem.flags = flags
          if (currentFolder.folder === 'Star') {
            newItem = null
          }
          return newItem
        })
      }

      getUnReadTotal()

      onChangeMailFlag(params.uids, newList)
    }
  }

  // 监控内容变化
  const onChangeNewMail = (item) => {
    if (currentFolder.key === 'compose') {
      setNewMailInfo(item)
    }
  }

  // 清空联系人
  const onClearContact = async (params) => {
    const { code, msg } = await request.post('/api/user/contact/clear', params)
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 删除最近联系人
  const onDeleteContact = async (params) => {
    const { code, msg } = await request.post('/api/user/contact/delete', { email: params?.email, prefix: params?.prefix })
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 添加编辑最近联系人
  const onEditContact = async (params) => {
    const { code, msg } = await request.post('/api/user/contact/save', params)
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 获取用户列表
  const getUserList = async () => {
    const { code, data, msg } = await request.post('/api/user/list')
    if (code === 200) {
      const newData = {
        list: (data.list || []).map((e) => ({ ...e, is_me: e.email === currentAccountId })),
        total: data.total,
      }
      setUserList(newData)
    } else {
      Message.error(msg)
    }
  }

  // 获取联系人
  const getContactList = async (params) => {
    const { code, data, message } = await request.post('/api/user/contact/list', { prefix: params?.prefix })
    if (code == 200) {
      const list = (data.list || []).map((e) => ({ ...e, full_name: e.name }))
      if (params?.prefix === 'user_sent') {
        setRecentlyList(list)
      }
      if (params?.prefix === 'user_contact') {
        setContactList(list)
      }
    } else {
      Message.error(message)
    }
  }

  // 获取邮件详情
  const getCurrentMail = async (item) => {
    const params = {
      uid: item.uid,
      folder: item.folder,
    }
    setMailLoading(true)
    const { code, data, msg } = await request.post('/api/mail/detail', params, {
      headers: { 'X-Client-Host': window.location.host },
    })

    if (code === 200) {
      const newData = {
        ...data,
        attachments: data?.attachments?.map((e) => ({
          ...e,
          file_type: getFileType(e.file_type),
        })),
      }
      let content = transHtmlAttrs(newData.content, 'src', 'data-href')
      newData.content = content.replace(
        /<span style="color: rgb\(149, 157, 166\);">(&lt;([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)&gt;)<\/span>/g,
        (match, fullInner, email) => {
          return `&lt;<a href="mailto:${email}" target="_blank" rel="noopener noreferrer" data-mail-email="${email}" title="给TA写信">${email}</a>&gt;`
        }
      )

      setCurrentMail({ ...item, detail: newData })

      // 标记已读
      if (!item?.flags || !item.flags?.includes('Seen')) {
        onRead({ uids: [item.uid], folder: item.folder, type: 1 })
      } else {
        // 设置标记为内容
        onChangeMailFlag([item.uid], mailList.list)
      }

      // 草稿编辑
      if (item.folder === 'Drafts' && item.schedule === '0001-01-01T00:00:00Z') {
        const newItem = {
          ...item,
          detail: newData,
          to_info: item?.to_info?.map((e) => ({ label: e.name, value: e.email })),
          cc_info: item?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
        }
        onEdit(newItem)
      }
    } else {
      Message.error(msg)
    }

    setSelectedRowKeys([])
    setMailLoading(false)
  }

  // 获取邮件列表
  const getMailList = async (item, isRefresh) => {
    if (!isRefresh && !refreshCount) {
      setSelectedRowKeys([]) // 清空选中
      setCurrentMail(null) // 清空当前邮件
      setListLoading(true)
    }

    let url = '/api/mail/list'
    let params = {
      ...item,
    }
    if (item.folder === 'Star') {
      url = '/api/mail/star-list'
      params = {
        filter: item.filter,
        keyword: item.keyword,
      }
    }

    let { code, data, msg } = await request.post(url, params)
    if (code === 200) {
      const list = data?.list || []

      if (item.page === 1) {
        setMailList({
          ...data,
          list,
        })
      } else {
        setMailList({
          ...data,
          list: [...mailList.list, ...list],
        })
      }

      // 刷新邮件列表
      InboxRefresh()
    } else {
      Message.error(msg)
    }
    setListLoading(false)
  }

  // 获取未读邮件数量
  const getUnReadTotal = async () => {
    const { code, data, msg } = await request.post('/api/mail/unread-total')
    if (code === 200) {
      const list = folderList.map((item) => ({ ...item, total: data[item.folder] || 0 }))
      setFolderList(list)
    } else {
      Message.error(msg)
    }
  }

  // 切换选中邮件
  const onCutMail = (record, key) => {
    const list = mailList?.list || []
    const index = list.findIndex((e) => e.uid === record.uid)
    if (key === 'prev') {
      setCurrentMail(list[index - 1])
    } else if (key === 'next') {
      setCurrentMail(list[index + 1])
    }
  }

  // 写信
  const onEdit = (record) => {
    setCurrentMail(null)
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      return Message.warning('写邮件页已打开，请先关闭')
    }

    let compose = { key: 'compose', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> }
    // 编辑-草稿
    if (record?.uid) {
      compose.title = record.subject
    }

    setNewMailInfo(record)
    setCurrentFolder(compose)
    setFolderList((prev) => [compose, ...prev])
  }

  // 关闭写信
  const onCloseEdit = (key) => {
    const record = key ? folderList.find((item) => item.key === key) : false
    setCurrentFolder(record || menuList[0])
    setFolderList((prev) => prev.filter((item) => item.key !== 'compose'))

    // 清空草稿信息
    setNewMailInfo(null)
  }

  // 筛选事件
  const onSelectFilter = (filter) => {
    setFilterKeys(filter)
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      filter: filter,
      page: 1,
      size: pageSize,
    })
  }

  // 自动刷新，获取收件箱邮件
  const InboxRefresh = () => {
    setRefreshCount((prevCount) => {
      // 1分钟刷新一次，20分钟以后，5分钟刷新一次
      let count = prevCount >= 20 ? 5 * 60 * 1000 : 60 * 1000

      // 最大刷新次数：50次:20个10分钟+6个5分钟
      if (prevCount >= 50) {
        if (timerRef.current) clearTimeout(timerRef.current)
        return
      }
      // 先清除已有定时器
      if (timerRef.current) clearTimeout(timerRef.current)

      const params = {
        folder: currentFolder.folder,
        keyword: searchWord,
        filter: filterKeys,
        page: 1,
        size: pageSize,
      }

      timerRef.current = setTimeout(async () => {
        await getMailList(params, true)
        //本次请求完成，再开启下一轮计时
        setRefreshCount((prev) => prev + 1)
        InboxRefresh()
      }, count)
      return prevCount
    })
  }

  //   滚动到顶部
  const scrollToTop = () => {
    const scrollContainer = tableRef?.current?.querySelector('.arco-table-body')
    if (scrollContainer) {
      // 使用平滑滚动回到顶部
      scrollContainer.scrollTo({
        top: 0,
        behavior: 'smooth',
      })
    }
  }

  // 滚动加载
  const totalPages = Math.ceil(mailList?.total / pageSize)
  const throttledScrollHandler = useMemo(
    () =>
      throttle((e) => {
        const { scrollTop, scrollHeight, clientHeight } = e.target
        const distanceToBottom = scrollHeight - scrollTop - clientHeight

        if (distanceToBottom <= 200 && !listLoading) {
          let currentPage = Math.ceil(mailList.list.length / pageSize)
          if (currentPage < totalPages) {
            getMailList({
              folder: currentFolder.folder,
              keyword: searchWord,
              filter: filterKeys,
              page: currentPage + 1,
              size: pageSize,
            })
          }
        }
      }, 500),
    [totalPages, mailList.list, searchWord, currentFolder.folder, filterKeys, listLoading]
  )

  const onScroll = useCallback(
    (e) => {
      throttledScrollHandler(e)
    },
    [throttledScrollHandler]
  )

  // 监听邮件滚动加载事件
  const bindRef = useRef(false)
  useEffect(() => {
    const scrollContainer = tableRef.current
    if (!scrollContainer) return

    // 避免重复绑定
    if (!bindRef.current) {
      scrollContainer.addEventListener('scroll', onScroll)
      bindRef.current = true
    }

    return () => {
      scrollContainer.removeEventListener('scroll', onScroll)
      throttledScrollHandler.cancel()
      bindRef.current = false
    }
  }, [onScroll, throttledScrollHandler])

  // 搜索邮件
  const onSearch = (val) => {
    let folder = currentFolder.folder
    if (folder === 'DRAFTS') {
      const item = folderList.find((item) => item.key === 'inbox')
      folder = item.folder
      setCurrentFolder(item)
    }

    setCurrentMail(null)
    setSelectedRowKeys([])

    getMailList({
      folder: folder,
      keyword: val,
      page: 1,
      filter: filterKeys,
    })

    scrollToTop()
  }

  // 监控搜索事件
  useEffect(() => {
    if (registerMethod) {
      registerMethod('onSearch', onSearch)
    }
  }, [onSearch, registerMethod])

  // 监听选中邮件
  useEffect(() => {
    const init = async () => {
      if (!currentMail?.uid) return
      await getCurrentMail(currentMail)
    }
    init()
  }, [currentMail?.uid])

  // 监听切换目录 / 搜索 / 筛选变化，重新加载
  useEffect(() => {
    const init = () => {
      if (!currentFolder?.folder) return
      if (currentFolder.key !== 'compose') {
        getMailList({
          folder: currentFolder.folder,
          keyword: searchWord,
          filter: filterKeys,
          page: 1,
          size: pageSize,
        })
      }
    }
    init()
  }, [currentFolder, newMailInfo])

  // 立即加载第一页邮件
  useEffect(() => {
    const init = () => {
      getUnReadTotal()

      setCurrentFolder(folderList[0])
      getUserList()
      getContactList({ prefix: 'user_sent' })
      getContactList({ prefix: 'user_contact' })
    }
    init()
  }, [])

  // 统一上下文数据
  const mailContent = useMemo(
    () => ({
      baseUrl, // 基础URL
      userInfo, // 用户信息
      setSearchWord, // 设置搜索词

      folderList, // 目录菜单
      setFolderList, // 设置目录菜单
      currentFolder, // 当前文件夹
      setCurrentFolder, // 设置当前文件夹
      onEdit, // 写信
      onCloseEdit, // 关闭写信

      tableRef, // 表格容器
      mailList, // 邮件列表
      listLoading, // 列表加载中
      filterList, // 筛选列表
      filterKeys, // 筛选参数
      onSelectFilter, // 筛选事件
      isTable, // 是否表格模式
      setIsTable, // 设置表格模式
      isMove, // 是否移动模式
      setIsMove, // 设置是否移动模式

      selectedRowKeys, // 选中邮件
      setSelectedRowKeys, // 设置选中邮件

      currentMail, // 当前邮件
      setCurrentMail, // 设置当前邮件
      mailLoading, // 邮件加载中
      onUnSchedule, // 取消定时

      onCutMail, // 切换邮件
      onStar, // 添加星标
      onRead, // 标记已读
      onDelMail, // 删除邮件
      onMoveMail, // 移动邮件
      onReplyForward, //回复或转发邮件

      moveList, // 可移动文件夹

      contactList, // 联系人
      onEditContact, // 编辑联系人
      flagList, // 标记列表
      onChangeMailFlag, // 监控标记邮件
      onFlagMail, // 标记邮件
    }),
    [
      folderList,
      currentFolder,
      onEdit,
      onCloseEdit,
      onSelectFilter,
      mailList,
      selectedRowKeys,
      onChangeMailFlag,
      flagList,
      filterKeys,
    ]
  )

  return (
    <MailProvider value={mailContent}>
      <Layout className='pr-4!'>
        <Layout.Sider width={220} className='bg-transparent!'>
          {/* 左列：文件夹导航 */}
          <Menu />
        </Layout.Sider>
        {currentFolder?.key === 'compose' ? (
          // 写信
          <WriteMail
            key={newMailInfo?.uid || '0'}
            detail={newMailInfo}
            userList={userList?.list || []}
            onClose={onCloseEdit} // 关闭写邮件页
            onSend={onSend} // 发邮件或存草稿
            recentlyList={recentlyList} // 最近联系人
            contactList={contactList} //我的联系人
            onEditContact={onEditContact} // 添加编辑联系人
            onDeleteContact={onDeleteContact} // 删除联系人
            onClearContact={onClearContact} // 清空联系人
            onChange={onChangeNewMail} // 监控内容变化
          />
        ) : (
          //   右列：邮件列表 邮件详情
          <Content />
        )}
      </Layout>
    </MailProvider>
  )
}
export default MailLayout

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useOutletContext } from 'react-router'

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

import {
  Button,
  Card,
  Divider,
  Dropdown,
  Input,
  Layout,
  Menu,
  Message,
  Modal,
  Popover,
  Space,
  Spin,
  Table,
  Typography,
} from '@arco-design/web-react'
import {
  IconArrowLeft,
  IconAttachment,
  IconCheck,
  IconClockCircle,
  IconClose,
  IconDelete,
  IconDown,
  IconEdit,
  IconEmail,
  IconEye,
  IconFile,
  IconLayout,
  IconLeft,
  IconMenu,
  IconPlus,
  IconRedo,
  IconReply,
  IconRight,
  IconSearch,
  IconSend,
  IconSort,
  IconStar,
  IconToBottom,
} from '@arco-design/web-react/icon'

import request from 'src/api/request'

import AvatarImage from 'src/components/AvatarImage'
import WriteMail from 'src/components/WriteMail'

import IconAudio from 'src/assets/file_aduio.svg'
import IconExcel from 'src/assets/file_excel.svg'
import IconImage from 'src/assets/file_image.svg'
import IconPdf from 'src/assets/file_pdf.svg'
import IconPpt from 'src/assets/file_ppt.svg'
import IconText from 'src/assets/file_text.svg'
import IconVideo from 'src/assets/file_video.svg'
import IconWord from 'src/assets/file_word.svg'
import IconZip from 'src/assets/file_zip.svg'

import IconMoveFolder from 'src/assets/mail_move_folder.svg'
import IconMailNormal from 'src/assets/mail_normal.svg'
import IconMailOpen from 'src/assets/mail_open.svg'
import IconMailReply from 'src/assets/mail_reply.svg'
import IconSent from 'src/assets/mail_sent.svg'
import IconStarUnselect from 'src/assets/mail_star.svg'
import IconStarSelect from 'src/assets/mail_star_open.svg'
import IconMailTimer from 'src/assets/mail_timer.svg'

import { flatTree, getFileType, throttle } from 'src/utils/index'

// 左侧文件夹
const menuList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail className='text-lg!' /> },
  { key: 'star', folder: 'Star', title: '星标邮件', icon: <IconStar className='text-lg!' /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend className='text-lg!' /> },
  { key: 'drafts', folder: 'Drafts', title: '草稿箱', icon: <IconFile className='text-lg!' /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete className='text-lg!' /> },
]
// 移动文件夹
const moveList = [
  { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <IconEmail className='text-lg!' /> },
  { key: 'sent', folder: 'Sent', title: '已发送', icon: <IconSend className='text-lg!' /> },
  { key: 'delete', folder: 'Deleted', title: '垃圾箱', icon: <IconDelete className='text-lg!' /> },
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
const FlagList = (flags) => {
  // 1:添加 2:取消
  const list = [
    { flag: 'Seen', key: 2, title: '未读邮件' },
    { flag: 'Flagged', key: 2, title: '取消星标' },
  ]
  if (!flags?.includes('Flagged')) {
    list[1].title = '星标邮件'
    list[1].key = 1
  }
  return list.map((e) => <Menu.Item key={e.flag + '_' + e.key}>{e.title}</Menu.Item>)
}

// 邮件图标
const showMailIcon = (flags) => {
  const flagArr = Array.isArray(flags) ? flags : []
  if (flagArr.includes('Draft')) return <IconMailTimer />
  if (flagArr.includes('Answered')) return <IconMailReply />
  if (flagArr.includes('Seen')) return <IconMailOpen />
  return <IconMailNormal />
}

const MailLayout = () => {
  const { currentAccountId, userInfo } = useOutletContext()

  const [userList, setUserList] = useState({}) // 用户列表
  const [recentlyList, setRecentlyList] = useState([]) // 最近联系人
  const [contactList, setContactList] = useState([]) // 联系人

  const [folderList, setFolderList] = useState(menuList) // 文件夹
  const [currentFolder, setCurrentFolder] = useState({}) // 当前文件夹
  const [searchWord, setSearchWord] = useState('') // 搜索

  const [loading, setLoading] = useState(false) // 加载中
  const [mailList, setMailList] = useState([]) // 邮件列表
  const [total, setTotal] = useState(0) // 邮件总数
  const [selectedRowKeys, setSelectedRowKeys] = useState([]) // 选中的邮件

  const [currentLoading, setCurrentLoading] = useState(false) // 当前邮件加载中
  const [currentMail, setCurrentMail] = useState(null) // 当前邮件
  const [writeMail, setWriteMail] = useState(null) // 写邮件
  const [newWriteMail, setNewWriteMail] = useState(null) // 新的写邮件
  const [isTable, setIsTable] = useState(() => localStorage.getItem('isTable') === 'true') // 表格模式

  const [filterKeys, setFilterKeys] = useState(['all', 'date_desc']) // 已筛选参数

  const [_, setRefreshCount] = useState(0) // 刷新次数
  const timerRef = useRef(null) // 定时器
  const tableRef = useRef(null) // 表格
  const pageSize = 25 // 每页显示的邮件数(不能低于22，否则滚动条不出现，无法实现滚动加载更多)
  const totalPages = Math.ceil(total / pageSize) // 总页数

  // 筛选后展示的名称
  const filterNames = filterKeys
    .map((key) => {
      const item = flatTree(filterList)
        .filter((item) => !['all', 'date_desc'].includes(item.value))
        .find((e) => e.value === key)
      return item?.label
    })
    .filter(Boolean)

  // 切换表格模式
  const onChangeMode = () => {
    setIsTable(!isTable)
    setCurrentMail(null)

    localStorage.setItem('isTable', !isTable)
  }

  // 切换选中邮件
  const onCutMail = (record, key) => {
    const index = mailList.findIndex((e) => e.uid === record.uid)
    if (key === 'prev') {
      onMailDetail(mailList[index - 1])
    } else if (key === 'next') {
      onMailDetail(mailList[index + 1])
    }
  }

  // 关闭写邮件页，返回收件箱
  const onClickCompose = (key) => {
    setNewWriteMail(null)
    setWriteMail(null)

    setFolderList((prev) => prev.filter((item) => item.key !== 'compose'))
    setTimeout(() => {
      const item = folderList.find((item) => item.key === key)
      loadMailList(item.key)
    }, 300)
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

  // 写邮件
  const onWriteMail = async (key, mailData) => {
    setWriteMail(null)
    setNewWriteMail(null)
    // 草稿页已打开
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      setCurrentFolder(folderList.find((item) => item.key === 'compose'))
      return Message.warning('写邮件页已打开，请先关闭')
    }

    getContactList({ prefix: 'user_sent' })

    let composeItem = { key: 'compose', folder: 'DRAFTS', title: '草稿', icon: <IconFile className='text-lg!' /> }
    if (key !== 'new') {
      composeItem.title = mailData?.subject
      setWriteMail(mailData)
    }
    setFolderList((prev) => [composeItem, ...prev])
    setCurrentFolder(composeItem)

    setSelectedRowKeys([])
    setMailList([])
    setSearchWord('')
  }

  // 点击导航栏
  const loadMailList = async (key) => {
    setRefreshCount(0)
    setSelectedRowKeys([])
    setMailList([])
    setTotal(0)
    setCurrentMail(null)
    setSearchWord('')

    // 当前文件夹
    const item = folderList.find((item) => item.key === key)
    setCurrentFolder(item)
    // 草稿页
    if (key === 'compose') {
      setWriteMail(newWriteMail || writeMail)
      return
    }

    // 加载邮件列表
    getMailList({
      folder: item.folder,
      keyword: '',
      page: 1,
      filter: ['all', 'date_desc'],
      isRefresh: false,
    })
    InboxRefresh()
  }

  // 自动刷新，获取收件箱邮件
  const InboxRefresh = () => {
    setRefreshCount((prevCount) => {
      // 1分钟刷新一次，10分钟以后，10分钟刷新一次
      let count = prevCount >= 10 ? 60 * 1000 : 10 * 60 * 1000 // 60秒和10分钟

      // 最大刷新次数
      if (prevCount >= 30) {
        if (timerRef.current) clearTimeout(timerRef.current)
        return
      }
      // 先清除已有定时器
      if (timerRef.current) clearTimeout(timerRef.current)

      timerRef.current = setTimeout(async () => {
        await getMailList({
          folder: 'INBOX',
          keyword: '',
          page: 1,
          filter: filterKeys,
          isRefresh: true,
        })
        //本次请求完成，再开启下一轮计时
        setRefreshCount((prev) => prev + 1)
        InboxRefresh()
      }, count)
      return prevCount
    })
  }

  // 预览附件
  const onPreviewAttachment = (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
      file_name: item.file_name,
      file_type: item.file_type,
    }
    const jsonString = JSON.stringify(params)
    const base64Str = btoa(encodeURIComponent(jsonString))
    window.open(`/web/preview?preview=${base64Str}`, '_blank')
  }

  // 下载附件
  const onDownloadAttachment = async (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
    }
    const res = await request.post('/api/mail/download', params, {
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(res)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', item.file_name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // 获取邮件详情
  const onMailDetail = async (item, e) => {
    // 排除干扰点击
    const targetElement = e?.target

    const isCheckboxClick = targetElement
      ? targetElement?.classList.contains('arco-checkbox') ||
        targetElement?.classList.contains('arco-checkbox-input') ||
        targetElement?.closest('.arco-checkbox')
      : false

    // 排除非跳转项
    if (isCheckboxClick) return

    setCurrentLoading(true)
    const params = {
      uid: item.uid,
      folder: item.folder,
    }

    const { code, data, msg } = await request.post('/api/mail/detail', params)
    if (code === 200) {
      const newData = {
        ...data,
        attachments: data?.attachments?.map((e) => ({
          ...e,
          file_type: getFileType(e.file_type),
        })),
      }

      newData.content = transHtml(newData.content)
      setCurrentMail({ ...item, detail: newData })
      if (item.folder === 'Drafts' && item.schedule === '0001-01-01T00:00:00Z') {
        const newItem = {
          ...item,
          detail: newData,
          to_info: item?.to_info?.map((e) => ({ label: e.name, value: e.email })),
          cc_info: item?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
        }
        onWriteMail('rewrite', newItem)
      }

      // 标记已读
      if (!item?.flags || !item.flags?.includes('Seen')) {
        onRead(item, 1)
      }
    } else {
      Message.error(msg)
    }
    setCurrentLoading(false)
  }

  // 转换HTML内容
  const transHtml = (html) => {
    const parser = new DOMParser()
    const doc = parser.parseFromString(html, 'text/html')
    const divs = doc.body.querySelectorAll('div')

    divs.forEach((div) => {
      const p = document.createElement('p')
      for (let attr of div.attributes) {
        p.setAttribute(attr.name, attr.value)
      }
      while (div.firstChild) {
        p.appendChild(div.firstChild)
      }
      div.parentNode.replaceChild(p, div)
    })
    return doc.body.innerHTML
  }

  // 搜索邮件
  const onSearch = async (val) => {
    setCurrentMail(null)
    setSelectedRowKeys([])

    getMailList({
      folder: currentFolder.folder,
      keyword: val,
      page: 1,
      filter: filterKeys,
      isRefresh: false,
    })

    scrollToTop()
  }
  // 获取邮件数据
  const getMailList = async (keys) => {
    const { isRefresh, ...item } = keys

    setFilterKeys(keys.filter)
    // 加载邮件列表
    setLoading(true)
    let url = '/api/mail/list'
    let params = {
      ...item,
      size: pageSize,
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
      const list = (data?.list || []).map((e) => {
        const to_reply = e?.to_info?.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ')
        const cc_reply = e.cc ? e?.cc_info?.map((t) => t.name + ' &lt;' + t.email + '&gt;').join(', ') : ''

        return {
          ...e,
          to_reply,
          cc_reply,
        }
      })

      if (item.folder === 'INBOX') {
        let inbox = list.filter((e) => !e?.flags?.includes('Seen'))?.length || 0
        setFolderList((prev) =>
          prev.map((e) => {
            if (e.folder === item.folder) {
              return { ...e, total: inbox }
            }
            return e
          })
        )
      }
      // 刷新
      if (!isRefresh) {
        if (item.page === 1) {
          setMailList(list)
        } else {
          setMailList([...mailList, ...list])
        }
        setTotal(data?.total || 0)
      }
    } else {
      Message.error(msg)
    }

    setLoading(false)
  }

  // 删除邮件
  const onDelMail = async (ids) => {
    Modal.confirm({
      title: '提示',
      content: '是否确定删除?',
      className: 'simpleModal',
      onOk: async () => {
        setSelectedRowKeys([])

        if (currentFolder.folder === 'Deleted') {
          const { code } = await request.post('/api/mail/delete', { folder: 'Deleted', uids: ids })
          if (code === 200) {
            Message.success('邮件已彻底删除')
          }
        } else {
          if (currentFolder.folder === 'Drafts') {
            // 判断是否是定时邮件
            const isSchedule = mailList.filter((e) => ids.includes(e.uid) && Array.isArray(e.flags) && e.flags.includes('Draft'))
            if (isSchedule?.length > 0) {
              return Message.error('请先取消定时邮件后再进行删除操作')
            }
          }

          // 其他文件夹：移动到垃圾箱
          const { code } = await request.post('/api/mail/move', {
            uids: ids,
            from_folder: currentFolder.folder,
            to_folder: 'Deleted',
          })
          if (code === 200) {
            Message.success('已移入垃圾箱')
          }
        }
        setCurrentMail(null)
        loadMailList(currentFolder.key)
      },
    })
  }

  // 移动邮件
  const onMoveMail = async (e) => {
    const { code } = await request.post('/api/mail/move', {
      from_folder: currentFolder.folder,
      to_folder: e,
      uids: [currentMail.uid],
    })
    if (code !== 200) {
      Message.error('移动失败')
      return
    }
    Message.success('移动成功')
    loadMailList(currentFolder.key)
    setCurrentMail(null)
  }

  // 标记邮件
  const onFlagMail = async (e) => {
    const key = e.split('_')
    if (key[0] === 'Seen') {
      onRead(currentMail, Number(key[1]))
    }

    if (key[0] === 'Flagged') {
      onStar(currentMail)
    }
  }

  // 邮件原内容
  const FormContent = `<p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><br></p>
  <p style="line-height: 1;">
  <span style="color: rgb(140, 140, 140);">— </span>
  <span style="font-size: 12px; color: rgb(140, 140, 140);">原始邮件</span>
  <span style="color: rgb(140, 140, 140);"> ————————————</span>
  </p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 12px;">发件人：</span>
  <span style="font-size: 12px;">${currentMail?.from_info?.name} &lt;${currentMail?.from}&gt; </span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>发件时间：</span>
  <span style="font-size: 12px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>收件人：</span>
  <span style="font-size: 12px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>主题：</span>
  <span style="font-size: 12px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  const FormContentCc = `<p style="line-height: 1;"><br></p>
  <p style="line-height: 1;"><br></p>
  <p style="line-height: 1;">
  <span style="color: rgb(140, 140, 140);">— </span>
  <span style="font-size: 12px; color: rgb(140, 140, 140);">原始邮件</span>
  <span style="color: rgb(140, 140, 140);"> ————————————</span>
  </p>
  <blockquote>
  <span style="color: rgb(140, 140, 140); font-size: 12px;">发件人：</span>
  <span style="font-size: 12px;">${currentMail?.from_info?.name} &lt;${currentMail?.from}&gt;</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>发件时间：</span>
  <span style="font-size: 12px;">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>收件人：</span>
  <span style="font-size: 12px;">${currentMail?.to_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>抄送：</span>
  <span style="font-size: 12px;">${currentMail?.cc_reply}</span>
  <span style="color: rgb(140, 140, 140); font-size: 12px;"><br>主题：</span>
  <span style="font-size: 12px;">${currentMail?.subject}</span>
  </blockquote>${currentMail?.detail?.content || ''}`

  // 回复邮件
  const onReply = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `回复: ${currentMail.subject}`,
      to_info: currentMail.to_info.map((e) => ({ label: e.name, value: e.email })),
      cc_info: currentMail?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
      is_reply: true,
    }

    if (currentFolder.key === 'inbox') {
      newMail.to_info = [{ label: currentMail.from_info.name, value: currentMail.from }]
    }
    onWriteMail('reply', newMail)
  }

  // 转发邮件
  const onForward = () => {
    if (!currentMail) return

    const newMail = {
      ...currentMail,
      subject: `转发: ${currentMail.subject}`,
      to_info: [],
      cc_info: [],
      detail: {
        content: currentMail?.cc ? FormContentCc : FormContent,
      },
      is_forward: true,
    }

    onWriteMail('forward', newMail)
  }

  // 标记星标邮件
  const onStar = async (item) => {
    const type = item?.flags?.includes('Flagged') ? 2 : 1 // 1:添加 2:取消
    const params = {
      uid: item.uid,
      folder: item.folder,
      status: 'Flagged',
      type,
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      setMailList((prev) => {
        let newList = [...prev]
        const index = newList.findIndex((item) => item.uid === params.uid)
        let flags = newList[index]?.flags || []
        if (type === 1) {
          flags.push('Flagged')
        } else {
          flags = flags?.filter((item) => item !== 'Flagged')
        }
        newList[index] = {
          ...newList[index],
          flags,
        }
        if (currentFolder.folder === 'Star') {
          newList = newList?.filter((item) => item.uid !== params.uid)
        }
        return newList
      })

      setCurrentMail((prev) => {
        let newItem = { ...prev }
        let flags = newItem?.flags || []
        if (type === 1) {
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
  }

  // 标记已读
  const onRead = async (item, type = 1) => {
    const params = {
      uid: item.uid,
      folder: item.folder,
      status: 'Seen',
      type,
    }
    const { code } = await request.post('/api/mail/status', params)
    if (code === 200) {
      setMailList((prev) => {
        const newList = [...prev]
        const index = newList.findIndex((item) => item.uid === params.uid)
        let flags = newList[index]?.flags || []
        if (type === 1) {
          flags.push('Seen')
        }
        if (type === 2) {
          flags = flags?.filter((item) => item !== 'Seen')
        }
        newList[index] = {
          ...newList[index],
          flags,
        }
        return newList
      })

      setFolderList((prev) =>
        prev.map((item) => {
          if (item.folder === 'INBOX') {
            return { ...item, total: item.total - 1 }
          }
          return item
        })
      )
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

  // 标记已读
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
        keyword: '',
        page: 1,
        filter: filterKeys,
        isRefresh: false,
      })
    }
  }
  // 发送邮件&草稿
  const onSend = async (type, form, html, fileList, detail, customTime, setLoading) => {
    const values = form.getFieldsValue()
    if (!values.to_info || !values.subject) {
      Message.warning('请填写收件人和主题')
      return
    }

    const formData = new FormData()
    const to = values.to_info.map((e) => e.value)
    const cc = values?.cc_info?.map((e) => e.value) || ''

    formData.append('to', to)
    formData.append('cc', cc)
    formData.append('subject', values.subject)
    formData.append('content', html)
    if (detail?.uid) {
      formData.append('uid', detail.uid)
    }
    if (customTime) {
      formData.append('x-schedule-send', customTime)
    }

    // 回复
    if (detail?.is_reply) {
      const references = detail.references + ' ' + detail?.message_id
      formData.append('in_reply_to', detail?.message_id)
      formData.append('references', references)
    }

    // 转发
    if (detail?.is_forward) {
      const references = detail.references + ' ' + detail?.message_id
      formData.append('references', references)
    }

    const partIds = []
    fileList.forEach((file) => {
      if (file?.part_id) {
        partIds.push(file.part_id)
      } else {
        formData.append('files', file?.originFile)
      }
    })
    if (partIds.length > 0) {
      formData.append('part_ids', partIds.join(','))
    }

    setLoading(true)
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
        onClickCompose(type === 'Sent' ? 'sent' : 'drafts')
      } else {
        Message.error(msg)
      }
    }
    setLoading(false)
  }

  // 初始加载邮件列表
  useEffect(() => {
    const init = async () => {
      await loadMailList('inbox')
      getContactList({ prefix: 'user_contact' })
      getUserList()
    }
    init()
  }, [])

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
  const throttledScrollHandler = useMemo(
    () =>
      throttle((e) => {
        const { scrollTop, scrollHeight, clientHeight } = e.target
        const distanceToBottom = scrollHeight - scrollTop - clientHeight

        if (distanceToBottom <= 300 && !loading) {
          let currentPage = Math.ceil(mailList.length / pageSize)
          if (currentPage < totalPages) {
            getMailList({
              folder: currentFolder.folder,
              keyword: searchWord,
              page: currentPage + 1,
              filter: filterKeys,
              isRefresh: false,
            })
          }
        }
      }, 500),
    [totalPages, mailList.length, searchWord]
  )

  const onScroll = useCallback(
    (e) => {
      throttledScrollHandler(e)
    },
    [throttledScrollHandler]
  )

  // 监听滚动事件
  useEffect(() => {
    const scrollContainer = tableRef?.current?.querySelector('.arco-table-body')

    if (scrollContainer) {
      scrollContainer.addEventListener('scroll', onScroll)
    }

    return () => {
      if (scrollContainer) {
        scrollContainer.removeEventListener('scroll', onScroll)
      }
      throttledScrollHandler.cancel()
    }
  }, [onScroll, throttledScrollHandler])

  // 销毁定时器
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  // 选中筛选
  const onSelectFilter = (key) => {
    const item = flatTree(filterList).find((e) => e.value === key)
    let filter = [...filterKeys]
    filter[item.key] = item.value

    setFilterKeys(filter)
    setCurrentMail(null)

    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      page: 1,
      filter: filter,
      isRefresh: false,
    })
  }

  // 筛选菜单
  const filterMenu = (
    <Menu onClickMenuItem={onSelectFilter}>
      {filterList.map((group, groupIdx) => (
        <Menu.ItemGroup key={groupIdx} title={group.label}>
          {group.children?.map((menuItem, itemIdx) => {
            const selectedChild = menuItem.children?.find((child) => filterKeys.includes(child.value))
            const currentSelectLabel = selectedChild?.label ?? ''

            return menuItem.children?.length ? (
              <Menu.SubMenu
                key={itemIdx}
                title={
                  <div className='flex flex-1 items-center justify-between'>
                    <span>{menuItem.label}</span>
                    <span className='text-gray-400'>{currentSelectLabel}</span>
                  </div>
                }>
                {menuItem.children.map((subItem) => (
                  <Menu.Item key={subItem.value} className='flex items-center justify-between'>
                    {subItem.label}
                    {filterKeys.includes(subItem.value) && <IconCheck />}
                  </Menu.Item>
                ))}
              </Menu.SubMenu>
            ) : (
              <Menu.Item key={menuItem.value} className='flex items-center justify-between'>
                {menuItem.label}
                {filterKeys.includes(menuItem.value) && <IconCheck />}
              </Menu.Item>
            )
          })}
        </Menu.ItemGroup>
      ))}
    </Menu>
  )

  // 清空筛选
  const onClearFilter = (e) => {
    e.stopPropagation()
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      page: 1,
      filter: ['all', 'date_desc'],
      isRefresh: false,
    })
  }

  return (
    <Layout className='flex-1'>
      {/* 左列：文件夹导航 */}
      <Layout.Sider width={220} theme='light' className='mail-menu box-shadow-none bg-transparent!'>
        <div className='p-4'>
          <Button type='primary rounded!' icon={<IconEdit />} long onClick={() => onWriteMail('new')}>
            写信
          </Button>
        </div>
        <Menu
          className={'bg-transparent! px-2'}
          selectedKeys={[currentFolder?.key || '']}
          onClickMenuItem={(key) => loadMailList(key)}>
          {folderList.map((item) => (
            <Menu.Item key={item.key} className='leading-8!'>
              <div className='flex items-center'>
                {item.icon}
                <span className='inline-block w-27 overflow-hidden align-middle text-ellipsis whitespace-nowrap'>
                  {item.title}
                </span>
              </div>
              {item?.key === 'inbox' && item?.total > 0 && <span className='font-medium text-blue-600'>{item.total}</span>}
              {item?.key === 'compose' && (
                <Button
                  className='m-0!'
                  type='text'
                  size='mini'
                  onClick={() => onClickCompose('inbox')}
                  icon={<IconClose className='m-0! text-gray-500!' />}></Button>
              )}
            </Menu.Item>
          ))}
        </Menu>
      </Layout.Sider>

      {/* 写信 */}
      {currentFolder?.key === 'compose' && (
        <Spin className={'mr-4 w-full rounded-t-xl bg-white'} block loading={currentLoading}>
          <WriteMail
            key={writeMail?.uid || '0'}
            detail={writeMail}
            userList={userList?.list || []}
            onChange={setNewWriteMail} // 监控邮件内容变化
            onClose={onClickCompose} // 关闭写邮件页
            onSend={onSend} // 发邮件或存草稿
            recentlyList={recentlyList} // 最近联系人
            contactList={contactList} //我的联系人
            onEditContact={onEditContact} // 添加编辑联系人
            onDeleteContact={onDeleteContact} // 删除联系人
            onClearContact={onClearContact} // 清空联系人
          />
        </Spin>
      )}

      {currentFolder?.key !== 'compose' && (
        <Layout className='relative mr-4! rounded-t-xl'>
          {/* 切换模式按钮 */}
          {!(isTable && currentMail) && (
            <div className={`absolute top-4 right-4 z-20`}>
              <Button size='small' onClick={() => onChangeMode()} icon={isTable ? <IconLayout /> : <IconMenu />}></Button>
            </div>
          )}
          {/* 搜索框 */}
          <div className='fixed top-0 z-10 w-98 py-3'>
            <Input.Search
              prefix={<IconSearch />}
              placeholder='搜索主题/发件人'
              searchButton
              allowClear
              value={searchWord}
              onChange={setSearchWord}
              onSearch={onSearch}
              onClear={onSearch}
            />
          </div>
          {/* 中列：邮件列表 */}
          <Layout.Sider
            ref={tableRef}
            width={isTable ? (currentMail ? 0 : '100%') : 360}
            className={`box-shadow-none z-10 flex-1`}>
            <Table
              size='middle'
              loading={loading}
              scroll={{ y: 'calc(100vh - 120px)' }}
              className={`email-list h-full ${isTable ? 'email-table' : ''}`}
              rowKey='uid'
              pagination={false}
              border={{
                wrapper: false,
              }}
              rowClassName={(record) => {
                const selectClass = currentMail?.uid === record?.uid ? ' select-row' : ''
                return 'cursor-pointer' + selectClass
              }}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys: selectedRowKeys,
                onChange: (selectedRowKeys) => setSelectedRowKeys(selectedRowKeys),
              }}
              onRow={(record) => {
                return {
                  onClick: (e) => {
                    onMailDetail(record, e)
                  },
                }
              }}
              columns={[
                {
                  title: (
                    <div className='flex items-center justify-between'>
                      <div>
                        <span className={'mr-2 text-base font-bold'}>{currentFolder?.title}</span>
                        <Dropdown
                          trigger='click'
                          triggerProps={{
                            popupStyle: { maxHeight: '400px', width: '200px' },
                          }}
                          droplist={filterMenu}>
                          <Button className='flex items-center' size='small' type={filterNames.length > 0 ? 'secondary' : 'text'}>
                            <IconSort className={`text-base! ${filterNames.length > 0 ? '' : 'text-neutral-600!'}`} />
                            {filterNames.length > 0 && (
                              <>
                                <span>{filterNames.join('; ')}</span>
                                <IconClose onClick={(e) => onClearFilter(e)} />
                              </>
                            )}
                          </Button>
                        </Dropdown>
                      </div>
                      <Space>
                        {currentFolder.folder !== 'Star' && selectedRowKeys.length > 0 && (
                          <Button size='mini' icon={<IconDelete />} onClick={() => onDelMail(selectedRowKeys)}>
                            {currentFolder.folder === 'Deleted' ? '清空' : '删除'}
                          </Button>
                        )}
                        <span className={`${isTable ? 'mr-9' : ''}`}>共 {total} 封</span>
                      </Space>
                    </div>
                  ),
                  dataIndex: 'date',
                  render: (_, record) => (
                    <div className={`w-full ${!record?.flags?.includes('Seen') && record.folder === 'INBOX' ? 'font-bold' : ''}`}>
                      {isTable ? (
                        <div className='flex w-full gap-2 overflow-hidden'>
                          <div className='flex w-60 items-center justify-between gap-1.5'>
                            <div className='flex flex-1 gap-1.5 overflow-hidden'>
                              {showMailIcon(record?.flags)}
                              {currentFolder?.folder === 'Sent' ? (
                                <>
                                  <IconSent />
                                  <div className='flex-1 truncate'>
                                    {record?.to_info?.map((t) => t.name).join(', ') || record?.to}
                                    {record?.cc_info?.length > 0 ? ',  ' : ''}
                                    {record?.cc_info?.map((t) => t.name).join(', ') || record?.cc}
                                  </div>
                                </>
                              ) : (
                                record?.from_info.name || record?.from
                              )}
                            </div>
                            {record.has_attach ? <IconAttachment className='text-base text-gray-400!' /> : ''}
                          </div>
                          <div className='flex w-[calc(100%-500px)] gap-2'>
                            <div className={'max-w-1/2 truncate'}>{record?.subject || ''}</div>
                            <div className={'flex-1 truncate font-light text-gray-400'}>{record?.text || ''}</div>
                          </div>
                          <div className='flex w-50 justify-end text-right'>
                            {record.size}
                            <div className='mr-2 w-20'>
                              {dayjs(record?.send_time).isBefore(dayjs().subtract(1, 'week'))
                                ? dayjs(record?.send_time).format('MM/DD')
                                : dayjs(record?.send_time).fromNow()}
                            </div>
                            {record?.flags?.includes('Flagged') ? (
                              <IconStarSelect className='text-xl!' />
                            ) : (
                              <IconStarUnselect className='text-xl!' />
                            )}
                          </div>
                        </div>
                      ) : (
                        <>
                          <div className='mb-1 flex justify-between gap-2'>
                            <div className='flex w-[calc(100%-72px)] items-center gap-1.5'>
                              {showMailIcon(record?.flags)}
                              {currentFolder?.folder === 'Sent' ? (
                                <>
                                  <IconSent />
                                  <div className={`${record?.to_info.length > 1 ? 'flex-1' : ''} truncate`}>
                                    {record?.to_info?.map((t) => t.name).join(', ') || record?.to}
                                    {record?.cc_info?.length > 0 ? ',  ' : ''}
                                    {record?.cc_info?.map((t) => t.name).join(', ') || record?.cc}
                                  </div>
                                </>
                              ) : (
                                record?.from_info.name || record?.from
                              )}
                              {record.has_attach ? <IconAttachment className='text-base text-gray-400!' /> : ''}
                            </div>
                            <div className='w-18 text-right'>
                              {dayjs(record?.send_time).isBefore(dayjs().subtract(1, 'week'))
                                ? dayjs(record?.send_time).format('MM/DD')
                                : dayjs(record?.send_time).fromNow()}
                            </div>
                          </div>
                          <div className='truncate'>{record?.subject || ''}</div>
                          <div className='flex h-5.5 items-center justify-between'>
                            <div className={'flex-1 truncate font-light text-gray-400'}>{record?.text || ''}</div>
                            {record?.flags?.includes('Flagged') && <IconStarSelect className='cursor-pointer text-xl!' />}
                          </div>
                        </>
                      )}
                    </div>
                  ),
                },
              ]}
              data={mailList}
            />
          </Layout.Sider>

          {/* 右列：邮件详情 + 顶部操作按钮栏 */}
          <Layout.Content className={`relative h-full flex-1 bg-white ${isTable && currentMail ? ' z-10 w-full' : ''}`}>
            {currentMail && (
              <Spin block loading={currentLoading}>
                {/* 邮件操作工具栏 */}
                <div className='flex items-center justify-between gap-2 border-b border-gray-200 p-4'>
                  <div className='flex items-center gap-2'>
                    {isTable && currentMail && (
                      <Button size='small' icon={<IconArrowLeft />} onClick={() => setCurrentMail()}>
                        返回
                      </Button>
                    )}
                    <Button size='small' icon={<IconDelete />} onClick={() => onDelMail([currentMail.uid])}>
                      {currentFolder.key === 'delete' ? '彻底删除' : '删除'}
                    </Button>
                    <Button size='small' icon={<IconReply />} onClick={onReply}>
                      回复
                    </Button>
                    <Button size='small' icon={<IconRedo />} onClick={onForward}>
                      转发
                    </Button>
                    <Dropdown
                      triggerProps={{ autoAlignPopupWidth: true }}
                      trigger='click'
                      droplist={<Menu onClickMenuItem={onFlagMail}>{FlagList(currentMail?.flags)}</Menu>}>
                      <Button size='small'>
                        <div className='flex items-center gap-2'>
                          <IconStar />
                          标记为
                          <IconDown />
                        </div>
                      </Button>
                    </Dropdown>
                    <Dropdown
                      triggerProps={{ autoAlignPopupWidth: true }}
                      trigger='click'
                      droplist={
                        <Menu onClickMenuItem={onMoveMail}>
                          {moveList
                            .filter((e) => ![currentFolder.folder].includes(e.folder))
                            .map((e) => (
                              <Menu.Item key={e.folder}>{e.title}</Menu.Item>
                            ))}
                        </Menu>
                      }>
                      <Button size='small'>
                        <div className='flex items-center gap-2'>
                          <IconMoveFolder />
                          移动到
                          <IconDown />
                        </div>
                      </Button>
                    </Dropdown>
                  </div>

                  {isTable && (
                    <Button.Group type='text'>
                      <Button
                        size='small'
                        icon={<IconLeft />}
                        disabled={currentMail?.uid === mailList[0]?.uid}
                        onClick={() => onCutMail(currentMail, 'prev')}>
                        上一封
                      </Button>
                      <Button
                        size='small'
                        disabled={currentMail?.uid === mailList[mailList?.length - 1]?.uid}
                        onClick={() => onCutMail(currentMail, 'next')}>
                        下一封
                        <IconRight />
                      </Button>
                    </Button.Group>
                  )}
                </div>
                <div className='h-[calc(100vh-117px)] flex-1 overflow-y-auto p-4'>
                  {/* 邮件头部信息 */}
                  <div className='mb-4 flex items-center gap-2'>
                    <span className='text-lg font-bold'>{currentMail.subject}</span>
                    <Button size='mini' type='text' onClick={() => onStar(currentMail)}>
                      {currentMail?.flags?.includes('Flagged') ? (
                        <IconStarSelect className='text-xl!' />
                      ) : (
                        <IconStarUnselect className='text-xl!' />
                      )}
                    </Button>
                  </div>
                  <div className='mb-4 flex items-start gap-3'>
                    {/* 头像 */}
                    <AvatarImage email={currentMail?.from_info?.email} name={currentMail?.from_info?.name} />
                    <div className='flex-1 text-sm'>
                      <Popover
                        position='bl'
                        trigger='hover'
                        key={currentMail.from}
                        triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                        content={
                          <div>
                            <div className='flex gap-2'>
                              <AvatarImage email={currentMail?.from_info?.email} name={currentMail?.from_info?.name} />
                              <div>
                                <div className='flex items-center gap-2 font-bold'>{currentMail?.from_info?.name}</div>
                                <Typography.Text copyable>{currentMail.from}</Typography.Text>
                              </div>
                            </div>
                            {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(currentMail.from) && (
                              <div className={'mt-2'}>
                                <Button
                                  type='primary'
                                  size='small'
                                  long
                                  icon={<IconPlus />}
                                  onClick={() =>
                                    onEditContact({
                                      name: currentMail?.from_info?.name,
                                      email: currentMail.from,
                                      prefix: 'user_contact',
                                    })
                                  }>
                                  添加联系人
                                </Button>
                              </div>
                            )}
                          </div>
                        }>
                        <div className='mb-1'>
                          <strong>{currentMail?.from_info?.name}</strong>
                          <span className='text-gray-400'>&nbsp;&lt;{currentMail.from}&gt;</span>
                        </div>
                      </Popover>
                      <div className='flex items-start justify-between gap-2'>
                        <div className='flex-1'>
                          <div className='mb-1 flex'>
                            <div className='whitespace-nowrap text-gray-400'>收件人</div>
                            <div className='flex flex-wrap'>
                              {currentMail?.to_info?.map((e, index) => (
                                <Popover
                                  position='bl'
                                  trigger='hover'
                                  key={e.email}
                                  triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                                  content={
                                    <div>
                                      <div className='flex gap-2'>
                                        <AvatarImage email={e?.email} name={e?.name} />
                                        <div>
                                          <div className='flex items-center gap-2 font-bold'>{e?.name}</div>
                                          <Typography.Text copyable>{e?.email}</Typography.Text>
                                        </div>
                                      </div>
                                      {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(e?.email) && (
                                        <div className={'mt-2'}>
                                          <Button
                                            type='primary'
                                            size='small'
                                            long
                                            icon={<IconPlus />}
                                            onClick={() =>
                                              onEditContact({
                                                name: e?.name,
                                                email: e?.email,
                                                prefix: 'user_contact',
                                              })
                                            }>
                                            添加联系人
                                          </Button>
                                        </div>
                                      )}
                                    </div>
                                  }>
                                  <span className='mr-1 ml-3'>{e.name}</span>
                                  <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                                  {index !== currentMail?.to_info?.length - 1 && <span className='text-gray-400'>,</span>}
                                </Popover>
                              ))}
                            </div>
                          </div>
                          {currentMail?.cc && (
                            <div className='flex items-center'>
                              <div className='text-gray-400'>抄送</div>
                              <div className='flex flex-wrap'>
                                {currentMail?.cc_info?.map((e, index) => (
                                  <Popover
                                    position='bl'
                                    trigger='hover'
                                    key={e.email}
                                    triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                                    content={
                                      <div>
                                        <div className='flex gap-2'>
                                          <AvatarImage email={e?.email} name={e?.name} />
                                          <div>
                                            <div className='flex items-center gap-2 font-bold'>{e?.name}</div>
                                            <Typography.Text copyable>{e?.email}</Typography.Text>
                                          </div>
                                        </div>
                                        {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(e?.email) && (
                                          <div className={'mt-2'}>
                                            <Button
                                              type='primary'
                                              size='small'
                                              long
                                              icon={<IconPlus />}
                                              onClick={() =>
                                                onEditContact({
                                                  name: e?.name,
                                                  email: e?.email,
                                                  prefix: 'user_contact',
                                                })
                                              }>
                                              添加联系人
                                            </Button>
                                          </div>
                                        )}
                                      </div>
                                    }>
                                    <span className='mr-1 ml-3'>{e.name}</span>
                                    <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                                    {index !== currentMail?.cc_info?.length - 1 && <span className='text-gray-400'>,</span>}
                                  </Popover>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                        <div className='w-45 text-right text-gray-400'>
                          {dayjs(currentMail?.send_time).format('YYYY年MM月DD日 HH:mm:ss') || ''}
                        </div>
                      </div>
                    </div>
                  </div>
                  <Divider />
                  {/* 邮件正文 */}
                  {!['0001-01-01T00:00:00Z', ''].includes(currentMail.schedule) && (
                    <div className='mb-5 flex items-center rounded bg-[#e6edf5] px-4 py-2'>
                      <IconClockCircle className='mr-1 text-blue-500!' />
                      此邮件是定时邮件，将在
                      <span className='mx-2 text-blue-500'>{dayjs(currentMail.schedule).format('YYYY年MM月DD日 HH:mm:ss')}</span>
                      发出。
                      <Button type='text' size='mini' onClick={() => onUnSchedule(currentMail)}>
                        取消发送
                      </Button>
                    </div>
                  )}
                  <div
                    className='mail-detail'
                    dangerouslySetInnerHTML={{
                      __html: currentMail.detail?.content || '<div class="text-gray-500">暂无邮件内容</div>',
                    }}
                  />

                  {/* 附件 */}
                  {currentMail?.has_attach && (
                    <Card
                      className='mt-10'
                      title={
                        <>
                          <IconAttachment className='mr-1' />
                          {currentMail?.detail?.attachments?.length}个 附件 {currentMail?.detail?.attach_size}
                        </>
                      }>
                      <div className='flex flex-col gap-2'>
                        {currentMail?.detail?.attachments?.map((item, index) => (
                          <div key={index} className='flex items-center justify-between gap-2 bg-gray-100 p-2 hover:bg-gray-200'>
                            <div className='flex flex-1 items-center'>
                              <span className='mr-2'>
                                {item?.file_type === 'video' && <IconVideo />}
                                {item?.file_type === 'audio' && <IconAudio />}
                                {item?.file_type === 'zip' && <IconZip />}
                                {item?.file_type === 'image' && <IconImage />}

                                {item?.file_type === 'ppt' && <IconPpt />}
                                {item?.file_type === 'pdf' && <IconPdf />}
                                {item?.file_type === 'excel' && <IconExcel />}
                                {item?.file_type === 'word' && <IconWord />}

                                {item?.file_type === 'text' && <IconText />}
                              </span>
                              {item.file_name}
                              <span className='text-gray-400'>（{item.size}）</span>
                            </div>
                            <Space>
                              <Button type='text' size='small' onClick={() => onPreviewAttachment(item)}>
                                <IconEye />
                                预览
                              </Button>
                              <Button type='text' size='small' onClick={() => onDownloadAttachment(item)}>
                                <IconToBottom />
                                下载
                              </Button>
                            </Space>
                          </div>
                        ))}
                      </div>
                    </Card>
                  )}
                </div>
              </Spin>
            )}
            {!currentMail && (
              <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
            )}
          </Layout.Content>
        </Layout>
      )}
    </Layout>
  )
}

export default MailLayout

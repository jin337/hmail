import { useEffect, useMemo, useRef, useState } from 'react'

import { Message, Modal } from '@arco-design/web-react'
import { IconFile } from '@arco-design/web-react/icon'

import { useAutoRefresh } from 'src/hooks/useAutoRefresh'
import { defaultFilterKeys, PAGE_SIZE, baseFlags, filterList, menuList, moveList } from 'src/constants/mail'
import { getFileType, throttle } from 'src/utils/index'
import { buildQuotedContent, createMailFormData, mergeMailPage, normalizeDetailContent } from 'src/utils/mail'
import * as mailApi from 'src/api/mail'
import * as userApi from 'src/api/user'
import * as contactApi from 'src/api/contact'

/**
 * 邮件页的全部状态与业务逻辑。
 * 视图层（index.jsx / Menu / List / Content / Detail）只消费这里暴露的数据与方法。
 */
export function useMailbox({ currentAccountId, baseUrl, userInfo, searchWord, setSearchWord, searchSeq = 0 }) {
  const tableRef = useRef()
  const refreshParamsRef = useRef(null)
  const lastSearchSeqRef = useRef(searchSeq)

  const [folderList, setFolderList] = useState(menuList)
  const [currentFolder, setCurrentFolder] = useState({})
  const [mailList, setMailList] = useState({})
  const [listLoading, setListLoading] = useState(false)
  const [filterKeys, setFilterKeys] = useState(defaultFilterKeys)
  const [selectedRowKeys, setSelectedRowKeys] = useState([])

  const [isTable, setIsTable] = useState(() => localStorage.getItem('isTable') === 'true')
  const [isMove, setIsMove] = useState(false)

  const [currentMail, setCurrentMail] = useState(null)
  const [mailLoading, setMailLoading] = useState(false)
  const [newMailInfo, setNewMailInfo] = useState(null)

  const [userList, setUserList] = useState([])
  const [recentlyList, setRecentlyList] = useState([])
  const [contactList, setContactList] = useState([])
  const [flagList, setFlagList] = useState(baseFlags)

  // 自动刷新：拉取当前列表参数的最新第一页
  const autoRefresh = useAutoRefresh(async () => {
    const params = refreshParamsRef.current
    if (!params) return
    await getMailList(params, true)
  })

  // 获取未读邮件数量
  async function getUnReadTotal() {
    const { code, data, msg } = await mailApi.fetchUnreadTotal()
    if (code === 200) {
      setFolderList((prev) => prev.map((item) => ({ ...item, total: data[item.folder] || 0 })))
    } else {
      Message.error(msg)
    }
  }

  // 获取邮件列表
  async function getMailList(item, isRefresh) {
    if (!isRefresh) {
      setSelectedRowKeys([])
      setCurrentMail(null)
      setListLoading(true)
    }
    // 记录自动刷新使用的最新参数
    refreshParamsRef.current = item

    const isStar = item?.folder === 'Star'
    const params = isStar ? { filter: item.filter, keyword: item.keyword } : { ...item }

    const { code, data, msg } = isStar ? await mailApi.fetchStarMailList(params) : await mailApi.fetchMailList(params)

    if (code === 200) {
      setMailList((prev) => mergeMailPage(prev, data, item.page))

      autoRefresh.schedule()
      getUnReadTotal()
    } else {
      Message.error(msg)
    }
    setListLoading(false)
  }

  // 取消定时发送
  const onUnSchedule = async (item, type = 2) => {
    const { code } = await mailApi.unScheduleMail({
      uid: item.uid,
      folder: item.folder,
      status: 'Draft',
      type,
    })
    if (code === 200) {
      setCurrentMail(null)
      getMailList({
        folder: item.folder,
        filter: filterKeys,
        keyword: searchWord,
        page: 1,
        size: PAGE_SIZE,
      })
    }
  }

  // 发送邮件 / 存草稿
  const onSend = async (type, detailProps, customTime) => {
    const { detail, ...rest } = detailProps

    if (!rest.to_info || !rest.subject) {
      Message.warning('请填写收件人和主题')
      return
    }

    const formData = createMailFormData({ rest, detail, type, customTime })

    if (type !== 'Drafts' && rest?.is_reply) {
      // 回复成功后标记原邮件为已回复
      await mailApi.updateMailStatus({
        uid: rest?.uid,
        folder: rest?.folder,
        status: 'Answered',
        type: 1,
      })
    }

    const { code, msg } =
      type === 'Drafts' ? await mailApi.saveDraft(formData) : await mailApi.sendMail(formData)

    if (code === 200) {
      if (customTime) {
        // 定时发送：邮件停留在草稿箱等待调度
        Message.success('定时邮件已保存，将在指定时间发出')
        onCloseEdit('drafts')
      } else {
        Message.success(msg)
        onCloseEdit(type === 'Sent' ? 'sent' : 'drafts')
        getContactList({ prefix: 'user_sent' })
      }
    } else {
      Message.error(msg)
    }
  }

  // 回复 / 转发
  const onReplyForward = (key) => {
    if (!currentMail) return

    const FormContent = buildQuotedContent(currentMail)
    let newMail = null

    if (key === 'is_reply') {
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

    if (key === 'is_forward') {
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
          const { code } = await mailApi.deleteMail({ folder: 'Deleted', uids: ids })
          if (code === 200) {
            Message.success('邮件已彻底删除')
          }
        } else {
          if (folder === 'Drafts') {
            // 定时邮件需先取消定时
            const isSchedule = mailList?.list?.filter(
              (e) => ids.includes(e.uid) && Array.isArray(e.flags) && e.flags.includes('Draft')
            )
            if (isSchedule?.length > 0) {
              return Message.error('请先取消定时邮件后再进行删除操作')
            }
          }

          const { code } = await mailApi.moveMail({
            uids: ids,
            from_folder: folder,
            to_folder: 'Deleted',
          })
          if (code === 200) {
            Message.success('已移入垃圾箱')
          }
        }

        getMailList({
          folder: currentFolder.folder,
          keyword: searchWord,
          filter: filterKeys,
          page: 1,
          size: PAGE_SIZE,
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

    const { code } = await mailApi.moveMail({ from_folder: from, to_folder: to, uids: list })
    if (code !== 200) {
      Message.error('移动失败')
      return
    }
    Message.success('移动成功')
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      filter: filterKeys,
      page: 1,
      size: PAGE_SIZE,
    })
  }

  // 标记为
  const onFlagMail = ({ uids, from, to }) => {
    const key = to.split('_')
    const item = { uids, folder: from, type: Number(key[1]) }

    if (key[0] === 'Seen') {
      onRead(item)
    }

    if (key[0] === 'Flagged') {
      onStar(item)
    }
  }

  // 根据选中邮件动态生成「标记为」列表
  const onChangeMailFlag = (selected, list = mailList?.list) => {
    const selectedMails = (list || []).filter((item) => selected.includes(item.uid))

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
    } else if (someHasSeen) {
      readList = [
        { flag: 'Seen', key: 1, title: '已读邮件' },
        { flag: 'Seen', key: 2, title: '未读邮件' },
      ]
    } else {
      readList = [{ flag: 'Seen', key: 1, title: '已读邮件' }]
    }

    let starList = []
    if (allHasFlagged) {
      starList = [{ flag: 'Flagged', key: 2, title: '取消星标' }]
    } else if (someHasFlagged) {
      starList = [
        { flag: 'Flagged', key: 1, title: '星标邮件' },
        { flag: 'Flagged', key: 2, title: '取消星标' },
      ]
    } else {
      starList = [{ flag: 'Flagged', key: 1, title: '星标邮件' }]
    }

    setFlagList([...readList, { title: '分割线', flag: 'divider' }, ...starList])
  }

  // 标记已读 / 未读
  const onRead = async (item) => {
    if (item.folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }

    const uids = item.uids.filter(Boolean)
    if (uids.length === 0) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }

    const params = { uids, folder: item.folder, status: 'Seen', type: item.type }
    const { code } = await mailApi.updateMailStatus(params)
    if (code === 200) {
      const uidSet = new Set(uids)
      const newList = (mailList.list || []).map((mailItem) => {
        if (!uidSet.has(mailItem.uid)) return mailItem

        let flags = mailItem.flags || []
        if (item.type === 1) {
          if (!flags.includes('Seen')) flags = [...flags, 'Seen']
        }
        if (item.type === 2) {
          flags = flags.filter((f) => f !== 'Seen')
        }
        return { ...mailItem, flags }
      })

      setMailList((prev) => ({ ...prev, list: newList }))
      getUnReadTotal()
      onChangeMailFlag(uids, newList)
    }
  }

  // 标记 / 取消星标
  const onStar = async (item) => {
    if (item.folder === 'Star') {
      return Message.warning('请选择非星标邮件')
    }

    const uids = item.uids.filter(Boolean)
    if (uids.length === 0) {
      return Message.warning({
        content: '未选中任何邮件',
        showIcon: true,
        position: 'bottom',
      })
    }

    const params = { uids, folder: item.folder, status: 'Flagged', type: item.type }
    const { code } = await mailApi.updateMailStatus(params)
    if (code === 200) {
      const uidSet = new Set(uids)

      let newList = (mailList.list || []).map((mailItem) => {
        if (!uidSet.has(mailItem.uid)) return mailItem

        let flags = mailItem.flags || []
        if (item.type === 1) {
          if (!flags.includes('Flagged')) flags = [...flags, 'Flagged']
        } else {
          flags = flags.filter((flag) => flag !== 'Flagged')
        }
        return { ...mailItem, flags }
      })

      // 星标文件夹下取消星标：从列表移除
      if (currentFolder.folder === 'Star') {
        newList = newList.filter((mailItem) => !uidSet.has(mailItem.uid))
      }

      setMailList((prev) => ({ ...prev, list: newList }))

      if (currentMail) {
        setCurrentMail((prev) => {
          if (!prev) return prev

          let flags = prev?.flags || []
          if (item.type === 1) {
            if (!flags.includes('Flagged')) flags = [...flags, 'Flagged']
          } else {
            flags = flags.filter((flag) => flag !== 'Flagged')
          }

          if (currentFolder.folder === 'Star') return null
          return { ...prev, flags }
        })
      }

      getUnReadTotal()
      onChangeMailFlag(uids, newList)
    }
  }

  // 写信内容变化
  const onChangeNewMail = (item) => {
    if (currentFolder.key === 'compose') {
      setNewMailInfo(item)
    }
  }

  // 清空联系人
  const onClearContact = async (params) => {
    const { code, msg } = await contactApi.clearContact(params)
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 删除联系人
  const onDeleteContact = async (params) => {
    const { code, msg } = await contactApi.deleteContact({ email: params?.email, prefix: params?.prefix })
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 新增 / 编辑联系人
  const onEditContact = async (params) => {
    const { code, msg } = await contactApi.saveContact(params)
    if (code === 200) {
      Message.success(msg)
      getContactList(params)
    }
  }

  // 获取用户列表
  const getUserList = async () => {
    const { code, data, msg } = await userApi.fetchUserList()
    if (code === 200) {
      setUserList({
        list: (data.list || []).map((e) => ({ ...e, is_me: e.email === currentAccountId })),
        total: data.total,
      })
    } else {
      Message.error(msg)
    }
  }

  // 获取联系人列表
  const getContactList = async (params) => {
    const { code, data, msg } = await contactApi.fetchContactList({ prefix: params?.prefix })
    if (code === 200) {
      const list = (data.list || []).map((e) => ({ ...e, full_name: e.name }))
      if (params?.prefix === 'user_sent') {
        setRecentlyList(list)
      }
      if (params?.prefix === 'user_contact') {
        setContactList(list)
      }
    } else {
      Message.error(msg)
    }
  }

  // 获取邮件详情
  const getCurrentMail = async (item) => {
    setMailLoading(true)
    const { code, data, msg } = await mailApi.fetchMailDetail({ uid: item.uid, folder: item.folder })

    if (code === 200) {
      const newData = {
        ...data,
        attachments: data?.attachments?.map((e) => ({
          ...e,
          file_type: getFileType(e.file_type),
        })),
        content: normalizeDetailContent(data?.content),
      }

      setCurrentMail({ ...item, detail: newData })

      if (!item?.flags || !item.flags?.includes('Seen')) {
        onRead({ uids: [item.uid], folder: item.folder, type: 1 })
      } else {
        onChangeMailFlag([item.uid], mailList.list)
      }

      // 草稿：打开编辑
      if (item.folder === 'Drafts' && item.schedule === '0001-01-01T00:00:00Z') {
        onEdit({
          ...item,
          detail: newData,
          to_info: item?.to_info?.map((e) => ({ label: e.name, value: e.email })),
          cc_info: item?.cc_info?.map((e) => ({ label: e.name, value: e.email })) || [],
        })
      }
    } else {
      Message.error(msg)
    }

    setSelectedRowKeys([])
    setMailLoading(false)
  }

  // 切换上一封 / 下一封
  const onCutMail = (record, key) => {
    const list = mailList?.list || []
    const index = list.findIndex((e) => e.uid === record.uid)
    if (key === 'prev') {
      setCurrentMail(list[index - 1])
    } else if (key === 'next') {
      setCurrentMail(list[index + 1])
    }
  }

  // 写信 / 编辑草稿
  const onEdit = (record) => {
    setCurrentMail(null)
    const isComposeExist = folderList.some((item) => item.key === 'compose')
    if (isComposeExist) {
      return Message.warning('写邮件页已打开，请先关闭')
    }

    const compose = { key: 'compose', folder: 'Drafts', title: '草稿箱', icon: <IconFile /> }
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
    setNewMailInfo(null)
  }

  // 筛选
  const onSelectFilter = (filter) => {
    setFilterKeys(filter)
    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      filter,
      page: 1,
      size: PAGE_SIZE,
    })
  }

  // 滚动到顶部
  const scrollToTop = () => {
    const scrollContainer = tableRef?.current?.querySelector('.arco-table-body')
    if (scrollContainer) {
      scrollContainer.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  // 滚动加载下一页
  const totalPages = Math.ceil(mailList?.total / PAGE_SIZE)
  const throttledScrollHandler = useMemo(
    () =>
      throttle((e) => {
        const { scrollTop, scrollHeight, clientHeight } = e.target
        const distanceToBottom = scrollHeight - scrollTop - clientHeight

        if (distanceToBottom <= 200 && !listLoading) {
          const currentPage = Math.ceil((mailList.list?.length || 0) / PAGE_SIZE)
          if (currentPage < totalPages) {
            getMailList({
              folder: currentFolder.folder,
              keyword: searchWord,
              filter: filterKeys,
              page: currentPage + 1,
              size: PAGE_SIZE,
            })
          }
        }
      }, 500),
    [totalPages, mailList.list, searchWord, currentFolder.folder, filterKeys, listLoading]
  )

  const onScroll = (e) => throttledScrollHandler(e)

  const bindRef = useRef(false)
  useEffect(() => {
    const scrollContainer = tableRef.current
    if (!scrollContainer) return

    if (!bindRef.current) {
      scrollContainer.addEventListener('scroll', onScroll)
      bindRef.current = true
    }

    return () => {
      scrollContainer.removeEventListener('scroll', onScroll)
      throttledScrollHandler.cancel()
      bindRef.current = false
    }
  }, [throttledScrollHandler])

  // 搜索邮件
  const onSearch = (val) => {
    setCurrentMail(null)
    setSelectedRowKeys([])
    getMailList({ folder: currentFolder.folder, keyword: val, page: 1, filter: filterKeys })
    scrollToTop()
  }

  // 顶部搜索框触发（searchSeq 由 Home 递增）
  useEffect(() => {
    if (searchSeq === lastSearchSeqRef.current) return
    lastSearchSeqRef.current = searchSeq
    onSearch(searchWord)
  }, [searchSeq])

  // 监听选中邮件，加载详情
  useEffect(() => {
    const init = async () => {
      if (!currentMail?.uid) return
      await getCurrentMail(currentMail)
    }
    init()
  }, [currentMail?.uid])

  // 切换目录 / 草稿内容变化，重新加载
  useEffect(() => {
    if (!currentFolder?.folder) return
    if (currentFolder.key === 'compose') return

    getMailList({
      folder: currentFolder.folder,
      keyword: searchWord,
      filter: filterKeys,
      page: 1,
      size: PAGE_SIZE,
    })

    // 切换文件夹，自动刷新重新计时
    autoRefresh.restart()
  }, [currentFolder, newMailInfo])

  // 首次加载
  useEffect(() => {
    setCurrentFolder(folderList[0])
    getUserList()
    getContactList({ prefix: 'user_sent' })
    getContactList({ prefix: 'user_contact' })
  }, [])

  // 统一上下文
  const mailContext = useMemo(
    () => ({
      baseUrl,
      userInfo,
      setSearchWord,

      folderList,
      setFolderList,
      currentFolder,
      setCurrentFolder,
      onEdit,
      onCloseEdit,

      tableRef,
      mailList,
      listLoading,
      filterList,
      filterKeys,
      onSelectFilter,
      isTable,
      setIsTable,
      isMove,
      setIsMove,

      selectedRowKeys,
      setSelectedRowKeys,

      currentMail,
      setCurrentMail,
      mailLoading,
      onUnSchedule,

      onCutMail,
      onStar,
      onRead,
      onDelMail,
      onMoveMail,
      onReplyForward,

      moveList,

      contactList,
      onEditContact,
      flagList,
      onChangeMailFlag,
      onFlagMail,
    }),
    [
      baseUrl,
      userInfo,
      folderList,
      currentFolder,
      mailList,
      selectedRowKeys,
      currentMail,
      mailLoading,
      listLoading,
      isTable,
      isMove,
      filterKeys,
      flagList,
      contactList,
      setSearchWord,
    ]
  )

  return {
    mailContext,
    compose: {
      detail: newMailInfo,
      userList: userList?.list || [],
      recentlyList,
      contactList,
      onClose: onCloseEdit,
      onSend,
      onEditContact,
      onDeleteContact,
      onClearContact,
      onChange: onChangeNewMail,
    },
  }
}

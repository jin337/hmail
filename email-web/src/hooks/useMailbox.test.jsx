import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useMailbox } from './useMailbox'

vi.mock('@arco-design/web-react', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...actual,
    Message: Object.assign(vi.fn(), {
      success: vi.fn(),
      error: vi.fn(),
      warning: vi.fn(),
      info: vi.fn(),
    }),
  }
})

vi.mock('src/api/mail', () => ({
  fetchMailList: vi.fn(),
  fetchStarMailList: vi.fn(),
  fetchUnreadTotal: vi.fn(),
  fetchMailDetail: vi.fn(),
  updateMailStatus: vi.fn(),
  downloadAttachment: vi.fn(),
  moveMail: vi.fn(),
  deleteMail: vi.fn(),
  saveDraft: vi.fn(),
  sendMail: vi.fn(),
  unScheduleMail: vi.fn(),
}))

vi.mock('src/api/user', () => ({
  fetchUserList: vi.fn(),
  createUser: vi.fn(),
  updateUser: vi.fn(),
  deleteUser: vi.fn(),
  changePassword: vi.fn(),
}))

vi.mock('src/api/contact', () => ({
  fetchContactList: vi.fn(),
  saveContact: vi.fn(),
  deleteContact: vi.fn(),
  clearContact: vi.fn(),
}))

import * as mailApi from 'src/api/mail'
import * as userApi from 'src/api/user'
import * as contactApi from 'src/api/contact'

const inboxMails = [
  { uid: 1, folder: 'INBOX', flags: [], subject: '第一封', send_time: '2026-01-02 10:00:00' },
  { uid: 2, folder: 'INBOX', flags: ['Seen'], subject: '第二封', send_time: '2026-01-01 10:00:00' },
]

const starMails = [
  { uid: 11, folder: 'Star', flags: ['Flagged'], subject: '星标一' },
  { uid: 12, folder: 'Star', flags: ['Flagged'], subject: '星标二' },
]

const renderMailbox = (props = {}) =>
  renderHook((p) => useMailbox(p), {
    initialProps: {
      currentAccountId: 'me@jin337.top',
      baseUrl: 'http://localhost:8058',
      userInfo: { email: 'me@jin337.top', full_name: '我' },
      searchWord: '',
      setSearchWord: vi.fn(),
      searchSeq: 0,
      ...props,
    },
  })

beforeEach(() => {
  mailApi.fetchMailList.mockResolvedValue({ code: 200, data: { total: 2, list: inboxMails } })
  mailApi.fetchStarMailList.mockResolvedValue({ code: 200, data: { total: 2, list: starMails } })
  mailApi.fetchUnreadTotal.mockResolvedValue({
    code: 200,
    data: { INBOX: 2, Star: 2, Sent: 0, Drafts: 1, Deleted: 0 },
  })
  mailApi.fetchMailDetail.mockResolvedValue({
    code: 200,
    data: { content: '<p>正文</p>', attachments: [], cc: false },
  })
  mailApi.updateMailStatus.mockResolvedValue({ code: 200 })
  mailApi.moveMail.mockResolvedValue({ code: 200 })
  mailApi.deleteMail.mockResolvedValue({ code: 200 })
  mailApi.sendMail.mockResolvedValue({ code: 200, msg: '发送成功' })
  mailApi.saveDraft.mockResolvedValue({ code: 200, msg: '草稿保存成功' })
  mailApi.unScheduleMail.mockResolvedValue({ code: 200 })
  userApi.fetchUserList.mockResolvedValue({
    code: 200,
    data: { total: 2, list: [{ email: 'me@jin337.top', full_name: '我' }, { email: 'other@jin337.top', full_name: '他' }] },
  })
  contactApi.fetchContactList.mockResolvedValue({ code: 200, data: { list: [{ name: '联系人', email: 'c@jin337.top' }] } })
  contactApi.saveContact.mockResolvedValue({ code: 200, msg: '保存成功' })
  contactApi.deleteContact.mockResolvedValue({ code: 200, msg: '删除成功' })
  contactApi.clearContact.mockResolvedValue({ code: 200, msg: '清空成功' })
})

describe('useMailbox 初始化', () => {
  it('默认进入收件箱并加载列表、未读数、联系人', async () => {
    const { result, unmount } = renderMailbox()

    await waitFor(() => {
      expect(result.current.mailContext.currentFolder.folder).toBe('INBOX')
      expect(result.current.mailContext.mailList.list).toHaveLength(2)
    })

    expect(mailApi.fetchMailList).toHaveBeenCalledWith(
      expect.objectContaining({ folder: 'INBOX', page: 1, size: 50, keyword: '' })
    )
    expect(result.current.mailContext.folderList.find((f) => f.folder === 'INBOX').total).toBe(2)
    expect(result.current.compose.userList).toHaveLength(2)
    expect(result.current.compose.recentlyList).toHaveLength(1)
    expect(result.current.compose.contactList).toHaveLength(1)

    unmount()
  })
})

describe('useMailbox 搜索', () => {
  it('searchSeq 变化时按关键字重新加载第一页', async () => {
    const { result, rerender, unmount } = renderMailbox()

    await waitFor(() => expect(result.current.mailContext.mailList.list).toHaveLength(2))

    rerender({
      currentAccountId: 'me@jin337.top',
      baseUrl: 'http://localhost:8058',
      userInfo: { email: 'me@jin337.top' },
      searchWord: '账单',
      setSearchWord: vi.fn(),
      searchSeq: 1,
    })

    await waitFor(() => {
      expect(mailApi.fetchMailList).toHaveBeenLastCalledWith(
        expect.objectContaining({ folder: 'INBOX', keyword: '账单', page: 1 })
      )
    })

    unmount()
  })
})

describe('useMailbox 标记', () => {
  it('标记已读后更新列表、未读数与标记菜单', async () => {
    const { result, unmount } = renderMailbox()
    await waitFor(() => expect(result.current.mailContext.mailList.list).toHaveLength(2))

    await act(async () => {
      await result.current.mailContext.onRead({ uids: [1], folder: 'INBOX', type: 1 })
    })

    const item = result.current.mailContext.mailList.list.find((m) => m.uid === 1)
    expect(item.flags).toContain('Seen')
    expect(mailApi.updateMailStatus).toHaveBeenCalledWith({ uids: [1], folder: 'INBOX', status: 'Seen', type: 1 })
    expect(mailApi.fetchUnreadTotal).toHaveBeenCalled()
    expect(result.current.mailContext.flagList.map((f) => f.title)).toContain('未读邮件')

    unmount()
  })

  it('星标文件夹下取消星标时从列表移除', async () => {
    const { result, unmount } = renderMailbox()
    await waitFor(() => expect(result.current.mailContext.mailList.list).toHaveLength(2))

    act(() => {
      result.current.mailContext.setCurrentFolder({ key: 'star', folder: 'Star', title: '星标邮件' })
    })
    await waitFor(() => expect(result.current.mailContext.mailList.list.map((m) => m.uid)).toEqual([11, 12]))
    expect(mailApi.fetchStarMailList).toHaveBeenCalled()

    await act(async () => {
      await result.current.mailContext.onStar({ uids: [11], folder: 'INBOX', type: 2 })
    })

    expect(result.current.mailContext.mailList.list.map((m) => m.uid)).toEqual([12])

    unmount()
  })
})

describe('useMailbox 写信', () => {
  it('写信打开与关闭会切换目录', async () => {
    const { result, unmount } = renderMailbox()
    await waitFor(() => expect(result.current.mailContext.currentFolder.folder).toBe('INBOX'))

    act(() => {
      result.current.mailContext.onEdit({ uid: 5, subject: '草稿主题' })
    })

    expect(result.current.mailContext.currentFolder.key).toBe('compose')
    expect(result.current.mailContext.folderList[0].title).toBe('草稿主题')

    act(() => {
      result.current.mailContext.onCloseEdit('drafts')
    })

    expect(result.current.mailContext.currentFolder.key).toBe('drafts')
    expect(result.current.mailContext.folderList.some((f) => f.key === 'compose')).toBe(false)
    expect(result.current.compose.detail).toBeNull()

    unmount()
  })

  it('发送邮件会提交 FormData 并关闭写信页', async () => {
    const { result, unmount } = renderMailbox()
    await waitFor(() => expect(result.current.mailContext.currentFolder.folder).toBe('INBOX'))

    act(() => {
      result.current.mailContext.onEdit({ uid: 5, subject: '草稿主题', folder: 'Drafts' })
    })

    await act(async () => {
      await result.current.compose.onSend(
        'Sent',
        {
          uid: 5,
          folder: 'Drafts',
          to_info: [{ value: 'a@jin337.top' }],
          subject: '草稿主题',
          detail: { content: '<p>正文</p>', attachments: [] },
        },
        ''
      )
    })

    expect(mailApi.sendMail).toHaveBeenCalledTimes(1)
    const formData = mailApi.sendMail.mock.calls[0][0]
    expect(formData.get('folder')).toBe('Drafts')
    expect(formData.get('uid')).toBe('5')
    expect(result.current.mailContext.currentFolder.key).toBe('sent')
    expect(result.current.compose.detail).toBeNull()

    unmount()
  })

  it('缺少收件人或主题时给出提示且不提交', async () => {
    const { result, unmount } = renderMailbox()

    await act(async () => {
      await result.current.compose.onSend('Sent', { to_info: [], subject: '' }, '')
    })

    expect(mailApi.sendMail).not.toHaveBeenCalled()

    unmount()
  })
})


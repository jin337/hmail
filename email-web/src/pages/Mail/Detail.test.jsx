import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MailProvider } from './MailContext'
import Detail from './Detail'

vi.mock('src/api/mail', () => ({
  downloadAttachment: vi.fn(),
}))

const currentMail = {
  uid: 1,
  folder: 'INBOX',
  subject: '测试邮件主题',
  from: 'sender@jin337.top',
  from_info: { name: '发件人' },
  to_info: [{ name: '王五', email: 'to@jin337.top' }],
  cc: false,
  send_time: '2026-01-01 10:00:00',
  schedule: '',
  flags: ['Seen'],
  has_attach: true,
  detail: {
    content: '<p>邮件正文内容</p>',
    attachments: [{ part_id: '2', file_name: '报告.pdf', size: '1.2MB', file_type: 'pdf' }],
    attach_size: '1.2MB',
  },
}

const buildContext = (overrides = {}) => ({
  baseUrl: 'http://localhost:8058',
  userInfo: { email: 'me@jin337.top', full_name: '我' },
  mailList: { total: 1, list: [currentMail] },
  currentMail,
  mailLoading: false,
  currentFolder: { key: 'inbox', folder: 'INBOX', title: '收件箱' },
  contactList: [],
  onEditContact: vi.fn(),
  onEdit: vi.fn(),
  onStar: vi.fn(),
  onUnSchedule: vi.fn(),
  selectedRowKeys: [],
  ...overrides,
})

describe('Detail', () => {
  it('渲染邮件标题、发件人、正文与附件', () => {
    render(
      <MailProvider value={buildContext()}>
        <Detail />
      </MailProvider>
    )

    expect(screen.getByText('测试邮件主题')).toBeInTheDocument()
    expect(screen.getByText('发件人')).toBeInTheDocument()
    expect(screen.getByText('收件人')).toBeInTheDocument()
    expect(screen.getByText('邮件正文内容')).toBeInTheDocument()
    expect(screen.getByText('报告.pdf')).toBeInTheDocument()
    expect(screen.getByText('预览')).toBeInTheDocument()
    expect(screen.getByText('下载')).toBeInTheDocument()
  })

  it('定时邮件显示提示条', () => {
    const scheduled = { ...currentMail, schedule: '2026-02-01 09:00:00' }
    render(
      <MailProvider value={buildContext({ currentMail: scheduled, mailList: { total: 1, list: [scheduled] } })}>
        <Detail />
      </MailProvider>
    )

    expect(screen.getByText('取消发送')).toBeInTheDocument()
    expect(screen.getByText(/此邮件是定时邮件/)).toBeInTheDocument()
  })

  it('没有选中邮件时显示空状态', () => {
    render(
      <MailProvider value={buildContext({ currentMail: null, mailList: { total: 0, list: [] } })}>
        <Detail />
      </MailProvider>
    )

    expect(screen.getByText('未选中任何邮件')).toBeInTheDocument()
  })
})

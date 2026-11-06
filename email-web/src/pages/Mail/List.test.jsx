import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { MailProvider } from './MailContext'
import ListLayout from './List'
import { defaultFilterKeys, filterList, baseFlags, moveList } from 'src/constants/mail'

vi.mock('src/utils/index', async (importOriginal) => {
  const actual = await importOriginal()
  return { ...actual, formatMailTime: () => '刚刚' }
})

const mails = [
  { uid: 1, folder: 'INBOX', flags: [], subject: '未读邮件', text: '摘要一', from_info: { name: '张三' }, from: 'a@x.com' },
  {
    uid: 2,
    folder: 'INBOX',
    flags: ['Seen', 'Flagged'],
    subject: '已读邮件',
    text: '摘要二',
    from_info: { name: '李四' },
    from: 'b@x.com',
    has_attach: true,
  },
]

const buildContext = (overrides = {}) => ({
  currentFolder: { key: 'inbox', folder: 'INBOX', title: '收件箱' },
  mailList: { total: 2, list: mails },
  listLoading: false,
  filterList,
  filterKeys: defaultFilterKeys,
  onSelectFilter: vi.fn(),
  currentMail: null,
  setCurrentMail: vi.fn(),
  isTable: false,
  tableRef: { current: null },
  setSelectedRowKeys: vi.fn(),
  onDelMail: vi.fn(),
  onStar: vi.fn(),
  onFlagMail: vi.fn(),
  flagList: baseFlags,
  onChangeMailFlag: vi.fn(),
  onMoveMail: vi.fn(),
  moveList,
  onRead: vi.fn(),
  isMove: false,
  ...overrides,
})

describe('List', () => {
  it('渲染邮件主题、发件人与总数', () => {
    render(
      <MailProvider value={buildContext()}>
        <ListLayout />
      </MailProvider>
    )

    expect(screen.getByText('未读邮件')).toBeInTheDocument()
    expect(screen.getByText('已读邮件')).toBeInTheDocument()
    expect(screen.getByText('张三')).toBeInTheDocument()
    expect(screen.getByText('共 2 封')).toBeInTheDocument()
  })

  it('未读邮件加粗，选中邮件使用 selelct-mail 标记', () => {
    const { container } = render(
      <MailProvider value={buildContext({ currentMail: mails[1] })}>
        <ListLayout />
      </MailProvider>
    )

    const items = container.querySelectorAll('.mail-item')
    expect(items).toHaveLength(2)
    expect(items[0].className).toContain('font-bold')
    expect(items[1].className).toContain('selelct-mail')
    expect(items[1].className).toContain('py-2')
  })

  it('空列表显示占位文案', () => {
    render(
      <MailProvider value={buildContext({ mailList: { total: 0, list: [] } })}>
        <ListLayout />
      </MailProvider>
    )

    expect(screen.getByText('暂无邮件待处理')).toBeInTheDocument()
  })
})

import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { MailProvider } from './MailContext'
import MenuLayout from './Menu'

const buildContext = (overrides = {}) => ({
  folderList: [
    { key: 'inbox', folder: 'INBOX', title: '收件箱', icon: <span />, total: 3 },
    { key: 'sent', folder: 'Sent', title: '已发送', icon: <span />, total: 0 },
  ],
  currentFolder: { key: 'inbox', folder: 'INBOX', title: '收件箱' },
  setCurrentFolder: vi.fn(),
  setSearchWord: vi.fn(),
  setSelectedRowKeys: vi.fn(),
  onEdit: vi.fn(),
  onCloseEdit: vi.fn(),
  ...overrides,
})

describe('Menu', () => {
  it('渲染文件夹与未读数量', () => {
    render(
      <MailProvider value={buildContext()}>
        <MenuLayout />
      </MailProvider>
    )

    expect(screen.getByText('收件箱')).toBeInTheDocument()
    expect(screen.getByText('已发送')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('点击写信按钮打开写信页', async () => {
    const onEdit = vi.fn()
    render(
      <MailProvider value={buildContext({ onEdit })}>
        <MenuLayout />
      </MailProvider>
    )

    await userEvent.click(screen.getByText('写信'))
    expect(onEdit).toHaveBeenCalledTimes(1)
  })

  it('切换目录时清空搜索词与选中项', async () => {
    const setCurrentFolder = vi.fn()
    const setSearchWord = vi.fn()
    const setSelectedRowKeys = vi.fn()

    render(
      <MailProvider value={buildContext({ setCurrentFolder, setSearchWord, setSelectedRowKeys })}>
        <MenuLayout />
      </MailProvider>
    )

    await userEvent.click(screen.getByText('已发送'))

    expect(setCurrentFolder).toHaveBeenCalledWith(expect.objectContaining({ key: 'sent' }))
    expect(setSearchWord).toHaveBeenCalledWith('')
    expect(setSelectedRowKeys).toHaveBeenCalledWith([])
  })
})

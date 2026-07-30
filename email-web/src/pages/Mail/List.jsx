import { useEffect, useMemo, useState } from 'react'

import { Button, Checkbox, Dropdown, Empty, Menu, Space, Spin } from '@arco-design/web-react'
import { IconAttachment, IconCheck, IconClose, IconDelete, IconSort } from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

import { useMailContext } from './MailContext'

import { flatTree, formatMailTime } from 'src/utils/index'

import IconMailNormal from 'src/assets/mail_normal.svg'
import IconMailOpen from 'src/assets/mail_open.svg'
import IconMailReply from 'src/assets/mail_reply.svg'
import IconSent from 'src/assets/mail_sent.svg'
import IconStarUnselect from 'src/assets/mail_star.svg'
import IconStarSelect from 'src/assets/mail_star_open.svg'
import IconMailTimer from 'src/assets/mail_timer.svg'

// 邮件图标
const showMailIcon = (flags) => {
  const flagArr = Array.isArray(flags) ? flags : []
  if (flagArr.includes('Draft')) return <IconMailTimer />
  if (flagArr.includes('Answered')) return <IconMailReply />
  if (flagArr.includes('Seen')) return <IconMailOpen />
  return <IconMailNormal />
}

// 邮件分组
const groupMailByTime = (list) => {
  if (list?.length === 0) return []
  const today = dayjs()
  // 定义分组容器，顺序决定最终展示顺序
  const groups = {
    today: [],
    yesterday: [],
    thisWeek: [],
    lastWeek: [],
    older: [],
  }

  list.forEach((item) => {
    const time = dayjs(item.send_time)
    if (time.isSame(today, 'day')) {
      groups.today.push(item)
    } else if (time.isSame(today.subtract(1, 'day'), 'day')) {
      groups.yesterday.push(item)
    } else if (time.isSame(today, 'week')) {
      groups.thisWeek.push(item)
    } else if (time.isSame(today.subtract(1, 'week'), 'week')) {
      groups.lastWeek.push(item)
    } else {
      groups.older.push(item)
    }
  })

  // 映射标题（和你需求对应）
  const groupMap = [
    { key: 'today', title: '今天' },
    { key: 'yesterday', title: '昨日' },
    { key: 'thisWeek', title: '周一' },
    { key: 'lastWeek', title: '上周' },
    { key: 'older', title: '更早' },
  ]

  const list2 = []
  groupMap.forEach(({ key, title }) => {
    const arr = groups[key]
    if (arr.length === 0) return
    list2.push({ key, title, total: arr.length })
    list2.push(...arr)
  })
  return list2
}

const ListLayout = () => {
  const {
    currentFolder,
    mailList,
    listLoading,
    filterList,
    filterKeys,
    onSelectFilter,
    currentMail,
    setCurrentMail,
    isTable,
    tableRef,
    selectedRowKeys,
    setSelectedRowKeys,
    onDelMail,
  } = useMailContext()
  const [mailData, setMailData] = useState([])

  const { selected, selectAll, isSelected, unSelectAll, isAllSelected, isPartialSelected, setValueSelected } =
    Checkbox.useCheckbox(
      (mailList?.list || [])?.map((x) => x.uid),
      []
    )

  // 组选择
  const onSelectGroup = (item) => {
    const inx = mailData.findIndex((x) => x.key === item.key)
    if (inx === -1) return
    const groupMails = mailData.slice(inx + 1, inx + 1 + item.total)
    const uidList = groupMails.map((mail) => mail.uid)

    if (isSelected(uidList[0])) {
      setValueSelected(uidList, false)
      return
    }
    setValueSelected(uidList, true)
  }

  // 筛选后展示的名称
  const filterNames = useMemo(
    () =>
      filterKeys
        .map((key) => {
          const item = flatTree(filterList)
            .filter((item) => !['all', 'date_desc'].includes(item.value))
            .find((e) => e.value === key)
          return item?.label
        })
        .filter(Boolean),
    [filterKeys]
  )

  // 监控邮件选中
  useEffect(() => {
    if (selected?.length > 0) {
      setSelectedRowKeys(selected)
    }
  }, [selected])

  // 监控邮件列表
  useEffect(() => {
    const init = () => {
      unSelectAll()

      const list = groupMailByTime(mailList?.list || [])
      setMailData(list)
    }
    init()
  }, [mailList])

  return (
    <>
      <div className='flex items-center justify-between gap-2 px-3 py-4'>
        <div className='flex items-center gap-2'>
          <Checkbox
            className='p-0!'
            onChange={(checked) => {
              if (checked) {
                selectAll()
              } else {
                unSelectAll()
              }
            }}
            checked={mailList?.list?.length === 0 ? false : isAllSelected()}
            indeterminate={isPartialSelected()}>
            <span className='ml-3 inline-block text-base font-bold'>{currentFolder.title}</span>
          </Checkbox>
          <Dropdown
            trigger='click'
            triggerProps={{ popupStyle: { maxHeight: '400px', width: '200px' } }}
            droplist={
              <Menu
                onClickMenuItem={(key) => {
                  const item = flatTree(filterList).find((e) => e.value === key)
                  let filter = [...filterKeys]
                  filter[item.key] = item.value
                  onSelectFilter(filter)
                }}>
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
            }>
            <Button className='flex items-center' size='small' type={filterNames.length > 0 ? 'secondary' : 'text'}>
              <IconSort className={`text-base! ${filterNames.length > 0 ? '' : 'text-neutral-600!'}`} />
              {filterNames.length > 0 && (
                <>
                  <span>{filterNames.join('; ')}</span>
                  <IconClose onClick={() => onSelectFilter(['all', 'date_desc'])} />
                </>
              )}
            </Button>
          </Dropdown>
        </div>
        <Space>
          {currentFolder.folder !== 'Star' && selectedRowKeys.length > 0 && (
            <Button size='mini' icon={<IconDelete />} onClick={() => {
              const list = mailData.filter((x) => selectedRowKeys.includes(x.uid))
              onDelMail(list)
            }}>
              {currentFolder.folder === 'Deleted' ? '清空' : '删除'}
            </Button>
          )}
          <span className={`${isTable ? 'mr-10' : ''}`}>共 {mailList?.total || 0} 封</span>
        </Space>
      </div>
      <Spin block loading={listLoading} className='h-[calc(100vh-116px)] overflow-auto' ref={tableRef}>
        {mailData?.map((item) =>
          item.key ? (
            <div
              className='cursor-pointer px-3 pt-2 text-(--color-text-2) underline-offset-3 hover:underline'
              key={item.key}
              onClick={() => onSelectGroup(item)}>
              {item.title}&nbsp;({item.total}&nbsp;封)
            </div>
          ) : (
            <div
              key={item.uid}
              className={`flex w-full cursor-pointer border-b border-(--color-neutral-3) px-3 py-2 hover:bg-(--color-fill-2) ${item?.uid === currentMail?.uid ? 'bg-(--color-fill-2)' : ''} ${!item?.flags?.includes('Seen') && item.folder === 'INBOX' ? 'font-bold' : ''}`}
              onClick={() => currentMail?.uid !== item?.uid && setCurrentMail(item)}>
              <Checkbox
                className='mr-4 p-0!'
                checked={isSelected(item.uid)}
                value={item.uid}
                onChange={(checked, e) => {
                  e.stopPropagation()
                  e.preventDefault()
                  setValueSelected(item.uid, checked)
                }}
              />
              {isTable ? (
                <div className='flex w-full gap-2 overflow-hidden'>
                  <div className='flex w-60 items-center justify-between gap-1.5'>
                    <div className='flex flex-1 gap-1.5 overflow-hidden'>
                      {showMailIcon(item?.flags)}
                      {currentFolder?.folder === 'Sent' ? (
                        <>
                          <IconSent />
                          <div className='flex-1 truncate'>
                            {item?.to_info?.map((t) => t.name).join(', ') || item?.to}
                            {item?.cc_info?.length > 0 ? ',  ' : ''}
                            {item?.cc_info?.map((t) => t.name).join(', ') || item?.cc}
                          </div>
                        </>
                      ) : (
                        item?.from_info?.name || item?.from
                      )}
                    </div>
                    {item.has_attach ? <IconAttachment className='text-base text-gray-400!' /> : ''}
                  </div>
                  <div className='flex w-[calc(100%-456px)] gap-2'>
                    <div className={'max-w-1/2 truncate'}>{item?.subject || ''}</div>
                    <div className={'flex-1 truncate font-light text-gray-400'}>{item?.text || ''}</div>
                  </div>
                  <div className='flex w-50 justify-end gap-2'>
                    <div className='w-20'>{item.size}</div>
                    <div className='w-20'>{formatMailTime(item?.send_time)}</div>
                    {item?.flags?.includes('Flagged') ? (
                      <IconStarSelect className='text-xl!' />
                    ) : (
                      <IconStarUnselect className='text-xl!' />
                    )}
                  </div>
                </div>
              ) : (
                <div className='w-[calc(100%-30px)] leading-6'>
                  <div className='mb-1 flex justify-between gap-2'>
                    <div className='flex w-[calc(100%-72px)] items-center gap-1.5'>
                      {showMailIcon(item?.flags)}
                      {currentFolder?.folder === 'Sent' ? (
                        <>
                          <IconSent />
                          <div className={`${item?.to_info.length > 1 ? 'flex-1' : ''} truncate`}>
                            {item?.to_info?.map((t) => t.name).join(', ') || item?.to}
                            {item?.cc_info?.length > 0 ? ',  ' : ''}
                            {item?.cc_info?.map((t) => t.name).join(', ') || item?.cc}
                          </div>
                        </>
                      ) : (
                        item?.from_info?.name || item?.from
                      )}
                      {item.has_attach ? <IconAttachment className='text-base text-gray-400!' /> : ''}
                    </div>
                    <div className='w-18 text-right'>{formatMailTime(item?.send_time)}</div>
                  </div>
                  <div className='truncate'>{item?.subject || ''}</div>
                  <div className='flex items-center justify-between'>
                    <div className={'flex-1 truncate font-light text-gray-400'}>{item?.text || ''}</div>
                    {item?.flags?.includes('Flagged') && <IconStarSelect className='cursor-pointer text-xl!' />}
                  </div>
                </div>
              )}
            </div>
          )
        )}
        {/* 列表为空 */}
        {mailData?.length === 0 && (
          <div className='flex h-full w-full items-center justify-center'>
            <Empty description='暂无数据' />
          </div>
        )}
      </Spin>
    </>
  )
}
export default ListLayout

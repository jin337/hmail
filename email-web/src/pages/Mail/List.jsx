import { useEffect, useMemo, useRef, useState } from 'react'

import { Button, Checkbox, Divider, Dropdown, Menu, Space, Spin } from '@arco-design/web-react'
import { IconAttachment, IconCheck, IconClose, IconDelete, IconDown, IconSort, IconStar } from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

import { useMailContext } from './MailContext'

import { flatTree, formatMailTime, isSvg } from 'src/utils/index'

import IconMoveFolder from 'src/assets/mail_move_folder.svg'
import IconMailNormal from 'src/assets/mail_normal.svg'
import IconMailOpen from 'src/assets/mail_open.svg'
import IconMailRead from 'src/assets/mail_read.svg'
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
    setSelectedRowKeys,
    onDelMail,
    onStar,
    onFlagMail,
    flagList,
    onChangeMailFlag,
    onMoveMail,
    moveList,
    onRead,
    isMove,
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
    setCurrentMail(null)
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
    setSelectedRowKeys(selected)
    onChangeMailFlag(selected)
  }, [selected])

  // 监控邮件列表
  const prevFolderRef = useRef(currentFolder?.folder)

  useEffect(() => {
    const init = () => {
      const currentFolderId = currentFolder?.folder
      if (prevFolderRef.current !== currentFolderId) {
        unSelectAll()
      }
      prevFolderRef.current = currentFolderId

      const list = groupMailByTime(mailList?.list || [])
      setMailData(list)
    }
    init()
  }, [mailList?.list, currentFolder?.folder, filterKeys])

  return (
    <>
      <div className='mail-menu flex items-center justify-between gap-2 border-b border-gray-200 px-4 py-2.5'>
        <div className='flex h-7 items-center gap-2'>
          <div className='flex w-63 items-center gap-2'>
            <Checkbox
              className='p-0! text-nowrap'
              onChange={(checked) => {
                if (checked) {
                  if (selected.length > 0) {
                    unSelectAll()
                  } else {
                    selectAll()
                  }
                } else {
                  unSelectAll()
                }
              }}
              checked={mailList?.list?.length === 0 ? false : isAllSelected()}
              indeterminate={isPartialSelected()}>
              {selected.length ? (
                <span className='ml-3 inline-block font-bold'>已选中 {selected.length} 封邮件</span>
              ) : (
                <span className='ml-3 inline-block text-base font-bold text-nowrap'>{currentFolder.title}</span>
              )}
            </Checkbox>
            {selected.length === 0 && (
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
                    {filterList.map((group, groupIdx) => {
                      if (group.value === 'divider') {
                        return <Divider style={{ margin: '4px 0' }} />
                      } else {
                        return (
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
                        )
                      }
                    })}
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
            )}
          </div>
          {currentFolder.folder !== 'Star' && isTable && !isMove && (
            <Space>
              <Button
                size='small'
                icon={<IconDelete />}
                onClick={() => {
                  onDelMail(mailData.filter((x) => selected.includes(x.uid)))
                }}>
                {currentFolder.folder === 'Deleted' ? '彻底删除' : '删除'}
              </Button>
              <Button size='small' onClick={() => onRead({ uids: selected, folder: currentFolder.folder, type: 1 })}>
                <div className='flex items-center gap-1'>
                  <IconMailRead />
                  全部已读
                </div>
              </Button>
              <Dropdown
                trigger='click'
                triggerProps={{ autoAlignPopupWidth: true }}
                droplist={
                  <Menu onClickMenuItem={(e) => onFlagMail({ uids: selected, to: e, from: currentFolder.folder })}>
                    {flagList.map((e, i) =>
                      e.flag === 'divider' ? (
                        <Divider key={i + `_divider`} style={{ margin: '4px 0' }} />
                      ) : (
                        <Menu.Item key={e.flag + '_' + e.key}>{e.title}</Menu.Item>
                      )
                    )}
                  </Menu>
                }>
                <Button size='small'>
                  <div className='flex items-center gap-1'>
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
                  <Menu onClickMenuItem={(e) => onMoveMail({ uids: selected, to: e, from: currentFolder.folder })}>
                    {moveList
                      .filter((e) => ![currentFolder.folder].includes(e.folder))
                      .map((e) => (
                        <Menu.Item key={e.folder}>{e.title}</Menu.Item>
                      ))}
                  </Menu>
                }>
                <Button size='small'>
                  <div className='flex items-center gap-1'>
                    <IconMoveFolder />
                    移动到
                    <IconDown />
                  </div>
                </Button>
              </Dropdown>
            </Space>
          )}
        </div>
        <span className={`${isTable && !isMove ? 'mr-10' : ''}`}>共 {mailList?.total || 0} 封</span>
      </div>

      {/* 邮件列表 */}
      <Spin block loading={listLoading} className='mail-list h-[calc(100vh-116px)] overflow-y-auto px-1' ref={tableRef}>
        {mailData?.map((item, index) =>
          item.key ? (
            <div
              className={`cursor-pointer px-3 underline-offset-3 ${index == 0 ? 'pt-2' : 'pt-3.5'} mb-1 text-(--color-text-2)`}
              key={item.key}>
              <span className='hover:underline' onClick={() => onSelectGroup(item)}>
                <span className='mr-1 font-bold'>{item.title}</span>({item.total}&nbsp;封)
              </span>
            </div>
          ) : (
            <div
              key={item.uid}
              className={`mail-item box-border flex w-full cursor-pointer rounded px-3 hover:bg-(--color-fill-2) ${selected?.includes(item?.uid) || item?.uid === currentMail?.uid ? ' selelct-mail' : ''} ${!item?.flags?.includes('Seen') ? ' font-bold' : ''} ${isTable ? ' items-center py-0.5' : 'py-2'}`}
              onClick={(e) => {
                // 排除干扰点击
                const targetElement = e?.target

                const isCheckboxClick = targetElement
                  ? targetElement?.classList.contains('arco-checkbox') ||
                    targetElement?.classList.contains('arco-checkbox-input') ||
                    targetElement?.closest('.arco-checkbox')
                  : false

                const isStar = isSvg(e)

                if (!isCheckboxClick && isSelected(item?.uid)) {
                  unSelectAll()
                }
                if (currentMail?.uid !== item?.uid) {
                  // 排除非跳转项
                  if (isCheckboxClick || isStar) return
                  setCurrentMail(item)
                  unSelectAll()
                }
              }}>
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
                // 列表模式
                <div className='flex w-full items-center gap-2 overflow-hidden'>
                  <div className='flex w-55 items-center justify-between gap-1.5'>
                    <div className='flex flex-1 items-center gap-1.5 overflow-hidden'>
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
                  <div className='flex w-[calc(100%-436px)] items-center gap-2'>
                    <div className={'max-w-1/2 truncate'}>{item?.subject || ''}</div>
                    <div className={'flex-1 truncate text-gray-500/60'}>{item?.text || ''}</div>
                  </div>
                  <div className='flex w-50 items-center justify-end gap-2'>
                    <div className='w-20'>{item.size}</div>
                    <div className='w-20'>{formatMailTime(item?.send_time)}</div>
                    <Button
                      onClick={(e) => {
                        e.stopPropagation()
                        e.preventDefault()
                        const type = item?.flags?.includes('Flagged') ? 2 : 1
                        onStar({ uids: [item.uid], folder: item.folder, type })
                      }}
                      type='text'
                      size='mini'
                      icon={item?.flags?.includes('Flagged') ? <IconStarSelect /> : <IconStarUnselect />}
                    />
                  </div>
                </div>
              ) : (
                // 详情模式
                <div className='w-[calc(100%-30px)] leading-6'>
                  <div className='flex justify-between gap-2'>
                    <div className='flex w-[calc(100%-72px)] items-center gap-1.5'>
                      {showMailIcon(item?.flags)}
                      {currentFolder?.folder === 'Sent' ? (
                        <>
                          <IconSent />
                          <div className={`${item?.to_info?.length > 1 ? 'flex-1' : ''} truncate`}>
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
                  <div className='flex h-6 items-center justify-between'>
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
          <div className='flex h-[calc(100vh-116px)] w-full items-center justify-center text-lg text-gray-600/80'>
            暂无邮件待处理
          </div>
        )}
      </Spin>
    </>
  )
}
export default ListLayout

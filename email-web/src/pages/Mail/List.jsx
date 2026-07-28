import { useMemo } from 'react'

import { Button, Checkbox, Dropdown, Menu } from '@arco-design/web-react'
import { IconAttachment, IconCheck, IconClose, IconSort } from '@arco-design/web-react/icon'

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

const ListLayout = () => {
  const { currentFolder, mailList, filterList, filterKeys, onSelectFilter, currentMail, setCurrentMail, isTable } =
    useMailContext()

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

  return (
    <>
      <div className='flex items-center justify-between gap-2 px-3 py-4'>
        <div className='flex items-center gap-2'>
          <Checkbox className='p-0!'>
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
        <span className={`${isTable ? 'mr-10' : ''}`}>共 {mailList?.total || 0} 封</span>
      </div>
      <div className='h-[calc(100vh-116px)] overflow-auto'>
        {mailList?.list?.map((item) => (
          <div
            key={item.uid}
            className={`flex w-full cursor-pointer border-b border-(--color-neutral-3) px-3! py-2! hover:bg-(--color-fill-2) ${item?.uid === currentMail?.uid ? 'bg-(--color-fill-2)' : ''}`}
            onClick={() => setCurrentMail(item)}>
            <Checkbox className='mr-4 p-0!' />
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
                      item?.from_info.name || item?.from
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
                      item?.from_info.name || item?.from
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
        ))}
      </div>
    </>
  )
}
export default ListLayout

import { cloneElement } from 'react'

import { Button, Menu } from '@arco-design/web-react'
import { IconClose, IconEdit } from '@arco-design/web-react/icon'

import { useMailContext } from './MailContext'

const MenuLayout = () => {
  const { folderList, currentFolder, setCurrentFolder, onEdit, onCloseEdit } = useMailContext()

  // 点击菜单
  const onSelectMenu = (key) => {
    const item = folderList.find((item) => item.key === key)
    setCurrentFolder(item)
  }

  return (
    <>
      <div className='p-4'>
        <Button type='primary rounded!' icon={<IconEdit />} long onClick={onEdit}>
          写信
        </Button>
      </div>

      <Menu className='mail-wrap bg-transparent! px-2' selectedKeys={[currentFolder?.key || '']} onClickMenuItem={onSelectMenu}>
        {folderList?.map((item) => (
          <Menu.Item key={item.key} className='leading-8! text-(--color-text-1)!'>
            <div className='flex items-center'>
              {cloneElement(item.icon, { className: 'text-(--color-text-1)! text-lg' })}
              <span className='inline-block w-27 overflow-hidden align-middle text-ellipsis whitespace-nowrap'>{item.title}</span>
            </div>
            {item?.key === 'inbox' && item?.total > 0 && <span className='font-medium text-blue-600'>{item.total}</span>}
            {item?.key === 'compose' && (
              <Button
                className='m-0!'
                type='text'
                size='mini'
                icon={<IconClose className='m-0! text-gray-500!' />}
                onClick={(e) => {
                  e.stopPropagation()
                  onCloseEdit()
                }}></Button>
            )}
          </Menu.Item>
        ))}
      </Menu>
    </>
  )
}
export default MenuLayout

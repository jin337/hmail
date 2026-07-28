import { } from 'react'

import { Button, Dropdown, Menu } from '@arco-design/web-react'
import { IconDelete, IconDown, IconRedo, IconReply, IconStar } from '@arco-design/web-react/icon'

import { useMailContext } from './MailContext'

import IconMoveFolder from 'src/assets/mail_move_folder.svg'

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

const Detail = () => {
  const { currentMail, currentFolder, moveList } = useMailContext()

  if (currentMail)
    return (
      <>
        <div className='flex items-center justify-between gap-2 border-b border-gray-200 p-4'>
          <div className='flex flex-wrap items-center gap-2'>
            <Button size='small' icon={<IconDelete />}>
              删除
            </Button>
            <Button size='small' icon={<IconReply />}>
              回复
            </Button>
            <Button size='small' icon={<IconRedo />}>
              转发
            </Button>
            <Dropdown
              triggerProps={{ autoAlignPopupWidth: true }}
              trigger='click'
              droplist={<Menu>{FlagList(currentMail?.flags)}</Menu>}>
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
                <Menu>
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
        </div>
      </>
    )

  return <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
}
export default Detail

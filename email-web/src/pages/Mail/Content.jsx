import { Button, Divider, Dropdown, Layout, Menu } from '@arco-design/web-react'
import {
  IconArrowLeft,
  IconDelete,
  IconDown,
  IconLayout,
  IconLeft,
  IconMenu,
  IconRedo,
  IconReply,
  IconRight,
  IconStar,
} from '@arco-design/web-react/icon'

// 组件
import Detail from './Detail'
import List from './List'

// 图标
import IconMoveFolder from 'src/assets/mail_move_folder.svg'
import IconMailRead from 'src/assets/mail_read.svg'

import { useMailContext } from './MailContext'

const Content = () => {
  const {
    isTable,
    setIsTable,
    currentMail,
    setCurrentMail,
    onCutMail,
    selectedRowKeys,
    mailList,
    onDelMail,
    currentFolder,
    onReplyForward,
    onRead,
    onFlagMail,
    flagList,
    onMoveMail,
    moveList,
    setIsMove,
    isMove,
  } = useMailContext()

  // 切换模式
  const cutTable = () => {
    setCurrentMail(null)
    if (isMove) {
      setIsTable(true)
    } else {
      setIsTable(!isTable)
    }
    setIsMove(false)
    localStorage.setItem('isTable', !isTable)
  }

  const onMoving = (_, { width }) => {
    if (width > 740) {
      setIsMove(true)
      setIsTable(true)
    } else {
      setIsMove(false)
      setIsTable(false)
    }
  }

  return (
    <Layout className='relative rounded-t-xl bg-white'>
      <Layout.Sider
        resizeBoxProps={{
          directions: ['right'],
          resizeTriggers: {
            right: (
              <div className='h-full w-full'>
                <div className='resizebox-custom-trigger-line h-full w-px flex-1 bg-gray-200' />
              </div>
            ),
          },
          onMoving,
        }}
        width={isTable && !isMove ? (currentMail ? 0 : '100%') : 410}
        className={`shadow-none!`}>
        {/* 邮件列表 */}
        <List />
      </Layout.Sider>

      <Layout.Content>
        {/* 切换模式按钮 */}
        {!(isTable && !isMove && currentMail) && (
          <div className='absolute top-2.5 right-4 z-20'>
            <Button size='small' onClick={cutTable} icon={isTable && !isMove ? <IconLayout /> : <IconMenu />}></Button>
          </div>
        )}
        {/* 操作按钮 */}
        <div className='flex items-center justify-between gap-2 border-b border-gray-200 px-4 py-2.5'>
          <div className='flex flex-wrap items-center gap-2'>
            {isTable && !isMove && currentMail && (
              <Button size='small' icon={<IconArrowLeft />} onClick={() => setCurrentMail()}>
                返回
              </Button>
            )}
            <Button
              size='small'
              icon={<IconDelete />}
              onClick={() => {
                const list = (mailList?.list || []).filter((x) => selectedRowKeys.includes(x.uid))
                onDelMail(selectedRowKeys.length > 0 ? list : [currentMail])
              }}>
              {currentFolder?.folder === 'Deleted' ? '彻底删除' : '删除'}
            </Button>
            {currentMail?.uid && selectedRowKeys.length === 0 ? (
              <>
                <Button size='small' icon={<IconReply />} onClick={() => onReplyForward('is_reply')}>
                  回复
                </Button>
                <Button size='small' icon={<IconRedo />} onClick={() => onReplyForward('is_forward')}>
                  转发
                </Button>
              </>
            ) : (
              <Button
                size='small'
                onClick={() =>
                  onRead({
                    uids: selectedRowKeys,
                    folder: selectedRowKeys.length > 0 ? currentFolder.folder : currentMail?.folder,
                    type: 1,
                  })
                }>
                <div className='flex items-center gap-1'>
                  <IconMailRead />
                  全部已读
                </div>
              </Button>
            )}
            <Dropdown
              triggerProps={{ autoAlignPopupWidth: true }}
              trigger='click'
              droplist={
                <Menu
                  onClickMenuItem={(e) => {
                    onFlagMail({
                      to: e,
                      uids: selectedRowKeys.length > 0 ? selectedRowKeys : [currentMail?.uid],
                      from: selectedRowKeys.length > 0 ? currentFolder.folder : currentMail?.folder,
                    })
                  }}>
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
                <Menu
                  onClickMenuItem={(e) =>
                    onMoveMail({
                      to: e,
                      uids: selectedRowKeys.length > 0 ? selectedRowKeys : [currentMail?.uid],
                      from: selectedRowKeys.length > 0 ? currentFolder.folder : currentMail?.folder,
                    })
                  }>
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
          </div>

          {/* 切换邮件 */}
          {isTable && !isMove && currentMail && (
            <Button.Group className='flex!' type='text'>
              <Button
                size='small'
                icon={<IconLeft />}
                disabled={currentMail?.uid === mailList?.list[0]?.uid}
                onClick={() => onCutMail(currentMail, 'prev')}>
                上一封
              </Button>
              <Button
                size='small'
                disabled={currentMail?.uid === mailList?.list[mailList?.list?.length - 1]?.uid}
                onClick={() => onCutMail(currentMail, 'next')}>
                下一封
                <IconRight />
              </Button>
            </Button.Group>
          )}
        </div>
        {/* 邮件详情 */}
        <Detail />
      </Layout.Content>
    </Layout>
  )
}
export default Content

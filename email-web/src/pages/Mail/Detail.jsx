import { } from 'react'

import { Button, Card, Divider, Dropdown, Menu, Popover, Space, Typography } from '@arco-design/web-react'
import {
  IconArrowLeft,
  IconAttachment,
  IconClockCircle,
  IconDelete,
  IconDown,
  IconEye,
  IconLeft,
  IconPlus,
  IconRedo,
  IconReply,
  IconRight,
  IconStar,
  IconToBottom,
} from '@arco-design/web-react/icon'

import dayjs from 'dayjs'

import request from 'src/api/request'
import { useMailContext } from './MailContext'

import IconAudio from 'src/assets/file_aduio.svg'
import IconExcel from 'src/assets/file_excel.svg'
import IconImage from 'src/assets/file_image.svg'
import IconPdf from 'src/assets/file_pdf.svg'
import IconPpt from 'src/assets/file_ppt.svg'
import IconText from 'src/assets/file_text.svg'
import IconVideo from 'src/assets/file_video.svg'
import IconWord from 'src/assets/file_word.svg'
import IconZip from 'src/assets/file_zip.svg'

import IconMoveFolder from 'src/assets/mail_move_folder.svg'
import IconStarUnselect from 'src/assets/mail_star.svg'
import IconStarSelect from 'src/assets/mail_star_open.svg'

import AvatarImage from 'src/components/AvatarImage'

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

  if (!flags?.includes('Seen')) {
    list[0].title = '已读邮件'
    list[0].key = 1
  }

  return list.map((e) => <Menu.Item key={e.flag + '_' + e.key}>{e.title}</Menu.Item>)
}

const Detail = () => {
  const {
    baseUrl,
    userInfo,
    mailList,
    currentMail,
    setCurrentMail,
    onCutMail,
    currentFolder,
    moveList,
    isTable,
    contactList,
    onEditContact,
    onStar,
    onRead,
    onDelMail,
    onMoveMail,
    onReplyForward
  } = useMailContext()

  // 标记邮件
  const onFlagMail = async (e) => {
    const key = e.split('_')
    if (key[0] === 'Seen') {
      onRead(currentMail, Number(key[1]))
    }

    if (key[0] === 'Flagged') {
      onStar(currentMail)
    }
  }

  // 预览附件
  const onPreviewAttachment = (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
      file_name: item.file_name,
      file_type: item.file_type,
    }
    const jsonString = JSON.stringify(params)
    const base64Str = btoa(encodeURIComponent(jsonString))
    window.open(`/web/preview?preview=${base64Str}`, '_blank')
  }

  // 下载附件
  const onDownloadAttachment = async (item) => {
    const params = {
      uid: currentMail.uid,
      part_id: item.part_id,
      folder: currentFolder.folder,
    }
    const res = await request.post('/api/mail/download', params, {
      responseType: 'blob',
    })
    const url = window.URL.createObjectURL(res)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', item.file_name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  if (currentMail)
    return (
      <>
        <div className='flex items-center justify-between gap-2 border-b border-gray-200 p-4'>
          {/* 操作按钮 */}
          <div className='flex flex-wrap items-center gap-2'>
            {isTable && currentMail && (
              <Button size='small' icon={<IconArrowLeft />} onClick={() => setCurrentMail()}>
                返回
              </Button>
            )}
            <Button size='small' icon={<IconDelete />} onClick={() => onDelMail([currentMail.uid])}>
              {currentFolder.folder === 'Deleted' ? '彻底删除' : '删除'}
            </Button>
            <Button size='small' icon={<IconReply />} onClick={() => onReplyForward('is_reply')}>
              回复
            </Button>
            <Button size='small' icon={<IconRedo />} onClick={() => onReplyForward('is_forward')}>
              转发
            </Button>
            <Dropdown
              triggerProps={{ autoAlignPopupWidth: true }}
              trigger='click'
              droplist={<Menu onClickMenuItem={onFlagMail}>{FlagList(currentMail?.flags)}</Menu>}>
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
                <Menu onClickMenuItem={onMoveMail}>
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

          {/* 切换邮件 */}
          {isTable && (
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
        <div className='h-[calc(100vh-117px)] overflow-y-auto p-4'>
          <div className='mb-4 flex items-center gap-2'>
            <span className='text-lg font-bold'>{currentMail.subject}</span>
            <Button size='mini' type='text' onClick={() => onStar(currentMail)}>
              {currentMail?.flags?.includes('Flagged') ? (
                <IconStarSelect className='text-xl!' />
              ) : (
                <IconStarUnselect className='text-xl!' />
              )}
            </Button>
          </div>
          <div className='mb-4 flex items-start gap-3'>
            <AvatarImage baseUrl={baseUrl} email={currentMail?.from_info?.email} name={currentMail?.from_info?.name} />
            <div className='flex-1 text-sm'>
              <Popover
                position='bl'
                trigger='hover'
                key={currentMail.from}
                triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                content={
                  <div>
                    <div className='flex gap-2'>
                      <AvatarImage baseUrl={baseUrl} email={currentMail?.from_info?.email} name={currentMail?.from_info?.name} />
                      <div>
                        <div className='flex items-center gap-2 font-bold'>{currentMail?.from_info?.name}</div>
                        <Typography.Text copyable>{currentMail.from}</Typography.Text>
                      </div>
                    </div>
                    {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(currentMail.from) && (
                      <div className={'mt-2'}>
                        <Button
                          type='primary'
                          size='small'
                          long
                          icon={<IconPlus />}
                          onClick={() =>
                            onEditContact({
                              name: currentMail?.from_info?.name,
                              email: currentMail.from,
                              prefix: 'user_contact',
                            })
                          }>
                          添加联系人
                        </Button>
                      </div>
                    )}
                  </div>
                }>
                <div className='mb-1'>
                  <strong>{currentMail?.from_info?.name}</strong>
                  <span className='text-gray-400'>&nbsp;&lt;{currentMail.from}&gt;</span>
                </div>
              </Popover>
              <div className='flex flex-wrap items-start justify-between gap-2'>
                <div className='flex-1'>
                  <div className='mb-1 flex'>
                    <div className='whitespace-nowrap text-gray-400'>收件人</div>
                    <div className='flex flex-wrap'>
                      {currentMail?.to_info?.map((e, index) => (
                        <Popover
                          position='bl'
                          trigger='hover'
                          key={e.email}
                          triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                          content={
                            <div>
                              <div className='flex gap-2'>
                                <AvatarImage baseUrl={baseUrl} email={e?.email} name={e?.name} />
                                <div>
                                  <div className='flex items-center gap-2 font-bold'>{e?.name}</div>
                                  <Typography.Text copyable>{e?.email}</Typography.Text>
                                </div>
                              </div>
                              {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(e?.email) && (
                                <div className={'mt-2'}>
                                  <Button
                                    type='primary'
                                    size='small'
                                    long
                                    icon={<IconPlus />}
                                    onClick={() =>
                                      onEditContact({
                                        name: e?.name,
                                        email: e?.email,
                                        prefix: 'user_contact',
                                      })
                                    }>
                                    添加联系人
                                  </Button>
                                </div>
                              )}
                            </div>
                          }>
                          <span className='mr-1 ml-3'>{e.name}</span>
                          <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                          {index !== currentMail?.to_info?.length - 1 && <span className='text-gray-400'>,</span>}
                        </Popover>
                      ))}
                    </div>
                  </div>
                  {currentMail?.cc && (
                    <div className='flex items-center'>
                      <div className='text-gray-400'>抄送</div>
                      <div className='flex flex-wrap'>
                        {currentMail?.cc_info?.map((e, index) => (
                          <Popover
                            position='bl'
                            trigger='hover'
                            key={e.email}
                            triggerProps={{ mouseEnterDelay: 500, showArrow: false }}
                            content={
                              <div>
                                <div className='flex gap-2'>
                                  <AvatarImage baseUrl={baseUrl} email={e?.email} name={e?.name} />
                                  <div>
                                    <div className='flex items-center gap-2 font-bold'>{e?.name}</div>
                                    <Typography.Text copyable>{e?.email}</Typography.Text>
                                  </div>
                                </div>
                                {![...contactList, { email: userInfo.email }]?.map((e) => e.email).includes(e?.email) && (
                                  <div className={'mt-2'}>
                                    <Button
                                      type='primary'
                                      size='small'
                                      long
                                      icon={<IconPlus />}
                                      onClick={() =>
                                        onEditContact({
                                          name: e?.name,
                                          email: e?.email,
                                          prefix: 'user_contact',
                                        })
                                      }>
                                      添加联系人
                                    </Button>
                                  </div>
                                )}
                              </div>
                            }>
                            <span className='mr-1 ml-3'>{e.name}</span>
                            <span className='text-gray-400'>&lt;{e.email}&gt;</span>
                            {index !== currentMail?.cc_info?.length - 1 && <span className='text-gray-400'>,</span>}
                          </Popover>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
                <div className='w-45 text-right text-gray-400'>
                  {dayjs(currentMail?.send_time).format('YYYY年MM月DD日 HH:mm:ss') || ''}
                </div>
              </div>
            </div>
          </div>

          <Divider />
          {/* 定时邮件 */}
          {!['0001-01-01T00:00:00Z', ''].includes(currentMail.schedule) && (
            <div className='mb-5 flex items-center rounded bg-[#e6edf5] px-4 py-2'>
              <IconClockCircle className='mr-1 text-blue-500!' />
              此邮件是定时邮件，将在
              <span className='mx-2 text-blue-500'>{dayjs(currentMail.schedule).format('YYYY年MM月DD日 HH:mm:ss')}</span>
              发出。
              <Button type='text' size='mini'>
                取消发送
              </Button>
            </div>
          )}

          {/* 邮件内容 */}
          <div
            className='mail-detail'
            dangerouslySetInnerHTML={{
              __html: currentMail.detail?.content || '<div class="text-gray-500">暂无邮件内容</div>',
            }}
          />

          {/* 附件 */}
          {currentMail?.has_attach && (
            <Card
              className='mt-10'
              title={
                <>
                  <IconAttachment className='mr-1' />
                  {currentMail?.detail?.attachments?.length}个 附件 {currentMail?.detail?.attach_size}
                </>
              }>
              <div className='flex flex-col gap-2'>
                {currentMail?.detail?.attachments?.map((item, index) => (
                  <div key={index} className='flex items-center justify-between gap-2 bg-gray-100 p-2 hover:bg-gray-200'>
                    <div className='flex flex-1 items-center'>
                      <span className='mr-2'>
                        {item?.file_type === 'video' && <IconVideo />}
                        {item?.file_type === 'audio' && <IconAudio />}
                        {item?.file_type === 'zip' && <IconZip />}
                        {item?.file_type === 'image' && <IconImage />}

                        {item?.file_type === 'ppt' && <IconPpt />}
                        {item?.file_type === 'pdf' && <IconPdf />}
                        {item?.file_type === 'excel' && <IconExcel />}
                        {item?.file_type === 'word' && <IconWord />}

                        {item?.file_type === 'text' && <IconText />}
                      </span>
                      {item.file_name}
                      <span className='text-gray-400'>（{item.size}）</span>
                    </div>
                    <Space>
                      <Button type='text' size='small' onClick={() => onPreviewAttachment(item)}>
                        <IconEye />
                        预览
                      </Button>
                      <Button type='text' size='small' onClick={() => onDownloadAttachment(item)}>
                        <IconToBottom />
                        下载
                      </Button>
                    </Space>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>
      </>
    )

  return <div className='flex h-full items-center justify-center text-gray-300'>请在左侧选择一封邮件查看详情</div>
}
export default Detail

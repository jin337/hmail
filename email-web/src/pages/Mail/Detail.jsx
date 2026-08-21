import { useEffect, useRef, useState } from 'react'

import { Button, Card, Divider, Empty, Popover, Space, Spin, Typography } from '@arco-design/web-react'
import { IconAttachment, IconClockCircle, IconEmail, IconEye, IconPlus, IconToBottom } from '@arco-design/web-react/icon'

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

import IconStarUnselect from 'src/assets/mail_star.svg'
import IconStarSelect from 'src/assets/mail_star_open.svg'

import AvatarImage from 'src/components/AvatarImage'

const Detail = () => {
  const {
    baseUrl,
    userInfo,
    mailList,
    currentMail,
    mailLoading,
    currentFolder,
    contactList,
    onEditContact,
    onEdit,
    onStar,
    onUnSchedule,
    selectedRowKeys,
  } = useMailContext()

  const detailRef = useRef()

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

  // 选中邮件
  const [selectItems, setSelectItems] = useState([])
  useEffect(() => {
    const init = () => {
      if (!selectedRowKeys?.length) {
        setSelectItems([])
        return
      }

      const list = mailList.list || []
      const allSelected = []
      for (let i = selectedRowKeys.length - 1; i >= 0; i--) {
        const uid = selectedRowKeys[i]
        const findItem = list.find((m) => m.uid === uid)
        if (findItem) {
          allSelected.push(findItem)
        }
      }

      // 最多保留3条
      const next = allSelected.slice(0, 3)
      setSelectItems(next)
    }
    init()
  }, [mailList, selectedRowKeys])

  // 邮件内容点击邮箱地址
  useEffect(() => {
    if (!detailRef.current) return
    const container = detailRef.current

    const onMailLink = (e) => {
      const aDom = e.target.closest('a[data-mail-email]')
      if (!aDom) return

      e.preventDefault()
      e.stopPropagation()

      const email = aDom.dataset.mailEmail
      if (email) {
        const name = email.split('@')[0]
        onEdit({
          to_info: [{ label: name, value: email }],
        })
      }
    }

    // 只给外层容器绑定一次click
    container.addEventListener('click', onMailLink)

    return () => {
      container.removeEventListener('click', onMailLink)
    }
  }, [currentMail, onEdit])

  if (currentMail || selectedRowKeys?.length > 0) {
    return (
      <>
        {/* 邮件详情 */}
        {selectedRowKeys.length > 0 ? (
          <div className='relative h-[calc(100vh-117px)] w-full p-4'>
            {selectItems.map((item, index) => {
              return (
                <div
                  className={`top-card absolute h-36 rounded-lg border border-gray-200 bg-white p-4 shadow-md transition-all duration-300 ease-in-out`}
                  style={{
                    left: `${(index + 1) * 10}px`,
                    right: `${(index + 1) * 10}px`,
                    top: `${(index + 1) * 10}px`,
                    zIndex: `${selectItems?.length - index}`,
                  }}
                  key={index}>
                  <div className='mb-1 text-lg font-bold'>{item.subject}</div>{' '}
                  <div className='mb-10 truncate text-gray-600/90'>{item.text}</div>{' '}
                  <div className='text-[13px] font-light text-gray-400'>
                    {item.from_info?.name}·{dayjs(item?.send_time).format('YYYY年MM月DD日 HH:mm:ss') || ''}{' '}
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          <Spin block loading={mailLoading} className='h-[calc(100vh-117px)] overflow-y-auto p-4'>
            <div className='mb-4 flex items-center gap-2'>
              <span className='text-lg font-bold'>{currentMail.subject}</span>
              <Button
                size='mini'
                type='text'
                onClick={() => {
                  const type = currentMail?.flags?.includes('Flagged') ? 2 : 1
                  onStar({ uids: [currentMail.uid], folder: currentMail.folder, type: type })
                }}>
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
                        <AvatarImage
                          baseUrl={baseUrl}
                          email={currentMail?.from_info?.email}
                          name={currentMail?.from_info?.name}
                        />
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
                    <span className='ml-2 text-gray-400'>&lt;{currentMail.from}&gt;</span>
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
                            key={e.email + '_' + index}
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
                              key={e.email + '_' + index}
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
                <Button type='text' size='mini' onClick={() => onUnSchedule(currentMail)}>
                  取消发送
                </Button>
              </div>
            )}
            {/* 邮件内容 */}
            <div
              ref={detailRef}
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
          </Spin>
        )}
      </>
    )
  }

  return (
    <div className='flex h-full items-center justify-center text-gray-300'>
      <Empty
        description={<span className='text-base'>未选中任何邮件</span>}
        icon={<IconEmail className='stroke-1! text-[140px]!' />}
      />
    </div>
  )
}
export default Detail

import dayjs from 'dayjs'

import { transHtmlAttrs } from 'src/utils/index'

// 从邮件正文里提取内联图片 id（image_1.png -> 1）
export const getImageIds = (html) => {
  if (!html) return []

  const parser = new DOMParser()
  const doc = parser.parseFromString(html, 'text/html')

  const images = doc.querySelectorAll('img[data-href]')

  const ids = []
  images.forEach((img) => {
    const href = img.getAttribute('data-href')
    const match = href.match(/image_([\d.]+)\.(png|jpg|jpeg|gif|webp|bmp)$/i)
    if (match && match[1]) {
      ids.push(match[1])
    }
  })

  return ids
}

// 详情正文：data-href 与 src 互换，并把 <邮箱> 变成可点击的 mailto 链接
export const normalizeDetailContent = (content) => {
  const swapped = transHtmlAttrs(content, 'src', 'data-href')
  return swapped?.replace(
    /<span style="color: rgb\(149, 157, 166\);">(&lt;([a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)&gt;)<\/span>/g,
    (match, fullInner, email) => {
      return `&lt;<a href="mailto:${email}" target="_blank" rel="noopener noreferrer" data-mail-email="${email}" title="给TA写信">${email}</a>&gt;`
    }
  )
}

// 回复/转发时拼接的原始邮件引用块
export const buildQuotedContent = (currentMail) => {
  const to_reply = currentMail?.to_info
    ?.map(
      (t) =>
        `<span style="color: rgb(0, 0, 0);">${t.name}</span> <span style="color: rgb(149, 157, 166);">&lt;${t.email}&gt;</span>`
    )
    .join(', ')

  const cc_reply = currentMail?.cc
    ? currentMail?.cc_info
        ?.map(
          (t) =>
            `<span style="color: rgb(0, 0, 0);">${t.name}</span> <span style="color: rgb(149, 157, 166);">&lt;${t.email}&gt;</span>`
        )
        .join(', ')
    : ''

  return `<div><br></div>
    <div><br></div>
    <article>
      <div style="display:flex;align-items:center;padding-top:8px">
        <div style="color:#959DA6;font-size:12px;line-height:30px">原始邮件</div>
        <hr style="border-width: medium; border-style: none; border-color: currentcolor; border-image: none;flex-grow:1;border-top:1px solid rgba(21, 46, 74, 0.07);margin-left:8px">
            </div>
      <div style="line-height: 20px; border-radius: 6px; background-color: rgba(20, 46, 77, 0.05); color: rgb(0, 0, 0); margin: 0px; padding: 8px; width: 100%;">
        <div style="line-height: 20px; font-size: 12px;">
          发件人：<span style="color: rgb(0, 0, 0);">${currentMail?.from_info?.name}</span><span style="color: rgb(149, 157, 166);">&lt;${currentMail?.from}&gt;</span>
        </div>
        <div style="line-height: 20px; font-size: 12px;">
          发件时间：<span style="color: rgb(0, 0, 0);">${dayjs(currentMail?.date).format('YYYY年MM月DD日 HH:mm:ss')}</span>
        </div>
        <div style="line-height: 20px; font-size: 12px;">收件人：${to_reply}</div>
        ${currentMail?.cc ? `<div style="line-height: 20px; font-size: 12px;">抄送：${cc_reply}</div>` : ''}
        <div style="line-height: 20px; font-size: 12px;">
          主题：<span style="color: rgb(0, 0, 0);">${currentMail?.subject}</span>
        </div>
      </div>
      <div><br></div>
      <div>${currentMail?.detail?.content}</div>
    </article>`
}

// 组装发送/存草稿的 FormData
export const createMailFormData = ({ rest, detail, type, customTime }) => {
  const formData = new FormData()
  const to = rest.to_info.map((e) => e.value)
  const cc = rest?.cc_info?.map((e) => e.value) || ''

  formData.append('to', to)
  formData.append('cc', cc)
  formData.append('subject', rest.subject)

  if (rest?.is_reply || rest?.is_forward) {
    formData.append('folder', rest?.folder)
  } else {
    formData.append('folder', rest?.uid ? 'Drafts' : type)
  }

  if (rest?.uid) {
    formData.append('uid', rest.uid)
  }
  if (customTime) {
    formData.append('x-schedule-send', customTime)
  }

  // 回复
  if (rest?.is_reply) {
    const references = rest.references + ' ' + rest?.message_id
    formData.append('in_reply_to', rest?.message_id)
    formData.append('references', references)
  }

  // 转发
  if (rest?.is_forward) {
    const references = rest.references + ' ' + rest?.message_id
    formData.append('references', references)
  }

  // 邮件内容
  const content = transHtmlAttrs(detail.content, 'data-href', 'src')
  formData.append('content', content)

  // 提取 HTML 中的图片 ID
  const imageIds = getImageIds(content)

  // 附件
  const partIds = []
  detail?.attachments?.forEach((file) => {
    if (file?.part_id) {
      partIds.push(file.part_id)
    } else {
      formData.append('files', file?.originFile)
    }
  })

  const allPartIds = [...partIds, ...imageIds]
  if (allPartIds.length > 0) {
    formData.append('part_ids', allPartIds.join(','))
  }

  return formData
}

// 邮件列表按时间分组
export const groupMailByTime = (list, keys) => {
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
  let groupMap = [
    { key: 'today', title: '今天' },
    { key: 'yesterday', title: '昨日' },
    { key: 'thisWeek', title: '周一' },
    { key: 'lastWeek', title: '上周' },
    { key: 'older', title: '更早' },
  ]

  // 由旧到新
  if (keys.includes('date_asc')) {
    groupMap.reverse()
  }

  const list2 = []
  groupMap.forEach(({ key, title }) => {
    const arr = groups[key]
    if (arr.length === 0) return
    list2.push({ key, title, total: arr.length })
    list2.push(...arr)
  })
  return list2
}

// 列表分页合并：第一页替换，其余页追加
export const mergeMailPage = (prev, data, page) => {
  const list = data?.list || []
  if (page === 1) {
    return { ...data, list }
  }
  return { ...data, list: [...(prev?.list || []), ...list] }
}

// 判断邮件标记
export const hasFlag = (item, flag) => Array.isArray(item?.flags) && item.flags.includes(flag)

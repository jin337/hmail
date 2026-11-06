import dayjs from 'dayjs'
import { describe, expect, it } from 'vitest'

import {
  buildQuotedContent,
  createMailFormData,
  getImageIds,
  groupMailByTime,
  mergeMailPage,
  normalizeDetailContent,
} from './mail'

describe('getImageIds', () => {
  it('提取 data-href 中的内联图片 id', () => {
    const html = `
      <p>正文</p>
      <img data-href="http://localhost/api/viewfile?url=static/images/a@b.com/12/image_3.png" />
      <img data-href="http://localhost/api/viewfile?url=static/images/a@b.com/12/image_7.JPG" />
      <img src="http://localhost/api/viewfile?url=static/images/a@b.com/12/image_9.gif" />
    `
    expect(getImageIds(html)).toEqual(['3', '7'])
  })

  it('空内容返回空数组', () => {
    expect(getImageIds('')).toEqual([])
    expect(getImageIds(null)).toEqual([])
  })
})

describe('normalizeDetailContent', () => {
  it('互换 src 与 data-href，并把尖括号邮箱替换为 mailto 链接', () => {
    const html =
      '<div><img src="static/a.png" data-href="http://x/a.png">' +
      '<span style="color: rgb(149, 157, 166);">&lt;test@jin337.top&gt;</span></div>'

    const result = normalizeDetailContent(html)

    expect(result).toContain('src="http://x/a.png"')
    expect(result).toContain('data-href="static/a.png"')
    expect(result).toContain('href="mailto:test@jin337.top"')
    expect(result).toContain('data-mail-email="test@jin337.top"')
    expect(result).not.toContain('&lt;test@jin337.top&gt;')
  })
})

describe('groupMailByTime', () => {
  const today = dayjs()
  const list = [
    { uid: '1', send_time: today.format('YYYY-MM-DD HH:mm:ss') },
    { uid: '2', send_time: today.subtract(1, 'day').format('YYYY-MM-DD HH:mm:ss') },
    { uid: '3', send_time: today.subtract(3, 'year').format('YYYY-MM-DD HH:mm:ss') },
  ]

  it('按今天/昨日/更早分组', () => {
    const result = groupMailByTime(list, ['all', 'date_desc'])
    const groups = result.filter((item) => item.key)

    expect(groups.map((item) => item.title)).toEqual(['今天', '昨日', '更早'])
    expect(groups.map((item) => item.total)).toEqual([1, 1, 1])
    expect(result.filter((item) => item.uid).map((item) => item.uid)).toEqual(['1', '2', '3'])
  })

  it('由旧到新时分组顺序反转', () => {
    const result = groupMailByTime(list, ['all', 'date_asc'])
    const groups = result.filter((item) => item.key)
    expect(groups.map((item) => item.title)).toEqual(['更早', '昨日', '今天'])
  })

  it('空列表返回空数组', () => {
    expect(groupMailByTime([], ['all'])).toEqual([])
  })
})

describe('mergeMailPage', () => {
  const prev = { total: 3, list: [{ uid: 1 }, { uid: 2 }] }

  it('第一页替换原列表', () => {
    expect(mergeMailPage(prev, { total: 1, list: [{ uid: 9 }] }, 1)).toEqual({ total: 1, list: [{ uid: 9 }] })
  })

  it('后续页追加到已有列表', () => {
    expect(mergeMailPage(prev, { total: 3, list: [{ uid: 3 }] }, 2)).toEqual({
      total: 3,
      list: [{ uid: 1 }, { uid: 2 }, { uid: 3 }],
    })
  })

  it('首次加载无旧数据时也能合并', () => {
    expect(mergeMailPage(undefined, { total: 1, list: [{ uid: 1 }] }, 2)).toEqual({ total: 1, list: [{ uid: 1 }] })
  })
})

describe('buildQuotedContent', () => {
  const mail = {
    from: 'sender@jin337.top',
    from_info: { name: '发件人' },
    to_info: [{ name: '收件人', email: 'to@jin337.top' }],
    cc_info: [{ name: '抄送人', email: 'cc@jin337.top' }],
    cc: true,
    subject: '测试主题',
    date: '2026-01-01 10:00:00',
    detail: { content: '<p>原始正文</p>' },
  }

  it('包含原始邮件信息与正文', () => {
    const html = buildQuotedContent(mail)
    expect(html).toContain('原始邮件')
    expect(html).toContain('发件人')
    expect(html).toContain('sender@jin337.top')
    expect(html).toContain('收件人')
    expect(html).toContain('抄送人')
    expect(html).toContain('测试主题')
    expect(html).toContain('<p>原始正文</p>')
    expect(html).toContain('2026年01月01日 10:00:00')
  })

  it('无抄送时不输出抄送行', () => {
    const html = buildQuotedContent({ ...mail, cc: false })
    expect(html).not.toContain('抄送人')
  })
})

describe('createMailFormData', () => {
  const detail = {
    content: '<p>正文<img src="http://x/image_5.png" data-href="cid:image_5"></p>',
    attachments: [
      { part_id: '2', file_name: 'a.pdf' },
      { originFile: new File(['hello'], 'b.txt'), file_name: 'b.txt' },
    ],
  }

  it('组装收件人、主题、正文、附件与图片 id', () => {
    const formData = createMailFormData({
      rest: {
        to_info: [{ value: 'a@jin337.top' }],
        cc_info: [{ value: 'b@jin337.top' }],
        subject: '主题',
      },
      detail,
      type: 'Sent',
    })

    expect(formData.get('to')).toBe('a@jin337.top')
    expect(formData.get('cc')).toBe('b@jin337.top')
    expect(formData.get('subject')).toBe('主题')
    expect(formData.get('folder')).toBe('Sent')
    expect(formData.get('content')).toContain('src="cid:image_5"')
    expect(formData.get('part_ids')).toBe('2,5')
    expect(formData.getAll('files')).toHaveLength(1)
    expect(formData.get('uid')).toBeNull()
  })

  it('编辑草稿时 folder 固定为 Drafts 并带上 uid', () => {
    const formData = createMailFormData({
      rest: {
        uid: 9,
        to_info: [{ value: 'a@jin337.top' }],
        subject: '主题',
      },
      detail: { content: '正文' },
      type: 'Sent',
    })
    expect(formData.get('folder')).toBe('Drafts')
    expect(formData.get('uid')).toBe('9')
  })

  it('定时发送与回复头信息', () => {
    const formData = createMailFormData({
      rest: {
        to_info: [{ value: 'a@jin337.top' }],
        subject: 'Re: 主题',
        is_reply: true,
        message_id: '<mid@x>',
        references: '<ref@x>',
      },
      detail: { content: '正文' },
      type: 'Sent',
      customTime: '2026-02-01 09:00:00',
    })

    expect(formData.get('x-schedule-send')).toBe('2026-02-01 09:00:00')
    expect(formData.get('in_reply_to')).toBe('<mid@x>')
    expect(formData.get('references')).toBe('<ref@x> <mid@x>')
  })
})

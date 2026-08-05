import { useEffect, useRef } from 'react'

import './index.scss'
import ToolBar from './ToolBar'

// 图标导入
import align from './icons/align.svg'
import bgcolor from './icons/bgcolor.svg'
import bold from './icons/bold.svg'
import clear from './icons/clear.svg'
import color from './icons/color.svg'
import hr from './icons/hr.svg'
import indent_minus from './icons/indent_minus.svg'
import indent_plus from './icons/indent_plus.svg'
import italic from './icons/italic.svg'
import line_height from './icons/line_height.svg'
import order_number from './icons/order_number.svg'
import order_object from './icons/order_object.svg'
import redo from './icons/redo.svg'
import strike from './icons/strike.svg'
import underline from './icons/underline.svg'
import undo from './icons/undo.svg'

const toolBar = [
  { title: '清除格式', type: 'clear', icon: clear },
  { title: '撤销', type: 'undo', icon: undo },
  { title: '重做', type: 'redo', icon: redo },
  { title: null, type: 'divider', icon: null },
  {
    title: '默认字体',
    type: 'font',
    icon: null,
    children: [
      { title: '默认字体', type: 'font-default', icon: null },
      { title: '黑体', type: 'font-hei-ti', icon: null },
      { title: '仿宋', type: 'font-song-ti', icon: null },
      { title: '楷体', type: 'font-kai-ti', icon: null },
      { title: '标楷体', type: 'font-biao-kai-ti', icon: null },
      { title: '华文仿宋', type: 'font-hua-wen-song-ti', icon: null },
      { title: '华文楷体', type: 'font-hua-wen-kai-ti', icon: null },
      { title: '宋体', type: 'font-song-ti', icon: null },
      { title: '微软雅黑', type: 'font-microsoft-ya-hei', icon: null },
      { title: 'Arial', type: 'font-arial', icon: null },
      { title: 'Tahoma', type: 'font-tahoma', icon: null },
      { title: 'Verdana', type: 'font-verdana', icon: null },
      { title: 'Times New Roman', type: 'font-times-new-roman', icon: null },
    ],
  },
  {
    title: '字号',
    type: 'size',
    icon: null,
    children: [
      { title: '12', type: 'text-12', icon: null },
      { title: '13', type: 'text-13', icon: null },
      { title: '14', type: 'text-14', icon: null },
      { title: '15', type: 'text-15', icon: null },
      { title: '16', type: 'text-16', icon: null },
      { title: '19', type: 'text-19', icon: null },
      { title: '22', type: 'text-22', icon: null },
      { title: '24', type: 'text-24', icon: null },
      { title: '29', type: 'text-29', icon: null },
      { title: '32', type: 'text-32', icon: null },
      { title: '40', type: 'text-40', icon: null },
      { title: '48', type: 'text-48', icon: null },
    ],
  },
  {
    title: '行间距',
    type: 'line-height',
    icon: line_height,
    children: [
      { title: '1.0', type: 'leading-1.0', icon: null },
      { title: '1.15', type: 'leading-1.15', icon: null },
      { title: '1.3', type: 'leading-1.3', icon: null },
      { title: '1.5', type: 'leading-1.5', icon: null },
      { title: '2.0', type: 'leading-2.0', icon: null },
      { title: '3.0', type: 'leading-3.0', icon: null },
    ],
  },
  { title: null, type: 'divider', icon: null },
  { title: '加粗', type: 'bold', icon: bold },
  { title: '斜体', type: 'italic', icon: italic },
  { title: '下划线', type: 'underline', icon: underline },
  { title: '删除线', type: 'strike', icon: strike },
  {
    title: '字体颜色',
    type: 'color',
    icon: color,
    children: [
      {
        title: '颜色选择',
        type: 'color-picker',
        color: 'rgb(0, 0, 0)',
      },
    ],
  },
  {
    title: '背景颜色',
    type: 'bgcolor',
    icon: bgcolor,
    children: [
      {
        title: '颜色选择',
        type: 'color-picker',
        color: 'rgb(255, 255, 255)',
      },
    ],
  },
  { title: null, type: 'divider', icon: null },
  { title: '项目编号', type: 'object-list', icon: order_object },
  { title: '数字编号', type: 'number-list', icon: order_number },
  { title: '添加缩进', type: 'indent-plus', icon: indent_plus },
  { title: '减少缩进', type: 'indent-minus', icon: indent_minus },
  {
    title: '对齐',
    type: 'align',
    icon: align,
    children: [
      { title: '左对齐', type: 'left', icon: null },
      { title: '居中对齐', type: 'center', icon: null },
      { title: '右对齐', type: 'right', icon: null },
      { title: '两端对齐', type: 'justify', icon: null },
    ],
  },
  { title: null, type: 'divider', icon: null },
  { title: '分割线', type: 'hr', icon: hr },
]

const Edit = (props) => {
  const { value: initialValue, onChange, height = 300 } = props
  const editorRef = useRef(null)

  const undoStack = useRef([]) // 撤销栈
  const redoStack = useRef([]) // 重做栈
  const isUndoing = useRef(false) // 是否正在撤销
  const debounceTimer = useRef(null) //防抖的定时器引用

  // 设置样式
  const setStyle = (styleObj) => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const styleStr = Object.entries(styleObj)
      .map(([k, v]) => {
        const prop = k.replace(/([A-Z])/g, '-$1').toLowerCase()
        return `${prop}: ${v}`
      })
      .join('; ')

    if (!styleStr) return

    const blockTags = new Set(['DIV', 'P', 'UL', 'OL', 'LI'])

    // ---- 工具函数 ----

    // 计算某个 DOM 位置在编辑器纯文本中的字符偏移量
    const getOffset = (container, offset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node === container) {
          if (container.nodeType === Node.TEXT_NODE) return count + offset
          let idx = 0
          let child = container.firstChild
          while (child && idx < offset) {
            if (child.nodeType === Node.TEXT_NODE) count += child.textContent.length
            else if (child.tagName === 'BR') count += 1
            idx++
            child = child.nextSibling
          }
          return count
        }
        if (node.nodeType === Node.TEXT_NODE) count += node.textContent.length
        else if (node.tagName === 'BR') count += 1
        node = walker.nextNode()
      }
      return count
    }

    // 根据字符偏移量找到对应的 DOM 位置和文本节点
    const resolveOffset = (targetOffset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node.nodeType === Node.TEXT_NODE) {
          const len = node.textContent.length
          if (count + len >= targetOffset) return { node, offset: targetOffset - count }
          count += len
        } else if (node.tagName === 'BR') {
          if (count === targetOffset)
            return { node: node.parentNode, offset: Array.from(node.parentNode.childNodes).indexOf(node) }
          count += 1
        }
        node = walker.nextNode()
      }
      return { node: editor, offset: editor.childNodes.length }
    }

    // ---- Step 1: 规范化前，保存选区的字符偏移量 ----
    const range = sel.getRangeAt(0)
    const savedStart = getOffset(range.startContainer, range.startOffset)
    const savedEnd = getOffset(range.endContainer, range.endOffset)

    // ---- Step 2: 规范化 DOM —— 将顶层裸文本/内联节点包裹进 div ----
    {
      const children = Array.from(editor.childNodes)
      let group = []
      const flush = () => {
        if (group.length === 0) return
        const div = document.createElement('div')
        editor.insertBefore(div, group[0])
        group.forEach((n) => div.appendChild(n))
        group = []
      }
      children.forEach((child) => {
        const isBlock = child.nodeType === Node.ELEMENT_NODE && blockTags.has(child.tagName)
        if (isBlock) {
          flush()
        } else {
          group.push(child)
        }
      })
      flush()
    }

    // ---- Step 3: 通过偏移量恢复选区 ----
    const startPos = resolveOffset(savedStart)
    const endPos = resolveOffset(savedEnd)
    const newRange = document.createRange()
    newRange.setStart(startPos.node, startPos.offset)
    newRange.setEnd(endPos.node, endPos.offset)
    sel.removeAllRanges()
    sel.addRange(newRange)

    const startContainer = newRange.startContainer
    const endContainer = newRange.endContainer

    // ---- Step 4: 同一文本节点内选区，精确拆分包裹 ----
    if (startContainer === endContainer && startContainer.nodeType === Node.TEXT_NODE) {
      const node = startContainer
      const startOffset = newRange.startOffset
      const endOffset = newRange.endOffset
      if (startOffset === endOffset) return

      const len = node.textContent.length
      if (endOffset < len) node.splitText(endOffset)
      if (startOffset > 0) node.splitText(startOffset)

      const middle = startOffset > 0 ? node.nextSibling : node
      const span = document.createElement('span')
      span.setAttribute('style', styleStr)
      middle.parentNode.insertBefore(span, middle)
      span.appendChild(middle)

      sel.removeAllRanges()
      return
    }

    // ---- Step 5: 跨节点选区，收集所有文本节点后逐个包裹 ----
    const textNodes = []
    const treeWalker = document.createTreeWalker(newRange.commonAncestorContainer, NodeFilter.SHOW_TEXT, (node) =>
      newRange.intersectsNode(node) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT
    )
    let current = treeWalker.nextNode()
    while (current) {
      if (current.textContent.trim().length > 0 || current === startContainer || current === endContainer) {
        textNodes.push(current)
      }
      current = treeWalker.nextNode()
    }
    if (textNodes.length === 0) return

    textNodes.forEach((node) => {
      if (node === endContainer && endContainer.nodeType === Node.TEXT_NODE) {
        const endOffset = newRange.endOffset
        if (endOffset > 0 && endOffset < node.textContent.length) {
          node.splitText(endOffset)
        }
      }

      if (node === startContainer && startContainer.nodeType === Node.TEXT_NODE) {
        const startOffset = newRange.startOffset
        if (startOffset > 0 && startOffset < node.textContent.length) {
          node.splitText(startOffset)
          node = node.nextSibling
        }
      }

      if (!node || node.textContent.length === 0) return

      const span = document.createElement('span')
      span.setAttribute('style', styleStr)
      node.parentNode.insertBefore(span, node)
      span.appendChild(node)
    })

    // 恢复选区
    const sp = resolveOffset(savedStart)
    const ep = resolveOffset(savedEnd)
    const r = document.createRange()
    r.setStart(sp.node, sp.offset)
    r.setEnd(ep.node, ep.offset)
    sel.removeAllRanges()
    sel.addRange(r)
  }

  // 设置列表
  const setList = (type) => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const blockTags = new Set(['DIV', 'P', 'UL', 'OL', 'LI'])

    // ---- 工具函数 ----

    const getOffset = (container, offset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node === container) {
          if (container.nodeType === Node.TEXT_NODE) return count + offset
          let idx = 0
          let child = container.firstChild
          while (child && idx < offset) {
            if (child.nodeType === Node.TEXT_NODE) count += child.textContent.length
            else if (child.tagName === 'BR') count += 1
            idx++
            child = child.nextSibling
          }
          return count
        }
        if (node.nodeType === Node.TEXT_NODE) count += node.textContent.length
        else if (node.tagName === 'BR') count += 1
        node = walker.nextNode()
      }
      return count
    }

    const resolveOffset = (targetOffset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node.nodeType === Node.TEXT_NODE) {
          const len = node.textContent.length
          if (count + len >= targetOffset) return { node, offset: targetOffset - count }
          count += len
        } else if (node.tagName === 'BR') {
          if (count === targetOffset)
            return { node: node.parentNode, offset: Array.from(node.parentNode.childNodes).indexOf(node) }
          count += 1
        }
        node = walker.nextNode()
      }
      return { node: editor, offset: editor.childNodes.length }
    }

    // ---- Step 1: 保存选区偏移量 ----
    const range = sel.getRangeAt(0)
    const savedStart = getOffset(range.startContainer, range.startOffset)
    const savedEnd = getOffset(range.endContainer, range.endOffset)

    // ---- Step 2: 规范化 DOM —— 顶层裸文本/内联节点包裹进 div ----
    {
      const children = Array.from(editor.childNodes)
      let group = []
      const flush = () => {
        if (group.length === 0) return
        const div = document.createElement('div')
        editor.insertBefore(div, group[0])
        group.forEach((n) => div.appendChild(n))
        group = []
      }
      children.forEach((child) => {
        const isBlock = child.nodeType === Node.ELEMENT_NODE && blockTags.has(child.tagName)
        if (isBlock) {
          flush()
        } else {
          group.push(child)
        }
      })
      flush()
    }

    // ---- Step 3: 恢复选区 ----
    const startPos = resolveOffset(savedStart)
    const endPos = resolveOffset(savedEnd)
    const newRange = document.createRange()
    newRange.setStart(startPos.node, startPos.offset)
    newRange.setEnd(endPos.node, endPos.offset)
    sel.removeAllRanges()
    sel.addRange(newRange)

    // ---- Step 4: 收集选区命中的编辑器顶层块元素 ----
    const selectedBlocks = []
    Array.from(editor.childNodes).forEach((child) => {
      if (child.nodeType === Node.ELEMENT_NODE && newRange.intersectsNode(child)) {
        selectedBlocks.push(child)
      }
    })

    if (selectedBlocks.length === 0) return

    // ---- Step 5: 创建列表，将每个块的内容移入 <li> ----
    const list = document.createElement(type)
    list.style.marginLeft = '20px'
    if (type === 'UL') {
      list.style.listStyleType = 'disc'
    } else if (type === 'OL') {
      list.style.listStyleType = 'decimal'
    }
    editor.insertBefore(list, selectedBlocks[0])

    selectedBlocks.forEach((block) => {
      const li = document.createElement('li')
      while (block.firstChild) {
        li.appendChild(block.firstChild)
      }
      list.appendChild(li)
      block.remove()
    })

    // ---- Step 6: 恢复选区 ----
    const sp = resolveOffset(savedStart)
    const ep = resolveOffset(savedEnd)
    const r = document.createRange()
    r.setStart(sp.node, sp.offset)
    r.setEnd(ep.node, ep.offset)
    sel.removeAllRanges()
    sel.addRange(r)
  }

  // 分割线
  const setHr = () => {
    const editor = editorRef.current
    if (!editor) return
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount) return

    const line = document.createElement('div')
    line.style.margin = '16px 0px'
    line.style.borderTop = '1px solid rgb(230, 232, 235)'

    if (sel.isCollapsed) {
      const range = sel.getRangeAt(0)
      if (editor.contains(range.startContainer)) {
        range.insertNode(line)
      } else {
        editor.appendChild(line)
      }
    } else {
      const range = sel.getRangeAt(0)
      range.deleteContents()
      range.insertNode(line)
    }
  }

  // 清除格式
  const clearFormat = () => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const blockTags = new Set(['DIV', 'P', 'UL', 'OL', 'LI'])

    // ---- 工具函数 ----

    const getOffset = (container, offset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node === container) {
          if (container.nodeType === Node.TEXT_NODE) return count + offset
          let idx = 0
          let child = container.firstChild
          while (child && idx < offset) {
            if (child.nodeType === Node.TEXT_NODE) count += child.textContent.length
            else if (child.tagName === 'BR') count += 1
            idx++
            child = child.nextSibling
          }
          return count
        }
        if (node.nodeType === Node.TEXT_NODE) count += node.textContent.length
        else if (node.tagName === 'BR') count += 1
        node = walker.nextNode()
      }
      return count
    }

    const resolveOffset = (targetOffset) => {
      let count = 0
      const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
      let node = walker.nextNode()
      while (node) {
        if (node.nodeType === Node.TEXT_NODE) {
          const len = node.textContent.length
          if (count + len >= targetOffset) return { node, offset: targetOffset - count }
          count += len
        } else if (node.tagName === 'BR') {
          if (count === targetOffset)
            return { node: node.parentNode, offset: Array.from(node.parentNode.childNodes).indexOf(node) }
          count += 1
        }
        node = walker.nextNode()
      }
      return { node: editor, offset: editor.childNodes.length }
    }

    // ---- Step 1: 保存选区偏移量 ----
    const range = sel.getRangeAt(0)
    const savedStart = getOffset(range.startContainer, range.startOffset)
    const savedEnd = getOffset(range.endContainer, range.endOffset)

    // ---- Step 2: 规范化 DOM ----
    {
      const children = Array.from(editor.childNodes)
      let group = []
      const flush = () => {
        if (group.length === 0) return
        const div = document.createElement('div')
        editor.insertBefore(div, group[0])
        group.forEach((n) => div.appendChild(n))
        group = []
      }
      children.forEach((child) => {
        const isBlock = child.nodeType === Node.ELEMENT_NODE && blockTags.has(child.tagName)
        if (isBlock) {
          flush()
        } else {
          group.push(child)
        }
      })
      flush()
    }

    // ---- Step 3: 恢复选区 ----
    const startPos = resolveOffset(savedStart)
    const endPos = resolveOffset(savedEnd)
    const newRange = document.createRange()
    newRange.setStart(startPos.node, startPos.offset)
    newRange.setEnd(endPos.node, endPos.offset)
    sel.removeAllRanges()
    sel.addRange(newRange)

    // ---- 辅助：拆分 span 元素 ----
    // 将 span 在内部文本偏移 pos 处一分为二，返回后半部分
    const splitSpanAt = (span, pos) => {
      const before = span.cloneNode(false)
      const after = span.cloneNode(false)
      let offset = 0
      let splitDone = false
      Array.from(span.childNodes).forEach((child) => {
        if (splitDone) {
          after.appendChild(child)
        } else if (child.nodeType === Node.TEXT_NODE) {
          const len = child.textContent.length
          if (offset + len <= pos) {
            before.appendChild(child)
            offset += len
          } else {
            const localPos = pos - offset
            if (localPos > 0) {
              child.splitText(localPos)
              before.appendChild(child)
            }
            after.appendChild(child.nextSibling)
            splitDone = true
          }
        } else {
          before.appendChild(child)
        }
      })
      span.parentNode.insertBefore(after, span)
      span.parentNode.insertBefore(before, span)
      span.remove()
      return after
    }

    // ---- 辅助：解包 span（移除 span 保留子节点）----
    const unwrapSpan = (span) => {
      const parent = span.parentNode
      while (span.firstChild) parent.insertBefore(span.firstChild, span)
      span.remove()
    }

    // ---- 辅助：将 <li> 内容转为 <div> ----
    const liToDiv = (li) => {
      const div = document.createElement('div')
      while (li.firstChild) div.appendChild(li.firstChild)
      return div
    }

    // ---- Step 4: 收集选区命中的顶层块 ----
    const selectedBlocks = []
    Array.from(editor.childNodes).forEach((child) => {
      if (child.nodeType === Node.ELEMENT_NODE && newRange.intersectsNode(child)) {
        selectedBlocks.push(child)
      }
    })

    if (selectedBlocks.length === 0) return

    // ---- Step 5: 处理每个块 ----
    selectedBlocks.forEach((block) => {
      const isList = block.tagName === 'UL' || block.tagName === 'OL'

      if (isList) {
        const items = Array.from(block.querySelectorAll(':scope > li'))
        const selectedIndices = []

        items.forEach((li, index) => {
          if (!newRange.intersectsNode(li)) return
          selectedIndices.push(index)
          const isFull =
            savedStart === 0 ||
            (li === items[0] && newRange.startContainer === li && newRange.startOffset === 0) ||
            li.contains(newRange.startContainer)

          // 5a: 拆分边界处的 span，解包选区内的 span
          const treeWalker = document.createTreeWalker(li, NodeFilter.SHOW_TEXT)
          let textNode = treeWalker.nextNode()
          while (textNode) {
            const next = treeWalker.nextNode()
            const span = textNode.parentNode
            if (span !== li && span.tagName === 'SPAN' && span.hasAttribute('style')) {
              // 计算 span 第一个文本节点在编辑器中的偏移
              let spanTextOffset = 0
              const tw = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
              let n = tw.nextNode()
              while (n && n !== span.firstChild) {
                if (n.nodeType === Node.TEXT_NODE) spanTextOffset += n.textContent.length
                else if (n.tagName === 'BR') spanTextOffset += 1
                n = tw.nextNode()
              }
              const spanStart = spanTextOffset
              const spanEnd = spanStart + span.textContent.length

              // 拆分跨越选区边界的 span
              if (savedStart > spanStart && savedStart < spanEnd) {
                splitSpanAt(span, savedStart - spanStart)
              }
              if (savedEnd > spanStart && savedEnd < spanEnd) {
                // 找到包含 savedEnd 的 span（可能是拆分后的后半部分）
                const parentSpan = textNode.parentNode
                if (parentSpan.tagName === 'SPAN' && parentSpan.hasAttribute('style')) {
                  let parentStart = 0
                  const tw2 = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
                  let n2 = tw2.nextNode()
                  while (n2 && n2 !== parentSpan.firstChild) {
                    if (n2.nodeType === Node.TEXT_NODE) parentStart += n2.textContent.length
                    else if (n2.tagName === 'BR') parentStart += 1
                    n2 = tw2.nextNode()
                  }
                  if (savedEnd > parentStart && savedEnd < parentStart + parentSpan.textContent.length) {
                    splitSpanAt(parentSpan, savedEnd - parentStart)
                  }
                }
              }
            }
            textNode = next
          }

          // 重新遍历，解包选区内的 styled span
          const walker2 = document.createTreeWalker(li, NodeFilter.SHOW_TEXT)
          let tn = walker2.nextNode()
          while (tn) {
            const next2 = walker2.nextNode()
            const parent = tn.parentNode
            if (parent !== li && parent.tagName === 'SPAN' && parent.hasAttribute('style')) {
              unwrapSpan(parent)
            }
            tn = next2
          }
        })

        // 5b: 将选中的 <li> 转为 <div>，保留未选中的在列表中
        if (selectedIndices.length > 0) {
          const firstIdx = selectedIndices[0]
          const lastIdx = selectedIndices[selectedIndices.length - 1]
          const allSelected = selectedIndices.length === items.length
          const isStart = firstIdx === 0
          const isEnd = lastIdx === items.length - 1

          const selectedLis = selectedIndices.map((i) => items[i])
          const divs = selectedLis.map((li) => liToDiv(li))

          if (allSelected) {
            // 全部选中：列表整体替换为 div
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            block.remove()
          } else if (isStart) {
            // 开头连续选中：div 插入列表前，从列表中移除选中项
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            selectedLis.forEach((li) => li.remove())
          } else if (isEnd) {
            // 末尾连续选中：div 插入列表后，从列表中移除选中项
            selectedLis.forEach((li) => li.remove())
            divs.forEach((div) => block.parentNode.insertBefore(div, block.nextSibling))
          } else {
            // 中间选中：拆分列表为前后两段，div 放中间
            const beforeList = document.createElement(block.tagName)
            for (let i = 0; i < firstIdx; i++) {
              beforeList.appendChild(items[i].cloneNode(true))
            }
            const afterList = document.createElement(block.tagName)
            for (let i = lastIdx + 1; i < items.length; i++) {
              afterList.appendChild(items[i].cloneNode(true))
            }
            block.parentNode.insertBefore(beforeList, block)
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            block.parentNode.insertBefore(afterList, block)
            block.remove()
          }
        }
      } else {
        // 非列表块（DIV / P 等）
        const isFull =
          savedStart === 0 ||
          (newRange.startContainer === block && newRange.startOffset === 0) ||
          (block.contains(newRange.startContainer) && block.contains(newRange.endContainer))

        // 5a: 拆分边界 span + 解包选区内 span
        const treeWalker = document.createTreeWalker(block, NodeFilter.SHOW_TEXT)
        let textNode = treeWalker.nextNode()
        while (textNode) {
          const next = treeWalker.nextNode()
          const span = textNode.parentNode
          if (span !== block && span.tagName === 'SPAN' && span.hasAttribute('style')) {
            let spanTextOffset = 0
            const tw = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
            let n = tw.nextNode()
            while (n && n !== span.firstChild) {
              if (n.nodeType === Node.TEXT_NODE) spanTextOffset += n.textContent.length
              else if (n.tagName === 'BR') spanTextOffset += 1
              n = tw.nextNode()
            }
            const spanStart = spanTextOffset
            const spanEnd = spanStart + span.textContent.length

            if (savedStart > spanStart && savedStart < spanEnd) {
              splitSpanAt(span, savedStart - spanStart)
            }
            if (savedEnd > spanStart && savedEnd < spanEnd) {
              const parentSpan = textNode.parentNode
              if (parentSpan.tagName === 'SPAN' && parentSpan.hasAttribute('style')) {
                let parentStart = 0
                const tw2 = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
                let n2 = tw2.nextNode()
                while (n2 && n2 !== parentSpan.firstChild) {
                  if (n2.nodeType === Node.TEXT_NODE) parentStart += n2.textContent.length
                  else if (n2.tagName === 'BR') parentStart += 1
                  n2 = tw2.nextNode()
                }
                if (savedEnd > parentStart && savedEnd < parentStart + parentSpan.textContent.length) {
                  splitSpanAt(parentSpan, savedEnd - parentStart)
                }
              }
            }
          }
          textNode = next
        }

        // 重新遍历，解包选区内的 styled span
        const walker2 = document.createTreeWalker(block, NodeFilter.SHOW_TEXT)
        let tn = walker2.nextNode()
        while (tn) {
          const next2 = walker2.nextNode()
          const parent = tn.parentNode
          if (parent.tagName === 'SPAN' && parent.hasAttribute('style')) {
            unwrapSpan(parent)
          }
          tn = next2
        }
      }
    })

    // ---- Step 6: 恢复选区 ----
    const sp = resolveOffset(savedStart)
    const ep = resolveOffset(savedEnd)
    const r = document.createRange()
    r.setStart(sp.node, sp.offset)
    r.setEnd(ep.node, ep.offset)
    sel.removeAllRanges()
    sel.addRange(r)
  }

  // 记录内容变化
  const recordChange = (html) => {
    if (isUndoing.current) return

    if (undoStack.current[undoStack.current.length - 1] === html) return

    undoStack.current.push(html)
    redoStack.current = []

    // 保留最近 50 步
    if (undoStack.current.length > 50) {
      undoStack.current.shift()
    }
  }

  // 撤销
  const handleUndo = () => {
    if (undoStack.current.length <= 1) return
    isUndoing.current = true
    const currentState = undoStack.current.pop()
    redoStack.current.push(currentState)

    const prevHtml = undoStack.current[undoStack.current.length - 1]
    editorRef.current.innerHTML = prevHtml
    onChange?.(prevHtml)

    requestAnimationFrame(() => {
      isUndoing.current = false
    })
  }

  // 重做
  const handleRedo = () => {
    if (redoStack.current.length === 0) return

    isUndoing.current = true
    const nextHtml = redoStack.current.pop()
    undoStack.current.push(nextHtml)

    editorRef.current.innerHTML = nextHtml
    onChange?.(nextHtml)

    requestAnimationFrame(() => {
      isUndoing.current = false
    })
  }

  // 命令入口
  const onCommand = (option) => {
    const { type, color } = option
    editorRef.current?.focus()

    switch (type) {
      // 撤销/重做
      case 'undo':
        handleUndo()
        break
      case 'redo':
        handleRedo()
        break

      case 'clear':
        clearFormat()
        break

      case 'bold':
        setStyle({ fontWeight: 'bold' })
        break
      case 'italic':
        setStyle({ fontStyle: 'italic' })
        break
      case 'underline':
        setStyle({ textDecoration: 'underline' })
        break
      case 'strike':
        setStyle({ textDecoration: 'line-through' })
        break

      // 字体颜色 / 背景色
      case 'color':
        setStyle({ color: color })
        break
      case 'bgcolor':
        setStyle({ backgroundColor: color })
        break

      // 字体
      case 'font-default':
        setStyle({ fontFamily: '' })
        break
      case 'font-hei-ti':
        setStyle({ fontFamily: 'SimHei' })
        break
      case 'font-song-ti':
        setStyle({ fontFamily: 'SimSun' })
        break
      case 'font-kai-ti':
        setStyle({ fontFamily: 'KaiTi, STKaiti' })
        break
      case 'font-biao-kai-ti':
        setStyle({ fontFamily: 'BiauKai, STBiauKai' })
        break
      case 'font-hua-wen-song-ti':
        setStyle({ fontFamily: 'STFangsong, FangSong' })
        break
      case 'font-hua-wen-kai-ti':
        setStyle({ fontFamily: 'STKaiti, KaiTi' })
        break
      case 'font-microsoft-ya-hei':
        setStyle({ fontFamily: '"Microsoft YaHei"' })
        break
      case 'font-arial':
        setStyle({ fontFamily: 'Arial' })
        break
      case 'font-tahoma':
        setStyle({ fontFamily: 'Tahoma' })
        break
      case 'font-verdana':
        setStyle({ fontFamily: 'Verdana' })
        break
      case 'font-times-new-roman':
        setStyle({ fontFamily: '"Times New Roman"' })
        break

      // 字号
      case 'text-12':
        setStyle({ fontSize: '12px' })
        break
      case 'text-14':
        setStyle({ fontSize: '14px' })
        break
      case 'text-16':
        setStyle({ fontSize: '16px' })
        break

      // 行高
      case 'leading-1.0':
        setStyle({ lineHeight: '1.0' })
        break
      case 'leading-1.15':
        setStyle({ lineHeight: '1.15' })
        break
      case 'leading-1.3':
        setStyle({ lineHeight: '1.3' })
        break
      case 'leading-1.5':
        setStyle({ lineHeight: '1.5' })
        break
      case 'leading-2.0':
        setStyle({ lineHeight: '2.0' })
        break
      case 'leading-3.0':
        setStyle({ lineHeight: '3.0' })
        break

      // 对齐
      case 'left':
        setStyle({ textAlign: 'left' })
        break
      case 'center':
        setStyle({ textAlign: 'center' })
        break
      case 'right':
        setStyle({ textAlign: 'right' })
        break
      case 'justify':
        setStyle({ textAlign: 'justify' })
        break

      // 缩进
      case 'indent-plus':
        setStyle({ marginLeft: '28px' })
        break
      case 'indent-minus':
        setStyle({ marginLeft: '0' })
        break

      // 列表
      case 'object-list':
        setList('UL')
        break
      case 'number-list':
        setList('OL')
        break

      case 'hr':
        setHr()
        break
    }

    setTimeout(() => {
      onChange?.(editorRef.current.innerHTML)
    }, 0)
  }

  // 输入事件
  const handleInput = () => {
    const editor = editorRef.current
    if (!editor) return

    const html = editor.innerHTML
    onChange?.(html)
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current)
    }
    debounceTimer.current = setTimeout(() => {
      recordChange(html)
    }, 500)
  }

  // 数据回填
  const isInitialMount = useRef(true)
  useEffect(() => {
    const dom = editorRef.current
    if (!dom) return
    const initialContent = initialValue || ''
    if (isInitialMount.current) {
      dom.innerHTML = initialContent
      undoStack.current = [initialContent]
      isInitialMount.current = false
      return
    }

    if (dom.innerHTML === '' && initialValue) {
      dom.innerHTML = initialValue
      undoStack.current = [initialValue]
      redoStack.current = []
    }
  }, [initialValue])

  return (
    <div className='y-mail-wrap'>
      <ToolBar items={toolBar} onCommand={onCommand} />

      <div
        style={{ height: `${height}px` }}
        ref={editorRef}
        className='y-mail-content'
        contentEditable='true'
        data-placeholder='请输入内容...'
        suppressContentEditableWarning={true}
        onInput={handleInput}
      />
    </div>
  )
}

export default Edit

import { useRef } from 'react'

const BLOCK_TAGS = ['DIV', 'P', 'UL', 'OL', 'LI']

export default function useEditor() {
  const editorRef = useRef(null)
  // 纯文本用div包裹
  const normalizeDOM = () => {
    const editor = editorRef.current
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
      const isBlock = child.nodeType === Node.ELEMENT_NODE && new Set(BLOCK_TAGS).has(child.tagName)
      if (isBlock) {
        flush()
      } else {
        group.push(child)
      }
    })
    flush()
  }
  //   判断当前选区是单行还是多行
  const getSelectionType = (range) => {
    const ancestor = range.commonAncestorContainer
    if (ancestor.nodeType === Node.ELEMENT_NODE) {
      return BLOCK_TAGS.includes(ancestor.tagName) ? 'multi' : 'single'
    }
    if (ancestor.nodeType === Node.TEXT_NODE) {
      return 'single'
    }
    return 'none'
  }

  // 向上查找最近的块级父元素
  const getBlockParent = (node) => {
    while (node && node.nodeType === Node.TEXT_NODE) node = node.parentNode
    while (node && !BLOCK_TAGS.includes(node.tagName)) {
      node = node.parentNode
    }
    return node
  }

  // 包裹Range
  const safeSurround = (targetRange, styleObj) => {
    let ancestor = targetRange.commonAncestorContainer

    // 找到最近的 SPAN
    if (ancestor.nodeType === Node.TEXT_NODE) ancestor = ancestor.parentElement
    let parentSpan = null
    let current = ancestor
    let depth = 0
    while (current && depth < 10) {
      // 最多10层
      if (current.tagName === 'SPAN') {
        parentSpan = current
        break
      }
      current = current.parentElement
      depth++
    }

    if (parentSpan && parentSpan.textContent === targetRange.toString()) {
      // 将新样式合并到现有样式中，自动覆盖同名属性
      const currentStyles = parentSpan.style
      Object.entries(styleObj).forEach(([key, value]) => {
        currentStyles[key] = value
      })
      return
    }

    const span = document.createElement('span')
    Object.assign(span.style, styleObj)

    try {
      targetRange.surroundContents(span)
    } catch (e) {
      console.warn('surroundContents 失败，使用 extract 兜底', e)
      const frag = targetRange.extractContents()
      span.appendChild(frag)
      targetRange.insertNode(span)
    }
  }

  // 设置样式
  const setStyle = (styleObj) => {
    const editor = editorRef.current
    if (!editor) return
    // 纯文本用div包裹
    normalizeDOM()

    const sel = window.getSelection()
    if (!sel || !sel.rangeCount) return null
    const range = sel.getRangeAt(0)
    if (!range) return

    const selectionType = getSelectionType(range) // single/multi

    // 单行
    if (selectionType === 'single') {
      safeSurround(range, styleObj)
    }

    // 多行
    if (selectionType === 'multi') {
      const startBlock = getBlockParent(range.startContainer)
      const endBlock = getBlockParent(range.endContainer)

      // 收集从 startBlock 到 endBlock 之间的所有块级节点
      const blockNodes = []
      if (startBlock && endBlock && startBlock.parentNode === endBlock.parentNode) {
        let current = startBlock
        while (current) {
          blockNodes.push(current)
          if (current === endBlock) break
          current = current.nextSibling
        }
      }

      // 遍历每一个块级节点，进行精准拆分与包裹
      blockNodes.forEach((block, index) => {
        const isFirst = index === 0
        const isLast = index === blockNodes.length - 1

        const innerRange = document.createRange()
        if (isFirst && isLast) {
          innerRange.selectNodeContents(block)
        } else if (isFirst) {
          innerRange.setStart(range.startContainer, range.startOffset)
          innerRange.setEnd(block, block.childNodes.length)
        } else if (isLast) {
          innerRange.setStart(block, 0)
          innerRange.setEnd(range.endContainer, range.endOffset)
        } else {
          innerRange.selectNodeContents(block)
        }

        if (!innerRange.collapsed) {
          safeSurround(innerRange, styleObj)
        }
      })
    }
  }

  // 设置列表
  const setList = (type) => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return
  }

  // 清除格式
  const clearFormat = () => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return
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

  return { editorRef, setStyle, setList, clearFormat, setHr }
}

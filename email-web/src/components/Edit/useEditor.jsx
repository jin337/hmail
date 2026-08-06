import { useRef } from 'react'

const BLOCK_TAGS = new Set(['DIV', 'P', 'UL', 'OL', 'LI'])

export default function useEditor() {
  const editorRef = useRef(null)

  // ---- 选区偏移量工具 ----
  const getOffset = (container, offset) => {
    const editor = editorRef.current
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
    const editor = editorRef.current
    let count = 0
    const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
    let node = walker.nextNode()
    while (node) {
      if (node.nodeType === Node.TEXT_NODE) {
        const len = node.textContent.length
        if (count + len >= targetOffset) return { node, offset: targetOffset - count }
        count += len
      } else if (node.tagName === 'BR') {
        if (count === targetOffset) return { node: node.parentNode, offset: Array.from(node.parentNode.childNodes).indexOf(node) }
        count += 1
      }
      node = walker.nextNode()
    }
    return { node: editor, offset: editor.childNodes.length }
  }

  // ---- DOM 规范化 ----
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
      const isBlock = child.nodeType === Node.ELEMENT_NODE && BLOCK_TAGS.has(child.tagName)
      if (isBlock) {
        flush()
      } else {
        group.push(child)
      }
    })
    flush()
  }

  // ---- 选区保存 / 恢复 ----
  const saveSelection = () => {
    const sel = window.getSelection()
    const range = sel.getRangeAt(0)
    return {
      start: getOffset(range.startContainer, range.startOffset),
      end: getOffset(range.endContainer, range.endOffset),
    }
  }

  const restoreSelection = (savedStart, savedEnd) => {
    const sel = window.getSelection()
    const startPos = resolveOffset(savedStart)
    const endPos = resolveOffset(savedEnd)
    const range = document.createRange()
    range.setStart(startPos.node, startPos.offset)
    range.setEnd(endPos.node, endPos.offset)
    sel.removeAllRanges()
    sel.addRange(range)
    return range
  }

  const normalizeAndRestore = () => {
    const saved = saveSelection()
    normalizeDOM()
    return { saved, range: restoreSelection(saved.start, saved.end) }
  }

  // ---- DOM 操作工具 ----
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

  const unwrapSpan = (span) => {
    const parent = span.parentNode
    while (span.firstChild) parent.insertBefore(span.firstChild, span)
    span.remove()
  }

  const getNodeEditorOffset = (targetNode) => {
    const editor = editorRef.current
    let count = 0
    const walker = document.createTreeWalker(editor, NodeFilter.SHOW_ALL)
    let node = walker.nextNode()
    while (node && node !== targetNode) {
      if (node.nodeType === Node.TEXT_NODE) count += node.textContent.length
      else if (node.tagName === 'BR') count += 1
      node = walker.nextNode()
    }
    return count
  }

  // ---- 收集选区命中的顶层块 ----
  const getSelectedBlocks = (range) => {
    const editor = editorRef.current
    const blocks = []
    Array.from(editor.childNodes).forEach((child) => {
      if (child.nodeType === Node.ELEMENT_NODE && range.intersectsNode(child)) {
        blocks.push(child)
      }
    })
    return blocks
  }

  // 设置样式
  const setStyle = (styleObj) => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const styleStr = Object.entries(styleObj)
      .map(([k, v]) => `${k.replace(/([A-Z])/g, '-$1').toLowerCase()}: ${v}`)
      .join('; ')
    if (!styleStr) return

    const { saved, range } = normalizeAndRestore()
    const startContainer = range.startContainer
    const endContainer = range.endContainer

    if (startContainer === endContainer && startContainer.nodeType === Node.TEXT_NODE) {
      const node = startContainer
      const startOffset = range.startOffset
      const endOffset = range.endOffset
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

    const textNodes = []
    const treeWalker = document.createTreeWalker(range.commonAncestorContainer, NodeFilter.SHOW_TEXT, (node) =>
      range.intersectsNode(node) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT
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
        const endOffset = range.endOffset
        if (endOffset > 0 && endOffset < node.textContent.length) node.splitText(endOffset)
      }
      if (node === startContainer && startContainer.nodeType === Node.TEXT_NODE) {
        const startOffset = range.startOffset
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

    restoreSelection(saved.start, saved.end)
  }

  // 设置列表
  const setList = (type) => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const { saved, range } = normalizeAndRestore()

    const selectedBlocks = getSelectedBlocks(range)
    if (selectedBlocks.length === 0) return

    const list = document.createElement(type)
    list.style.marginLeft = '20px'
    if (type === 'UL') list.style.listStyleType = 'disc'
    else if (type === 'OL') list.style.listStyleType = 'decimal'
    editor.insertBefore(list, selectedBlocks[0])

    selectedBlocks.forEach((block) => {
      const li = document.createElement('li')
      while (block.firstChild) li.appendChild(block.firstChild)
      list.appendChild(li)
      block.remove()
    })

    restoreSelection(saved.start, saved.end)
  }

  // 清除格式
  const clearFormat = () => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return
    const editor = editorRef.current
    if (!editor) return

    const { saved, range } = normalizeAndRestore()
    const selectedBlocks = getSelectedBlocks(range)
    if (selectedBlocks.length === 0) return

    const processSpanSplitting = (container) => {
      const treeWalker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT)
      let textNode = treeWalker.nextNode()
      while (textNode) {
        const next = treeWalker.nextNode()
        const span = textNode.parentNode
        if (span !== container && span.tagName === 'SPAN' && span.hasAttribute('style')) {
          const spanStart = getNodeEditorOffset(span.firstChild)
          const spanEnd = spanStart + span.textContent.length

          if (saved.start > spanStart && saved.start < spanEnd) {
            splitSpanAt(span, saved.start - spanStart)
          }
          if (saved.end > spanStart && saved.end < spanEnd) {
            const parentSpan = textNode.parentNode
            if (parentSpan.tagName === 'SPAN' && parentSpan.hasAttribute('style')) {
              const parentStart = getNodeEditorOffset(parentSpan.firstChild)
              const parentLen = parentSpan.textContent.length
              if (saved.end > parentStart && saved.end < parentStart + parentLen) {
                splitSpanAt(parentSpan, saved.end - parentStart)
              }
            }
          }
        }
        textNode = next
      }
    }

    const processUnwrap = (container, rootFilter) => {
      const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT)
      let tn = walker.nextNode()
      while (tn) {
        const next = walker.nextNode()
        const parent = tn.parentNode
        if (parent.tagName === 'SPAN' && parent.hasAttribute('style') && (!rootFilter || parent !== rootFilter)) {
          unwrapSpan(parent)
        }
        tn = next
      }
    }

    selectedBlocks.forEach((block) => {
      const isList = block.tagName === 'UL' || block.tagName === 'OL'

      if (isList) {
        const items = Array.from(block.querySelectorAll(':scope > li'))
        const selectedIndices = []

        items.forEach((li, index) => {
          if (!range.intersectsNode(li)) return
          selectedIndices.push(index)
          processSpanSplitting(li)
          processUnwrap(li, li)
        })

        if (selectedIndices.length > 0) {
          const firstIdx = selectedIndices[0]
          const lastIdx = selectedIndices[selectedIndices.length - 1]
          const allSelected = selectedIndices.length === items.length
          const isStart = firstIdx === 0
          const isEnd = lastIdx === items.length - 1
          const selectedLis = selectedIndices.map((i) => items[i])
          const divs = selectedLis.map((li) => {
            const div = document.createElement('div')
            while (li.firstChild) div.appendChild(li.firstChild)
            return div
          })

          if (allSelected) {
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            block.remove()
          } else if (isStart) {
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            selectedLis.forEach((li) => li.remove())
          } else if (isEnd) {
            selectedLis.forEach((li) => li.remove())
            divs.forEach((div) => block.parentNode.insertBefore(div, block.nextSibling))
          } else {
            const beforeList = document.createElement(block.tagName)
            for (let i = 0; i < firstIdx; i++) beforeList.appendChild(items[i].cloneNode(true))
            const afterList = document.createElement(block.tagName)
            for (let i = lastIdx + 1; i < items.length; i++) afterList.appendChild(items[i].cloneNode(true))
            block.parentNode.insertBefore(beforeList, block)
            divs.forEach((div) => block.parentNode.insertBefore(div, block))
            block.parentNode.insertBefore(afterList, block)
            block.remove()
          }
        }
      } else {
        processSpanSplitting(block)
        processUnwrap(block, null)
      }
    })

    restoreSelection(saved.start, saved.end)
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

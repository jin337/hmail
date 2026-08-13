import { useCallback, useEffect, useRef, useState } from 'react'

import './index.scss'

// 字体
const FONT_FAMILIES = [
  { label: '默认字体', value: 'inherit' },
  { label: '宋体', value: 'simsun' },
  { label: '黑体', value: 'simhei' },
  { label: '楷体', value: '楷体' },
  { label: '幼圆', value: '幼圆' },
  { label: 'Arial', value: 'Arial' },
  { label: 'Arial Black', value: 'Arial Black' },
  { label: 'Times New Roman', value: 'Times New Roman' },
  { label: 'Verdana', value: 'Verdana' },
]
// 字号
const FONT_SIZES = [
  { label: '9', value: '9px' },
  { label: '10', value: '10px' },
  { label: '11', value: '11px' },
  { label: '12', value: '12px' },
  { label: '14', value: '14px' },
  { label: '16', value: '16px' },
  { label: '18', value: '18px' },
  { label: '20', value: '20px' },
  { label: '22', value: '22px' },
  { label: '24', value: '24px' },
  { label: '26', value: '26px' },
  { label: '28', value: '28px' },
  { label: '36', value: '36px' },
  { label: '42', value: '42px' },
  { label: '48', value: '48px' },
  { label: '72', value: '72px' },
]
// 行间距
const LINE_HEIGHTS = [
  { label: '1.0', value: 1.0 },
  { label: '1.15', value: 1.15 },
  { label: '1.3', value: 1.3 },
  { label: '1.5', value: 1.5 },
  { label: '2.0', value: 2.0 },
  { label: '3.0', value: 3.0 },
]
// 对齐
const TEXT_ALIGN = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' },
  { label: '两端对齐', value: 'justify' },
]

// 默认样式
const DEFAULT_FORMAT = {
  fontFamily: 'inherit',
  fontSize: null,
  textColor: 'rgb(46, 48, 51)',
  backgroundColor: '',
  textAlign: 'left',
  lineHeight: 1.43,
  textIndent: '0em',
}

// 图标导入
import AlignIcon from './icons/align.svg'
import BgColorIcon from './icons/bgcolor.svg'
import BoldIcon from './icons/bold.svg'
import Checkcon from './icons/check.svg'
import ClearIcon from './icons/clear.svg'
import ColorIcon from './icons/color.svg'
import DownIcon from './icons/down.svg'
import HrIcon from './icons/hr.svg'
import IndentMinusIcon from './icons/indent_minus.svg'
import IndentPlusIcon from './icons/indent_plus.svg'
import ItalicIcon from './icons/italic.svg'
import LineHeightIcon from './icons/line_height.svg'
import OlIcon from './icons/order_number.svg'
import UlIcon from './icons/order_Object.svg'
import RedoIcon from './icons/redo.svg'
import StrikeIcon from './icons/strike.svg'
import UnderlineIcon from './icons/underline.svg'
import UndoIcon from './icons/undo.svg'

// 调色板
import ColorPicker from './ColorPicker'

// 防抖函数
const debounce = (func, wait) => {
  let timeout
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout)
      func(...args)
    }
    clearTimeout(timeout)
    timeout = setTimeout(later, wait)
  }
}

const RichTextEditor = ({ value = '', onChange }) => {
  const editorRef = useRef(null)
  const savedRange = useRef(null)
  const isInternalChange = useRef(false)
  const dropdownRef = useRef(null)

  const MAX_HISTORY = 50 // 最大历史记录数
  const undoStack = useRef([])
  const redoStack = useRef([])
  const isUndoingOrRedoing = useRef(false) // 防止撤销/重做时触发新的记录

  // 当前光标/选区的样式快照
  const [currentFormat, setCurrentFormat] = useState({
    bold: false,
    italic: false,
    underline: false,
    strikethrough: false,
    listType: null,
    isMediaSelected: false,
    ...DEFAULT_FORMAT,
  })

  // 工具栏配置项
  const toolBarItems = [
    { type: 'button', title: '清除格式', key: 'clear', icon: ClearIcon },
    { type: 'button', title: '撤销', key: 'undo', icon: UndoIcon },
    { type: 'button', title: '重做', key: 'redo', icon: RedoIcon },
    { type: 'divider' },
    { type: 'select', title: '字体', key: 'fontFamily', options: FONT_FAMILIES, defaultValue: 'inherit' },
    { type: 'select', title: '字号', key: 'fontSize', options: FONT_SIZES, defaultValue: null },
    { type: 'divider' },
    { type: 'toggle', title: '加粗', key: 'bold', icon: BoldIcon },
    { type: 'toggle', title: '斜体', key: 'italic', icon: ItalicIcon },
    { type: 'toggle', title: '下划线', key: 'underline', icon: UnderlineIcon },
    { type: 'toggle', title: '删除线', key: 'strike', icon: StrikeIcon },
    {
      type: 'color',
      title: '字体颜色',
      key: 'textColor',
      icon: ColorIcon,
      defaultValue: { default: 'rgb(46, 48, 51)', text: '默认颜色', color: 'rgb(247, 49, 22)' },
    },
    {
      type: 'color',
      title: '背景颜色',
      key: 'backgroundColor',
      icon: BgColorIcon,
      defaultValue: { default: '', text: '无颜色', color: 'rgb(255, 217, 0)' },
    },
    { type: 'divider' },
    { type: 'toggle', title: '无序列表', key: 'ul', icon: UlIcon },
    { type: 'toggle', title: '有序列表', key: 'ol', icon: OlIcon },
    { type: 'button', title: '增加缩进', key: 'plusIndent', icon: IndentPlusIcon },
    { type: 'button', title: '减少缩进', key: 'minusIndent', icon: IndentMinusIcon },
    { type: 'select', title: '对齐', key: 'textAlign', options: TEXT_ALIGN, defaultValue: null, icon: AlignIcon },
    { type: 'select', title: '行间距', key: 'lineHeight', options: LINE_HEIGHTS, defaultValue: 1.43, icon: LineHeightIcon },
    { type: 'divider' },
    { type: 'button', title: '插入分割线', key: 'hr', icon: HrIcon },
  ]

  // 添加占位符
  const addPlaceholderBlock = (rootEl) => {
    if (rootEl.textContent.trim() === '') {
      const block = document.createElement('div')
      block.innerHTML = '<br>'
      rootEl.innerHTML = ''
      rootEl.appendChild(block)
    }
  }

  // 状态提取核心逻辑
  const updateCurrentFormat = useCallback(() => {
    const selection = window.getSelection()
    if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
      return
    }

    const range = selection.getRangeAt(0)
    // 以默认样式为基准
    const newFormat = {
      bold: false,
      italic: false,
      underline: false,
      strikethrough: false,
      listType: null,
      isMediaSelected: false,
      ...DEFAULT_FORMAT,
    }

    // 隔离状态检测 (图片/分割线)
    const selectedNode =
      range.startContainer.nodeType === Node.ELEMENT_NODE
        ? range.startContainer.childNodes[range.startOffset]
        : range.startContainer.parentElement

    const isMedia =
      selectedNode?.nodeName === 'IMG' ||
      selectedNode?.nodeName === 'HR' ||
      selectedNode?.parentElement?.nodeName === 'IMG' ||
      selectedNode?.parentElement?.nodeName === 'HR'

    newFormat.isMediaSelected = !!isMedia
    if (newFormat.isMediaSelected) {
      setCurrentFormat(newFormat)
      return
    }

    // 块级样式提取 (向上寻找 div 或 li)
    let blockNode = range.startContainer
    if (blockNode.nodeType === Node.TEXT_NODE) blockNode = blockNode.parentElement
    while (blockNode && blockNode !== editorRef.current) {
      if (['DIV', 'LI'].includes(blockNode.nodeName)) break
      blockNode = blockNode.parentElement
    }

    // 块级样式提取
    if (blockNode && blockNode !== editorRef.current) {
      const style = blockNode.style
      newFormat.textAlign = style.textAlign || DEFAULT_FORMAT.textAlign
      newFormat.lineHeight = parseFloat(style.lineHeight) || DEFAULT_FORMAT.lineHeight
      newFormat.textIndent = style.textIndent || DEFAULT_FORMAT.textIndent
      newFormat.listType = blockNode.nodeName === 'LI' ? blockNode.parentElement?.nodeName.toLowerCase() : null
    }

    // 行内样式提取 (基于当前 span)
    let inlineNode = range.startContainer
    if (inlineNode.nodeType === Node.TEXT_NODE) inlineNode = inlineNode.parentElement
    if (inlineNode?.nodeName === 'SPAN') {
      const style = inlineNode.style
      newFormat.fontFamily = style.fontFamily?.replace(/['"]/g, '') || DEFAULT_FORMAT.fontFamily
      newFormat.fontSize = style.fontSize || DEFAULT_FORMAT.fontSize
      newFormat.textColor = style.color || DEFAULT_FORMAT.textColor
      newFormat.backgroundColor = style.backgroundColor || DEFAULT_FORMAT.backgroundColor
      newFormat.bold = style.fontWeight === 'bold' || parseInt(style.fontWeight) >= 700
      newFormat.italic = style.fontStyle === 'italic'
      newFormat.underline = style.textDecoration?.includes('underline')
      newFormat.strikethrough = style.textDecoration?.includes('line-through')
    }

    setCurrentFormat(newFormat)
  }, [currentFormat])

  // 获取当前光标/选区的精确位置
  const getCurrentSelection = () => {
    const selection = window.getSelection()
    if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) return null
    const range = selection.getRangeAt(0)
    return {
      startContainer: range.startContainer,
      startOffset: range.startOffset,
      endContainer: range.endContainer,
      endOffset: range.endOffset,
    }
  }

  // 恢复光标/选区位置
  const restoreSelection = (savedSelection) => {
    if (!savedSelection || !editorRef.current) return
    try {
      const selection = window.getSelection()
      const range = document.createRange()
      range.setStart(savedSelection.startContainer, savedSelection.startOffset)
      range.setEnd(savedSelection.endContainer, savedSelection.endOffset)
      selection.removeAllRanges()
      selection.addRange(range)
    } catch (e) {
      // 如果 DOM 结构变动导致节点找不到，降级处理：将光标移至末尾
      console.warn('恢复选区失败，已重置光标到末尾')
      const newRange = document.createRange()
      newRange.selectNodeContents(editorRef.current)
      newRange.collapse(false)
      const selection = window.getSelection()
      selection.removeAllRanges()
      selection.addRange(newRange)
    }
  }

  // 精准恢复选区，带异常降级
  const restoreSavedRange = (savedRange) => {
    if (!savedRange || !editorRef.current) return
    try {
      const sel = window.getSelection()
      sel.removeAllRanges()
      sel.addRange(savedRange)
      editorRef.current.focus()
    } catch (err) {
      console.warn('选区恢复失败，降级光标到末尾', err)
      const fallbackRange = document.createRange()
      fallbackRange.selectNodeContents(editorRef.current)
      fallbackRange.collapse(false)
      const sel = window.getSelection()
      sel.removeAllRanges()
      sel.addRange(fallbackRange)
    }
  }

  // 保存当前状态到撤销栈
  const saveHistory = useCallback(() => {
    if (!editorRef.current || isUndoingOrRedoing.current) return

    const currentHtml = editorRef.current.innerHTML
    const currentState = {
      html: currentHtml,
      selection: getCurrentSelection(),
    }

    // 避免重复保存相同的状态
    if (undoStack.current.length > 0 && undoStack.current[undoStack.current.length - 1].html === currentHtml) {
      return
    }

    undoStack.current.push(currentState)
    // 限制栈深度
    if (undoStack.current.length > MAX_HISTORY) {
      undoStack.current.shift()
    }
    // 产生新操作，清空重做栈
    redoStack.current = []
  }, [])

  // 每200ms合并为一次历史记录
  // eslint-disable-next-line react-hooks/refs
  const debouncedSaveHistory = useRef(debounce(saveHistory, 200)).current

  // 执行撤销
  const handleUndo = useCallback(() => {
    if (undoStack.current.length <= 1) return // 至少保留初始状态
    isUndoingOrRedoing.current = true

    // 将当前状态推入重做栈
    const currentState = undoStack.current.pop()
    redoStack.current.push(currentState)

    // 恢复上一个状态
    const prevState = undoStack.current[undoStack.current.length - 1]
    editorRef.current.innerHTML = prevState.html
    restoreSelection(prevState.selection)

    // 基于书签恢复光标
    if (prevState.bookmarkId) {
      const bookmark = editorRef.current.querySelector(`#${prevState.bookmarkId}`)
      if (bookmark) {
        const range = document.createRange()
        range.setStartBefore(bookmark)
        range.collapse(true)
        bookmark.remove() // 移除书签
        const sel = window.getSelection()
        sel.removeAllRanges()
        sel.addRange(range)
      }
    }

    // 触发外部 onChange
    isInternalChange.current = true
    onChange?.(prevState.html)
    updateCurrentFormat()

    isUndoingOrRedoing.current = false
  }, [onChange, updateCurrentFormat])

  // 执行重做
  const handleRedo = useCallback(() => {
    if (redoStack.current.length === 0) return
    isUndoingOrRedoing.current = true

    const nextState = redoStack.current.pop()
    undoStack.current.push(nextState)

    editorRef.current.innerHTML = nextState.html
    restoreSelection(nextState.selection)

    isInternalChange.current = true
    onChange?.(nextState.html)
    updateCurrentFormat()

    isUndoingOrRedoing.current = false
  }, [onChange, updateCurrentFormat])

  // 插入分割线
  const handleHr = () => {
    editorRef.current.focus()
    const selection = window.getSelection()
    if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) return null
    const range = selection.getRangeAt(0)

    const hr = document.createElement('hr')
    hr.style = 'margin: 20px 0; border-top: 1px solid rgb(230, 232, 235);'
    const br = document.createElement('br')
    // 先清空选区内容，插入分割线
    range.deleteContents()
    range.insertNode(hr)
    // 将光标移到hr后面，插入换行
    range.setStartAfter(hr)
    range.insertNode(br)
    // 光标定位到br后方，方便直接打字
    range.setStartAfter(br)
    range.collapse(true)

    selection.removeAllRanges()
    selection.addRange(range)
  }

  // 受控组件初始化与外部数据同步
  useEffect(() => {
    if (!isInternalChange.current && editorRef.current && value !== editorRef.current.innerHTML) {
      editorRef.current.innerHTML = value
    }
    isInternalChange.current = false
  }, [value])

  // 初始化时记录第一条历史
  useEffect(() => {
    if (editorRef.current && undoStack.current.length === 0) {
      saveHistory()
    }
  }, [saveHistory])

  // 换前先保存选区文本偏移
  const saveRangeOffset = (root) => {
    const sel = window.getSelection()
    if (!sel.rangeCount) return null
    const oldRange = sel.getRangeAt(0)

    // 获取从root起点到range起点的总字符偏移
    let startOffset = 0
    let endOffset = 0

    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null)

    let node = null
    let pos = 0
    let startFound = false
    let endFound = false

    while ((node = walker.nextNode())) {
      const len = node.textContent?.length ?? 0
      if (!startFound && node === oldRange.startContainer) {
        startOffset = pos + oldRange.startOffset
        startFound = true
      }
      if (!endFound && node === oldRange.endContainer) {
        endOffset = pos + oldRange.endOffset
        endFound = true
      }
      pos += len
    }
    return { startOffset, endOffset }
  }

  // 恢复选区
  const restoreRangeByOffset = (root, offsetInfo) => {
    if (!offsetInfo) return
    const { startOffset, endOffset } = offsetInfo
    const range = document.createRange()

    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, null)
    let node = null
    let pos = 0
    let startSet = false
    let endSet = false

    while ((node = walker.nextNode())) {
      const len = node.textContent?.length ?? 0
      if (!startSet && pos + len >= startOffset) {
        range.setStart(node, startOffset - pos)
        startSet = true
      }
      if (!endSet && pos + len >= endOffset) {
        range.setEnd(node, endOffset - pos)
        endSet = true
      }
      pos += len
      if (startSet && endSet) break
    }

    const sel = window.getSelection()
    sel.removeAllRanges()
    sel.addRange(range)
  }

  // 截取选区边界
  const splitRangeBoundaries = (range) => {
    const { startContainer, startOffset, endContainer, endOffset } = range

    // 处理起点：文本节点且不是边界，分割
    if (startContainer.nodeType === Node.TEXT_NODE) {
      const textNode = startContainer
      if (startOffset > 0 && startOffset < textNode.length) {
        const afterNode = textNode.splitText(startOffset)
        range.setStart(afterNode, 0)
      }
    }

    // 处理终点：文本节点且不是边界，分割
    if (endContainer.nodeType === Node.TEXT_NODE) {
      const textNode = endContainer
      if (endOffset > 0 && endOffset < textNode.length) {
        const beforeLen = endOffset
        textNode.splitText(beforeLen)
        // end 停留在原来前半部分
        range.setEnd(textNode, beforeLen)
      }
    }
  }

  // 处理行内样式
  const handleInLineCommand = (key, value, range, rootEl) => {
    // 仅处理有选区的情况
    if (range.collapsed) return
    const plainText = range.toString()
    if (!plainText.trim()) return
    // 切换行内样式
    const applyInlineStyle = (el, key, value) => {
      const computed = getComputedStyle(el)
      const toKebabCase = (str) => str.replace(/[A-Z]/g, (m) => '-' + m.toLowerCase())
      let styleObj = {}

      switch (key) {
        case 'fontFamily':
          styleObj.fontFamily = value
          break
        case 'fontSize':
          styleObj.fontSize = value
          break
        case 'bold':
          styleObj.fontWeight = 'bold'
          break
        case 'italic':
          styleObj.fontStyle = 'italic'
          break
        case 'underline': {
          const decoUnder = computed.textDecoration || ''
          if (!decoUnder.includes('underline')) {
            styleObj.textDecoration = [decoUnder, 'underline'].filter(Boolean).join(' ')
          } else {
            styleObj.textDecoration = ''
          }
          break
        }
        case 'strike': {
          const decoStrike = computed.textDecoration || ''
          if (!decoStrike.includes('line-through')) {
            styleObj.textDecoration = [decoStrike, 'line-through'].filter(Boolean).join(' ')
          } else {
            styleObj.textDecoration = ''
          }
          break
        }
        case 'textColor':
          styleObj.color = value
          break
        case 'backgroundColor':
          styleObj.backgroundColor = value
          break
      }

      const elStyle = el.style
      for (const [cssKey, cssValue] of Object.entries(styleObj)) {
        const kebabKey = toKebabCase(cssKey)
        if (cssValue === '') {
          elStyle.removeProperty(kebabKey)
          continue
        }
        if (['bold', 'italic', 'underline', 'strike'].includes(key) && elStyle[cssKey] === cssValue) {
          elStyle.removeProperty(kebabKey)
        } else {
          elStyle[cssKey] = cssValue
        }
      }
    }

    // 保存选区
    const offset = saveRangeOffset(rootEl)

    // 父span是否只包含当前这一个node，没有其他子内容
    const isOnlyChildInSpan = (node) => {
      const parent = node.parentElement
      if (!parent) return false
      // 父必须是span
      if (parent.nodeName !== 'SPAN') return false
      // span只能有一个直接子节点
      if (parent.childNodes.length !== 1) return false

      // 把两边空白、换行全部清理再对比
      const parentText = parent.textContent.replace(/\s+/g, ' ').trim()
      const nodeText = node.textContent.replace(/\s+/g, ' ').trim()

      return parentText === nodeText
    }

    let ancestor = range.commonAncestorContainer
    const startNode = range.startContainer
    const endNode = range.endContainer

    // 如果公共祖先本身是文本节点，则取它的父元素作为迭代树根
    if (ancestor.nodeType === Node.TEXT_NODE) {
      ancestor = ancestor.parentElement
    }

    // 优先判断：选区起点终点是否在同一个span内部
    let startInLine = startNode.nodeType === Node.TEXT_NODE ? startNode.parentElement : startNode
    while (startInLine && startInLine.nodeName !== 'SPAN' && startInLine !== rootEl) {
      startInLine = startInLine.parentElement
    }
    let endInLine = endNode.nodeType === Node.TEXT_NODE ? endNode.parentElement : endNode
    while (endInLine && endInLine.nodeName !== 'SPAN' && endInLine !== rootEl) {
      endInLine = endInLine.parentElement
    }
    if (startInLine && endInLine && startInLine === endInLine) {
      const elementToRange = (el) => {
        const r = document.createRange()
        r.selectNode(el)
        return r
      }
      const spanRange = elementToRange(startInLine)
      const s = range.compareBoundaryPoints(Range.START_TO_START, spanRange)
      const e = range.compareBoundaryPoints(Range.END_TO_END, spanRange)
      // s <=0 && e >=0 → 选区把整个span完整包住
      if (s <= 0 && e >= 0) {
        ancestor = startInLine
      }
      spanRange.detach()
    }

    // 不能跑出编辑器容器
    if (!rootEl.contains(ancestor)) {
      ancestor = rootEl
    }

    // 全部信息读完之后，才分割边界！！
    splitRangeBoundaries(range)

    const childNodes = []
    const nodeIter = document.createNodeIterator(ancestor, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT, {
      acceptNode(node) {
        if (node === ancestor) {
          // SKIP：ancestor本身不要收集，但是遍历它子节点
          return NodeFilter.FILTER_SKIP
        }

        // div/li块：不要块节点本身，但继续遍历内部子节点 → FILTER_SKIP，严禁REJECT
        if (['DIV', 'LI'].includes(node.nodeName)) {
          return NodeFilter.FILTER_SKIP
        }

        // 只处理文本 / span
        if (node.nodeType === Node.TEXT_NODE || node.nodeName === 'SPAN') {
          // 判断：node的全部内容 落在range内部
          const r2 = document.createRange()
          r2.selectNodeContents(node)
          // range完全包含r2
          const isFullInside =
            range.compareBoundaryPoints(Range.START_TO_START, r2) <= 0 && range.compareBoundaryPoints(Range.END_TO_END, r2) >= 0
          r2.detach()
          return isFullInside ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP
        }

        // 其它类型节点：跳过，但是不切断子树
        return NodeFilter.FILTER_SKIP
      },
    })

    let curr
    while ((curr = nodeIter.nextNode())) {
      childNodes.push(curr)
    }

    // 逐个处理选中的行内节点
    for (const child of childNodes) {
      if (child.nodeType === Node.TEXT_NODE) {
        if (!child.textContent.trim()) continue

        const parent = child.parentElement
        if (isOnlyChildInSpan(child)) {
          applyInlineStyle(parent, key, value)
        } else {
          // 需要新建span包裹
          const wrapperSpan = document.createElement('span')
          wrapperSpan.textContent = child.textContent
          applyInlineStyle(wrapperSpan, key, value)
          child.replaceWith(wrapperSpan)
        }
      }
      if (child.nodeType === Node.ELEMENT_NODE) {
        if (child.nodeName === 'SPAN') {
          applyInlineStyle(child, key, value)
        }
      }
    }

    // 恢复选区
    restoreRangeByOffset(rootEl, offset)
  }

  // 处理块级样式
  const handleBlockCommand = (key, value, range, rootEl) => {
    // 仅处理有选区的情况
    if (range.collapsed) return
    const plainText = range.toString()
    if (!plainText.trim()) return

    // 保存选区
    const offset = saveRangeOffset(rootEl)
    // 处理样式
    const transStyle = (el, key, value) => {
      const computed = getComputedStyle(el)
      let blockStyle = {}

      // 获取当前元素fontSize，用于px → em换算
      const fontSizePx = parseFloat(computed.fontSize) || 16
      switch (key) {
        case 'textAlign':
          blockStyle.textAlign = value
          break
        case 'lineHeight':
          blockStyle.lineHeight = value
          break
        case 'plusIndent': {
          const mlPx = parseFloat(computed.marginLeft) || 0
          const mlEm = mlPx / fontSizePx
          const nextEm = mlEm + 2
          blockStyle.marginLeft = `${nextEm}em`
          break
        }
        case 'minusIndent': {
          const mlPx = parseFloat(computed.marginLeft) || 0
          const mlEm = mlPx / fontSizePx
          const nextEm = Math.max(0, mlEm - 2)
          if (nextEm > 0) {
            blockStyle.marginLeft = `${nextEm}em`
          } else {
            blockStyle.marginLeft = ''
          }
          break
        }
      }

      return blockStyle
    }

    const startNode = range.startContainer
    const endNode = range.endContainer

    const blockTags = ['DIV', 'LI']
    // 优先判断：选区起点终点是否在同一个DIV内部
    let startBlock = startNode.nodeType === Node.TEXT_NODE ? startNode.parentElement : startNode
    while (startBlock && !blockTags.includes(startBlock.nodeName) && startBlock !== rootEl) {
      startBlock = startBlock.parentElement
    }
    let endBlock = endNode.nodeType === Node.TEXT_NODE ? endNode.parentElement : endNode
    while (endBlock && !blockTags.includes(endBlock.nodeName) && endBlock !== rootEl) {
      endBlock = endBlock.parentElement
    }
    const childNodes = []
    if (startBlock && endBlock && rootEl.contains(startBlock) && rootEl.contains(endBlock)) {
      let current = startBlock
      while (current) {
        if (blockTags.includes(current.nodeName)) {
          childNodes.push(current)
        }
        if (current === endBlock) break
        current = current.nextSibling
      }
    }

    // 分割边界
    splitRangeBoundaries(range)

    // 逐个处理选中的块级节点
    for (const child of childNodes) {
      const blockElStyle = transStyle(child, key, value)
      Object.assign(child.style, blockElStyle)
    }

    // 恢复选区
    restoreRangeByOffset(rootEl, offset)
  }

  // 处理列表
  const handleListCommand = (key, range, rootEl) => {
    // 仅处理有选区的情况
    if (range.collapsed) return
    const plainText = range.toString()
    if (!plainText.trim()) return

    // 保存选区
    const offset = saveRangeOffset(rootEl)
    // 分割边界
    splitRangeBoundaries(range)

    const startNode = range.startContainer
    const endNode = range.endContainer

    const blockTags = ['DIV', 'LI']
    // 优先判断：选区起点终点是否在同一个DIV内部
    let startBlock = startNode.nodeType === Node.TEXT_NODE ? startNode.parentElement : startNode
    while (startBlock && !blockTags.includes(startBlock.nodeName) && startBlock !== rootEl) {
      startBlock = startBlock.parentElement
    }
    let endBlock = endNode.nodeType === Node.TEXT_NODE ? endNode.parentElement : endNode
    while (endBlock && !blockTags.includes(endBlock.nodeName) && endBlock !== rootEl) {
      endBlock = endBlock.parentElement
    }
    const childNodes = []
    if (startBlock && endBlock && rootEl.contains(startBlock) && rootEl.contains(endBlock)) {
      let current = startBlock
      while (current) {
        if (blockTags.includes(current.nodeName)) {
          childNodes.push(current)
        }
        if (current === endBlock) break
        current = current.nextSibling
      }
    }

    // 分割边界
    splitRangeBoundaries(range)

    // 按 parentNode 分组，避免选区时混入外部节点
    const groups = []
    let lastParent = null
    for (const node of childNodes) {
      if (node.parentNode !== lastParent) {
        groups.push([])
        lastParent = node.parentNode
      }
      groups[groups.length - 1].push(node)
    }

    for (const groupNodes of groups) {
      const isInListMode = groupNodes.some((node) => node.nodeName === 'LI')

      if (isInListMode) {
        const liNodes = groupNodes.filter((node) => node.nodeName === 'LI')
        const commonParent = liNodes[0].parentElement
        if (commonParent.nodeName === key.toUpperCase()) {
          for (const liEl of groupNodes) {
            if (liEl.nodeName !== 'LI') continue
            const divEl = document.createElement('div')
            divEl.style.cssText = liEl.style.cssText
            while (liEl.firstChild) {
              divEl.appendChild(liEl.firstChild)
            }
            liEl.replaceWith(divEl)
          }
          if (commonParent) {
            const ulParent = commonParent.parentNode
            while (commonParent.firstChild) {
              ulParent.insertBefore(commonParent.firstChild, commonParent)
            }
            commonParent.remove()
          }
        } else {
          const targetListTag = key.toUpperCase()
          const listWrapper = document.createElement(targetListTag)
          Object.assign(listWrapper.style, {
            listStyleType: targetListTag === 'UL' ? 'disc' : 'decimal',
            marginLeft: '20px',
          })
          while (commonParent.firstChild) {
            listWrapper.appendChild(commonParent.firstChild)
          }
          commonParent.parentNode.replaceChild(listWrapper, commonParent)
        }
      } else {
        const targetListTag = key.toUpperCase()
        const listWrapper = document.createElement(targetListTag)
        Object.assign(listWrapper.style, {
          listStyleType: targetListTag === 'UL' ? 'disc' : 'decimal',
          marginLeft: '20px',
        })

        if (groupNodes[0]) {
          rootEl.insertBefore(listWrapper, groupNodes[0])
        }

        for (const divEl of groupNodes) {
          if (divEl.nodeName !== 'DIV') continue
          const liEl = document.createElement('li')
          liEl.style.cssText = divEl.style.cssText
          while (divEl.firstChild) {
            liEl.appendChild(divEl.firstChild)
          }
          listWrapper.appendChild(liEl)
          divEl.remove()
        }
      }
    }

    // 恢复选区
    restoreRangeByOffset(rootEl, offset)
  }

  // 命令分发系统
  const executeCommand = useCallback(
    (key, value) => {
      if (!editorRef.current) return
      // 操作前强制恢复缓存选区 + 保存当前真实选区
      const sel = window.getSelection()
      // 优先恢复失焦缓存的选区
      if ((savedRange.current && !sel.rangeCount) || !editorRef.current.contains(sel.anchorNode)) {
        restoreSavedRange(savedRange.current)
      }

      const selection = window.getSelection()
      if (!selection.rangeCount) return
      const originRange = selection.getRangeAt(0)

      // 隔离状态下禁止格式化
      if (currentFormat.isMediaSelected) return

      // 撤销/重做单独处理
      if (key === 'undo') {
        handleUndo()
        return
      }
      if (key === 'redo') {
        handleRedo()
        return
      }
      // 插入分割线
      if (key === 'hr') {
        handleHr()
      }

      const spanKeys = ['fontFamily', 'fontSize', 'bold', 'italic', 'underline', 'strike', 'textColor', 'backgroundColor']
      const divKeys = ['plusIndent', 'minusIndent', 'textAlign', 'lineHeight']
      const listKeys = ['ul', 'ol']
      // 处理 span 样式
      if (spanKeys.includes(key)) {
        // 执行DOM修改
        handleInLineCommand(key, value, originRange, editorRef.current)
      }
      // 处理 div 样式
      if (divKeys.includes(key)) {
        handleBlockCommand(key, value, originRange, editorRef.current)
      }
      // 处理列表
      if (listKeys.includes(key)) {
        handleListCommand(key, originRange, editorRef.current)
      }

      // 操作完成后，触发内容变化并更新工具栏状态
      isInternalChange.current = true
      onChange?.(editorRef.current.innerHTML)
      updateCurrentFormat()

      saveHistory()
    },
    [currentFormat, onChange, updateCurrentFormat, handleUndo, handleRedo, saveHistory]
  )

  // 粘贴处理：仅接收图片和纯文本，丢弃所有富文本格式
  const handlePaste = useCallback(
    (e) => {
      e.preventDefault()

      const clipboardData = e.clipboardData
      if (!clipboardData) return

      const selection = window.getSelection()
      if (!selection.rangeCount) return
      const range = selection.getRangeAt(0)

      // 处理图片粘贴
      const items = clipboardData.items
      for (let i = 0; i < items.length; i++) {
        if (items[i].type.indexOf('image') !== -1) {
          const file = items[i].getAsFile()
          if (file) {
            const img = document.createElement('img')
            img.src = URL.createObjectURL(file)
            range.deleteContents()
            range.insertNode(img)

            const newRange = document.createRange()
            newRange.setStartAfter(img)
            newRange.collapse(true)
            selection.removeAllRanges()
            selection.addRange(newRange)
          }
          isInternalChange.current = true
          onChange?.(editorRef.current.innerHTML)
          saveHistory()
          return
        }
      }

      // 处理纯文本粘贴（丢弃所有 HTML 标签和样式）
      const plainText = clipboardData.getData('text/plain')
      if (!plainText) return

      range.deleteContents()
      const textNode = document.createTextNode(plainText)
      range.insertNode(textNode)

      // 【修正】光标移到粘贴文本的后面
      const newRange = document.createRange()
      newRange.setStartAfter(textNode)
      newRange.collapse(true)
      selection.removeAllRanges()
      selection.addRange(newRange)

      isInternalChange.current = true
      onChange?.(editorRef.current.innerHTML)
      saveHistory()
    },
    [onChange, saveHistory]
  )

  // 监听键盘快捷键 (Ctrl+Z / Ctrl+Y)
  const handleKeyDown = useCallback(
    (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
        e.preventDefault()
        handleUndo()
      } else if ((e.ctrlKey || e.metaKey) && (e.key === 'y' || (e.shiftKey && e.key === 'z'))) {
        e.preventDefault()
        handleRedo()
      }
    },
    [handleUndo, handleRedo]
  )
  // 内容变化时触发
  const handleInput = () => {
    if (editorRef.current && onChange) {
      isInternalChange.current = true
      onChange(editorRef.current.innerHTML)
    }
    // 更新当前格式
    updateCurrentFormat()

    // 保存历史记录
    debouncedSaveHistory()

    // 添加占位块
    debounce(addPlaceholderBlock(editorRef.current), 300)

    // 保存当前的 Range，以便后续命令使用
    const selection = window.getSelection()
    if (selection.rangeCount > 0) {
      savedRange.current = selection.getRangeAt(0)
    }
  }

  // 处理编辑器失焦
  const handleBlur = () => {
    const selection = window.getSelection()
    if (selection.rangeCount > 0) {
      // 保存当前的 Range，以便后续命令使用
      savedRange.current = selection.getRangeAt(0)
    }
  }

  // 渲染工具栏项
  const ToolbarItem = ({ item, currentFormat, executeCommand }) => {
    const [isOpen, setIsOpen] = useState(false)
    const isDisabled = currentFormat.isMediaSelected && !['undo', 'redo'].includes(item.key)

    useEffect(() => {
      const handleClickOutside = (event) => {
        if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
          setIsOpen(false)
        }
      }
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }, [])

    switch (item.type) {
      case 'divider':
        return <div key={item.type} className='toolbar-divider' />

      case 'button':
      case 'toggle': {
        const isActive = item.type === 'toggle' && currentFormat[item.key] === true
        return (
          <div
            key={item.key}
            className={`toolbar-btn ${isActive ? 'active' : ''}`}
            title={item.title}
            disabled={isDisabled}
            onMouseDown={(e) => {
              e.stopPropagation()
              e.preventDefault()
              executeCommand(item.key)
            }}>
            <item.icon />
          </div>
        )
      }

      case 'select': {
        const currentLabel = item.options?.find((opt) => opt.value === currentFormat[item.key])?.label

        return (
          <div className='toolbar-select-wrapper' title={item.title} ref={dropdownRef}>
            <div className='toolbar-select-trigger' onClick={() => setIsOpen(!isOpen)}>
              {item.icon ? (
                <>
                  <item.icon />
                  <span className='toolbar-select-arrow'>
                    <DownIcon />
                  </span>
                </>
              ) : (
                <>
                  <span
                    className={['toolbar-select-value', item.key === 'fontFamily' && 'family', item.key === 'fontSize' && 'size']
                      .filter(Boolean)
                      .join(' ')}>
                    {currentLabel || item.defaultValue || item.title}
                  </span>
                  <span className='toolbar-select-arrow'>
                    <DownIcon />
                  </span>
                </>
              )}
            </div>
            {isOpen && (
              <div className='toolbar-select-dropdown'>
                <div className='dropdown-content'>
                  {item.options.map((opt) => (
                    <div
                      key={opt.value}
                      className='toolbar-select-option'
                      style={item.key === 'fontFamily' ? { fontFamily: opt.value } : undefined}
                      onMouseDown={(e) => {
                        e.stopPropagation()
                        e.preventDefault()
                        executeCommand(item.key, opt.value)
                        setIsOpen(false)
                      }}>
                      {opt.label}
                      {opt.value === currentFormat[item.key] && <Checkcon className='toolbar-select-option-icon' />}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )
      }

      case 'color': {
        return (
          <ColorPicker
            key={item.key}
            title={item.title}
            disabled={isDisabled}
            icon={item.icon}
            defaultValue={item.defaultValue}
            currentColor={currentFormat[item.key]}
            onChange={(color) => executeCommand(item.key, color)}
            addAfter={
              <span className='toolbar-select-arrow'>
                <DownIcon />
              </span>
            }
          />
        )
      }

      default:
        return null
    }
  }

  useEffect(() => {
    const el = editorRef.current
    if (!el) return
    addPlaceholderBlock(el)
  }, [])

  return (
    <div className='rich-text-editor'>
      <div className='rich-text-editor__toolbar'>
        {toolBarItems.map((item, index) => (
          <ToolbarItem key={index} item={item} currentFormat={currentFormat} executeCommand={executeCommand} />
        ))}
      </div>

      <div
        ref={editorRef}
        className='rich-text-editor__content'
        contentEditable
        suppressContentEditableWarning
        onInput={handleInput}
        onMouseUp={updateCurrentFormat}
        onKeyUp={updateCurrentFormat}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        onBlur={handleBlur}
      />
    </div>
  )
}

export default RichTextEditor

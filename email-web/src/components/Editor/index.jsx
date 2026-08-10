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
  { label: '1.0', value: 1.43 },
  { label: '1.15', value: 1.64 },
  { label: '1.3', value: 1.86 },
  { label: '1.5', value: 2.15 },
  { label: '2.0', value: 2.86 },
  { label: '3.0', value: 4.29 },
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

  // 完整保存选区（克隆Range，防止节点引用失效）
  const saveSelectionRange = () => {
    const sel = window.getSelection()
    if (!sel.rangeCount) return null
    return sel.getRangeAt(0).cloneRange()
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

  // 保存选区元信息（用于操作后还原选中高亮，不只是光标）
  const getSelectionMeta = (range) => {
    return {
      text: range.toString(),
      collapsed: range.collapsed,
      clone: range.cloneRange(),
    }
  }

  // 合并相邻的、样式相同的 span 标签
  const mergeAdjacentSpans = (container) => {
    const spans = container.querySelectorAll('span')
    for (let i = 0; i < spans.length - 1; i++) {
      const currentSpan = spans[i]
      const nextSpan = spans[i + 1]

      // 只有当两个节点都是 span，并且样式字符串完全相同时才合并
      if (
        nextSpan &&
        currentSpan.nodeName === 'SPAN' &&
        nextSpan.nodeName === 'SPAN' &&
        currentSpan.getAttribute('style') === nextSpan.getAttribute('style')
      ) {
        // 将后一个 span 的内容移动到前一个 span 的末尾
        while (nextSpan.firstChild) {
          currentSpan.appendChild(nextSpan.firstChild)
        }
        // 移除已经合并的后一个 span
        nextSpan.remove()
        // 合并后，索引需要回退一步，以检查新合并的 span 是否能和它后面的 span 继续合并
        i--
      }
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

  // Toggle 类型命令
  const handleToggleCommand = (key) => {
    switch (key) {
      case 'bold':
      case 'italic':
      case 'underline':
      case 'strike':
        // 行内样式 span
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 无选区（仅光标）直接跳过，不做任何操作
          if (range.collapsed) return

          // 定义样式映射关系
          const styleMap = {
            bold: { prop: 'fontWeight', value: 'bold', check: (v) => v === 'bold' || parseInt(v) >= 700 },
            italic: { prop: 'fontStyle', value: 'italic', check: (v) => v === 'italic' },
            underline: { prop: 'textDecoration', value: 'underline', check: (v) => v?.includes('underline') },
            strike: { prop: 'textDecoration', value: 'line-through', check: (v) => v?.includes('line-through') },
          }

          const currentStyle = styleMap[key]
          if (!currentStyle) return

          // 判断选区内是否“全部”都带有该样式（用于决定是添加还是剥离）
          let allHaveStyle = true
          let hasAnyText = false

          const checkWalker = document.createTreeWalker(
            range.cloneContents(), // 在选区的克隆副本上遍历，避免修改原 DOM 导致遍历出错
            NodeFilter.SHOW_TEXT,
            null
          )

          let textNode
          while ((textNode = checkWalker.nextNode())) {
            if (textNode.textContent.trim()) {
              hasAnyText = true
              const parentSpan = textNode.parentElement
              if (parentSpan?.nodeName === 'SPAN') {
                if (!currentStyle.check(parentSpan.style[currentStyle.prop])) {
                  allHaveStyle = false
                  break
                }
              } else {
                // 纯文本节点，说明没有样式
                allHaveStyle = false
                break
              }
            }
          }

          // 如果选区内没有文本，直接返回
          if (!hasAnyText) return

          // 提取选区内容
          const fragment = range.extractContents()

          // 创建一个临时容器来承载处理后的内容，方便进行规范化操作
          const tempContainer = document.createElement('span')
          tempContainer.appendChild(fragment)
          // ========== 情况 A：全部都有样式 -> 执行剥离（Remove） ==========
          if (allHaveStyle) {
            const spans = tempContainer.querySelectorAll('span')
            spans.forEach((span) => {
              // 移除对应的样式
              if (currentStyle.prop === 'textDecoration') {
                // textDecoration 可能包含多个值（如 underline line-through），需要精准移除
                const decorations = span.style.textDecoration.split(' ').filter((d) => d !== currentStyle.value)
                if (decorations.length > 0) {
                  span.style.textDecoration = decorations.join(' ')
                } else {
                  span.style.removeProperty('text-decoration')
                }
              } else {
                span.style.removeProperty(currentStyle.prop)
              }

              // 如果 span 已经没有任何 style 属性了，直接将其替换为纯文本节点
              if (!span.getAttribute('style')) {
                const newTextNode = document.createTextNode(span.textContent)
                span.parentNode.replaceChild(newTextNode, span)
              }
            })
          }
          // ========== 情况 B：没有或部分有样式 -> 执行添加（Add） ==========
          else {
            // 遍历片段中的所有文本节点，用带有样式的 <span> 包裹
            const addWalker = document.createTreeWalker(tempContainer, NodeFilter.SHOW_TEXT, {
              acceptNode: (node) => {
                // 过滤掉空白文本节点
                if (!node.textContent.trim()) {
                  return NodeFilter.FILTER_REJECT
                }

                // 检查该文本节点的父级是否已经包含了目标样式（
                let parent = node.parentElement
                while (parent && parent !== tempContainer) {
                  if (parent.nodeName === 'SPAN' && parent.style.fontWeight === 'bold') {
                    return NodeFilter.FILTER_REJECT
                  }
                  parent = parent.parentElement
                }

                // 剩下的就是需要被 <span> 包裹的纯文本节点
                return NodeFilter.FILTER_ACCEPT
              },
            })
            const nodesToWrap = []
            let node
            while ((node = addWalker.nextNode())) {
              if (node.textContent.trim()) nodesToWrap.push(node)
            }
            nodesToWrap.forEach((textNode) => {
              const wrapper = document.createElement('span')

              // 如果文本节点的父级已经是 span，先继承它原有的样式
              if (textNode.parentElement?.nodeName === 'SPAN') {
                wrapper.style.cssText = textNode.parentElement.style.cssText
              }

              // 叠加新的样式
              if (currentStyle.prop === 'textDecoration') {
                // 处理 textDecoration 的叠加
                const existing = wrapper.style.textDecoration || ''
                if (!existing.includes(currentStyle.value)) {
                  wrapper.style.textDecoration = `${existing} ${currentStyle.value}`.trim()
                }
              } else {
                wrapper.style[currentStyle.prop] = currentStyle.value
              }
              // 用新的 span 替换原有的文本节点
              wrapper.textContent = textNode.textContent
              textNode.parentNode.replaceChild(wrapper, textNode)
            })
          }
          // 规范化：合并相邻的相同样式 span
          mergeAdjacentSpans(tempContainer)
          // 将规范化后的内容插回选区
          range.insertNode(tempContainer)

          // 将光标/选区重新定位到刚刚插入的内容之后
          const newRange = document.createRange()
          if (newRange.parentNode) {
            newRange.setStartAfter(newRange)
            newRange.collapse(true)
            selection.removeAllRanges()
            selection.addRange(newRange)
          }
        }
        break
      case 'ul':
      case 'ol':
        // 列表切换逻辑 (div <-> li)
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 辅助函数：向上查找离节点最近的块级容器 (div 或 li)
          const getBlockNode = (node) => {
            if (node.nodeType === Node.TEXT_NODE) node = node.parentElement
            while (node && node !== editorRef.current) {
              if (['DIV', 'LI'].includes(node.nodeName)) return node
              node = node.parentElement
            }
            return null
          }

          // 收集选区覆盖的所有块级节点（使用 Set 去重）
          const blockNodes = new Set()
          const startBlock = getBlockNode(range.startContainer)
          if (startBlock) blockNodes.add(startBlock)

          if (!range.collapsed) {
            const endBlock = getBlockNode(range.endContainer)
            if (endBlock) blockNodes.add(endBlock)

            const walker = document.createTreeWalker(editorRef.current, NodeFilter.SHOW_ELEMENT, {
              acceptNode: (node) => {
                if (['DIV', 'LI'].includes(node.nodeName) && range.intersectsNode(node)) {
                  return NodeFilter.FILTER_ACCEPT
                }
                return NodeFilter.FILTER_SKIP
              },
            })
            let currentNode
            while ((currentNode = walker.nextNode())) {
              blockNodes.add(currentNode)
            }
          }

          // 遍历所有收集到的块级节点，执行列表的打包或解包
          blockNodes.forEach((node) => {
            const targetListTag = key === 'ul' ? 'UL' : 'OL'

            // ========== 情况 A：当前节点已经是目标列表项 -> 执行解包（Unwrap） ==========
            if (node.nodeName === 'LI' && node.parentElement?.nodeName === targetListTag) {
              // 创建一个新的 <div> 来承载原 <li> 的内容
              const newDiv = document.createElement('div')
              newDiv.innerHTML = node.innerHTML // 保留原有的行内样式（如加粗、颜色等）

              // 将新 <div> 插入到 <ul>/<ol> 的前面
              node.parentElement.parentNode.insertBefore(newDiv, node.parentElement)

              // 移除当前的 <li>
              node.remove()

              // 如果 <ul>/<ol> 变空了，将其也移除
              if (node.parentElement.children.length === 0) {
                node.parentElement.remove()
              }
            }
            // ========== 情况 B：当前节点是普通 <div> 或其他列表项 -> 执行打包（Wrap） ==========
            else if (node.nodeName === 'DIV') {
              // 创建新的 <li>，继承原 <div> 的内容
              const newLi = document.createElement('li')
              newLi.innerHTML = node.innerHTML

              // 显式继承原 div 的行内样式（如 color, font-size 等）
              if (node.style.cssText) {
                newLi.style.cssText = node.style.cssText
              }

              // 查找或创建对应的 <ul>/<ol> 容器
              let listContainer = null

              // 如果前一个兄弟节点已经是目标列表，直接复用
              const prevSibling = node.previousSibling
              if (prevSibling?.nodeName === targetListTag) {
                listContainer = prevSibling
              } else {
                // 否则创建一个新的列表容器
                listContainer = document.createElement(targetListTag.toLowerCase())
                node.parentNode.insertBefore(listContainer, node)
              }

              // 将 <li> 放入列表容器，并移除原 <div>
              listContainer.appendChild(newLi)
              node.remove()
            }
          })

          // 列表操作后，选区通常会丢失，这里简单将光标重置到编辑器末尾
          selection.removeAllRanges()
          const newRange = document.createRange()
          newRange.selectNodeContents(editorRef.current)
          newRange.collapse(false) // 折叠到末尾
          selection.addRange(newRange)
        }
        break
      default:
        console.warn(`未实现的 Toggle 命令: ${key}`)
    }
  }

  // 处理 Select 类型命令
  const handleSelectCommand = (key, value) => {
    switch (key) {
      case 'fontFamily':
      case 'fontSize':
        // 用 <span> 包裹并设置样式
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 无选区（仅光标）直接跳过，不做任何操作
          if (range.collapsed) return

          // 提取选区内的纯文本内容
          const plainText = range.toString()
          if (!plainText) return // 如果选区内没有文本，直接返回

          // 创建新的 <span> 标签，并设置对应的颜色样式
          const newSpan = document.createElement('span')
          newSpan.textContent = plainText

          if (key === 'fontFamily') {
            newSpan.style.fontFamily = value
          }
          if (key === 'fontSize') {
            newSpan.style.fontSize = value
          }

          // 清除选区原有内容，并插入新的 <span>
          range.deleteContents()
          range.insertNode(newSpan)

          // 将光标/选区重新定位到刚刚插入的 <span> 之后
          const newRange = document.createRange()
          newRange.setStartAfter(newSpan)
          newRange.collapse(true)
          selection.removeAllRanges()
          selection.addRange(newRange)
        }
        break
      case 'textColor':
      case 'backgroundColor':
        // 用 <span> 包裹并设置样式
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 提取选区内的纯文本内容
          const plainText = range.toString()
          if (!plainText) return // 如果选区内没有文本，直接返回

          // 创建新的 <span> 标签，并设置对应的颜色样式
          const newSpan = document.createElement('span')
          newSpan.textContent = plainText

          if (key === 'textColor') {
            newSpan.style.color = value
          } else if (key === 'backgroundColor') {
            newSpan.style.backgroundColor = value
          }

          // 清除选区原有内容，并插入新的带有颜色的 <span>
          range.deleteContents()
          range.insertNode(newSpan)

          // 将光标/选区重新定位到刚刚插入的 <span> 之后
          const newRange = document.createRange()
          newRange.setStartAfter(newSpan)
          newRange.collapse(true)
          selection.removeAllRanges()
          selection.addRange(newRange)
        }
        break
      case 'textAlign':
      case 'lineHeight':
        // 找到块级 div/li 并设置样式
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 辅助函数：向上查找离节点最近的块级容器 (div 或 li)
          const getBlockNode = (node) => {
            if (node.nodeType === Node.TEXT_NODE) node = node.parentElement
            while (node && node !== editorRef.current) {
              if (['DIV', 'LI'].includes(node.nodeName)) return node
              node = node.parentElement
            }
            return null
          }

          // 收集选区覆盖的所有块级节点
          const blockNodes = new Set() // 使用 Set 避免重复添加同一个块级节点

          // 添加起始块级节点
          const startBlock = getBlockNode(range.startContainer)
          if (startBlock) blockNodes.add(startBlock)

          // 如果选区跨行了，需要遍历中间的块级节点
          if (!range.collapsed) {
            const endBlock = getBlockNode(range.endContainer)
            if (endBlock) blockNodes.add(endBlock)

            // 遍历选区内的所有节点，找出属于编辑器内部的 div/li
            const walker = document.createTreeWalker(editorRef.current, NodeFilter.SHOW_ELEMENT, {
              acceptNode: (node) => {
                if (['DIV', 'LI'].includes(node.nodeName) && range.intersectsNode(node)) {
                  return NodeFilter.FILTER_ACCEPT
                }
                return NodeFilter.FILTER_SKIP
              },
            })

            let currentNode
            while ((currentNode = walker.nextNode())) {
              blockNodes.add(currentNode)
            }
          }

          // 遍历所有收集到的块级节点，应用样式
          blockNodes.forEach((node) => {
            if (key === 'textAlign') {
              if (value === DEFAULT_FORMAT.textAlign) {
                node.style.removeProperty('text-align')
              } else {
                node.style.textAlign = value
              }
            } else if (key === 'lineHeight') {
              if (value === DEFAULT_FORMAT.lineHeight) {
                node.style.removeProperty('line-height')
              } else {
                node.style.lineHeight = value
              }
            }
          })
        }
        break
      default:
        console.warn(`未实现的 Select 命令: ${key}`)
    }
  }

  // 处理 Button 类型命令
  const handleButtonCommand = (key) => {
    switch (key) {
      case 'plusIndent':
      case 'minusIndent':
        // 块级 div/li 增加减少 margin-left
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 向上查找当前光标所在的块级节点 (div 或 li)
          let blockNode = range.startContainer
          if (blockNode.nodeType === Node.TEXT_NODE) blockNode = blockNode.parentElement
          while (blockNode && blockNode !== editorRef.current) {
            if (['DIV', 'LI'].includes(blockNode.nodeName)) break
            blockNode = blockNode.parentElement
          }

          // 如果找不到块级节点，直接 return
          if (!blockNode || blockNode === editorRef.current) return

          // 获取当前的 margin-left 值，如果没有则默认为 0
          const currentMargin = parseFloat(blockNode.style.marginLeft) || 0
          const STEP = 2 // 每次缩进的步长，单位为 em

          // 根据是增加还是减少，计算新的 margin-left
          let newMargin = currentMargin
          if (key === 'plusIndent') {
            newMargin = currentMargin + STEP
          } else if (key === 'minusIndent') {
            newMargin = Math.max(0, currentMargin - STEP) // 防止缩进变成负数
          }

          // 应用新的样式
          if (newMargin > 0) {
            blockNode.style.marginLeft = `${newMargin}em`
          } else {
            // 如果缩进归零，直接移除该样式，保持 DOM 干净
            blockNode.style.removeProperty('margin-left')
          }
        }
        break
      case 'clear':
        // 清除格式逻辑 (移除选区内所有 span 样式及块级样式)
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 无选区（仅光标）直接跳过，不做任何操作
          if (range.collapsed) return

          // 提取选区内的纯文本内容（自动过滤掉所有 HTML 标签和样式）
          const plainText = range.toString()

          // 找到当前光标所在的块级节点 (div 或 li)
          let blockNode = range.startContainer
          if (blockNode.nodeType === Node.TEXT_NODE) blockNode = blockNode.parentElement
          while (blockNode && blockNode !== editorRef.current) {
            if (['DIV', 'LI'].includes(blockNode.nodeName)) break
            blockNode = blockNode.parentElement
          }

          // 如果找到了块级节点，直接用纯文本重建它
          if (blockNode && blockNode !== editorRef.current) {
            // 清空原有内容，插入纯文本
            blockNode.textContent = plainText

            // 重置该块级节点的样式为默认值
            blockNode.style.textAlign = DEFAULT_FORMAT.textAlign
            blockNode.style.lineHeight = DEFAULT_FORMAT.lineHeight
            blockNode.style.textIndent = DEFAULT_FORMAT.textIndent

            // 重建后，将光标定位到纯文本的末尾
            const newRange = document.createRange()
            newRange.setStart(blockNode.firstChild, plainText.length)
            newRange.collapse(true)
            selection.removeAllRanges()
            selection.addRange(newRange)
          }
        }
        break
      case 'hr':
        // 插入 <div><hr /></div>
        {
          const selection = window.getSelection()
          // 如果当前没有选区，或者选区不在编辑器内，恢复之前保存的选区
          if (!selection.rangeCount || !editorRef.current?.contains(selection.anchorNode)) {
            if (savedRange.current) {
              selection.removeAllRanges()
              selection.addRange(savedRange.current)
            } else {
              editorRef.current?.focus()
              return
            }
          }
          // 重新获取正确的选区
          const range = selection.getRangeAt(0)

          // 找到当前光标所在的块级节点 (div 或 li)
          let currentBlock = range.startContainer
          if (currentBlock.nodeType === Node.TEXT_NODE) currentBlock = currentBlock.parentElement
          while (currentBlock && currentBlock !== editorRef.current) {
            if (['DIV', 'LI'].includes(currentBlock.nodeName)) break
            currentBlock = currentBlock.parentElement
          }

          // 如果找不到块级节点，直接 return
          if (!currentBlock || currentBlock === editorRef.current) return

          // 创建分割线节点: <div><hr /></div>
          const hrWrapper = document.createElement('div')
          const hrElement = document.createElement('hr')
          hrWrapper.appendChild(hrElement)

          // 将分割线插入到当前块级节点的后面
          // 如果当前块级节点有父级（比如 li 在 ul 里面），则插在父级后面；否则插在自身后面
          const insertTarget =
            currentBlock.parentNode === editorRef.current ? currentBlock : currentBlock.closest('ul, ol') || currentBlock

          insertTarget.parentNode.insertBefore(hrWrapper, insertTarget.nextSibling)

          // 将光标移动到分割线下方（创建一个新的空 div）
          const newLine = document.createElement('div')
          newLine.innerHTML = '<br>' // 保证空行有高度
          hrWrapper.parentNode.insertBefore(newLine, hrWrapper.nextSibling)

          // 重新设置选区，将光标定位到新插入的空行中
          const newRange = document.createRange()
          newRange.setStart(newLine, 0)
          newRange.collapse(true) // 折叠选区，变成光标
          selection.removeAllRanges()
          selection.addRange(newRange)
        }
        break
      default:
        console.warn(`未实现的 Button 命令: ${key}`)
    }
  }

  // 将 CSS 文本转换为对象
  const cssTextToStyleObj = (cssText) => {
    const styleObj = {}
    if (!cssText?.trim()) return styleObj

    // 分割每条样式
    const rules = cssText.split(';').filter((rule) => rule.trim())

    for (const rule of rules) {
      const [propRaw, val] = rule.split(':').map((s) => s.trim())
      if (!propRaw || val === undefined) continue

      // 短横线转驼峰
      const prop = propRaw.replace(/-(\w)/g, (_, char) => char.toUpperCase())
      styleObj[prop] = val
    }
    return styleObj
  }

  //  range 是否完整选中整个 node（包含全部内容，无多余/缺失）
  const isRangeCoversWholeNode = (range, node) => {
    const nodeRange = document.createRange()
    nodeRange.selectNodeContents(node)
    return (
      range.compareBoundaryPoints(Range.START_TO_START, nodeRange) <= 0 &&
      range.compareBoundaryPoints(Range.END_TO_END, nodeRange) >= 0
    )
  }

  // 提取选区内部所有相交的 span 节点
  const getRangeInnerSpans = (range, rootEl) => {
    const spanSet = new Set()
    // 向上收集起止节点父span（真实DOM，无文档问题）
    const collectParentSpans = (node) => {
      let cur = node
      while (cur && cur !== rootEl) {
        if (cur.nodeName === 'SPAN') spanSet.add(cur)
        cur = cur.parentNode
      }
    }
    collectParentSpans(range.commonAncestorContainer)
    collectParentSpans(range.startContainer)
    collectParentSpans(range.endContainer)

    // 直接遍历编辑器真实DOM，过滤和选区相交的span，不用cloneContents
    const walker = document.createTreeWalker(rootEl, NodeFilter.SHOW_ELEMENT, {
      acceptNode: (el) => {
        if (el.nodeName === 'SPAN' && range.intersectsNode(el)) {
          return NodeFilter.FILTER_ACCEPT
        }
        return NodeFilter.FILTER_SKIP
      },
    })
    let span
    while ((span = walker.nextNode())) {
      spanSet.add(span)
    }
    return Array.from(spanSet)
  }

  // 判断是否是纯文本
  const isText = (frag) => {
    const walker = document.createTreeWalker(frag, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT, null)

    let node
    while ((node = walker.nextNode())) {
      // 只要遇到任意元素节点，就不是纯文本
      if (node.nodeType === Node.ELEMENT_NODE) {
        return false
      }
    }
    // 全部都是文本节点
    return true
  }

  // 创建 span 包裹纯文本
  const wrapSpan = (htmlStr, spanStyle) => {
    const div = document.createElement('div')
    div.innerHTML = htmlStr

    // 复制子节点列表（遍历中会改动dom，不能直接遍历childNodes）
    const childNodes = Array.from(div.childNodes)

    for (const node of childNodes) {
      // 判断是纯文本节点，且不是全空白
      if (node.nodeType === Node.TEXT_NODE && node.textContent.trim() !== '') {
        const span = document.createElement('span')
        span.textContent = node.textContent
        // 样式
        Object.assign(span.style, spanStyle)

        node.replaceWith(span)
      } else {
        Object.entries(spanStyle).forEach(([styleKey, styleVal]) => {
          node.style[styleKey] = styleVal
        })
      }
    }
    const frag = document.createDocumentFragment()
    while (div.firstChild) {
      frag.appendChild(div.firstChild)
    }
    return frag
  }
  // 处理行内样式
  const handleSpanCommand = (key, value, range) => {
    let plainText = range.toString()
    if (!plainText) return null

    // 样式
    let spanStyle = {}
    switch (key) {
      case 'fontFamily':
        spanStyle.fontFamily = value
        break
      case 'fontSize':
        spanStyle.fontSize = value
        break
      case 'bold':
        spanStyle.fontWeight = 'bold'
        break
      case 'italic':
        spanStyle.fontStyle = 'italic'
        break
      case 'underline':
        {
          // 多装饰叠加，不覆盖原有
          const decoUnder = spanStyle.textDecoration || ''
          if (!decoUnder.includes('underline')) {
            spanStyle.textDecoration = [decoUnder, 'underline'].filter(Boolean).join(' ')
          }
        }
        break
      case 'strike':
        {
          const decoStrike = spanStyle.textDecoration || ''
          if (!decoStrike.includes('line-through')) {
            spanStyle.textDecoration = [decoStrike, 'line-through'].filter(Boolean).join(' ')
          }
        }
        break
      case 'textColor':
        spanStyle.color = value
        break
      case 'backgroundColor':
        spanStyle.backgroundColor = value
        break
    }

    // 内部所有相交span
    const innerSpans = getRangeInnerSpans(range, editorRef.current)
    // 被选区完整包裹的顶层span（优先复用）
    const fullCoverSpans = innerSpans.filter((span) => isRangeCoversWholeNode(range, span))
    let targetSpan = null

    if (fullCoverSpans.length > 0) {
      // 存在完整选中的span：复用第一个顶层span，不新建
      targetSpan = fullCoverSpans[0]
      // 合并原有span所有样式
      Object.assign(spanStyle, cssTextToStyleObj(targetSpan.style.cssText))

      // 统一设置文本内容
      targetSpan.textContent = plainText

      Object.entries(spanStyle).forEach(([styleKey, styleVal]) => {
        targetSpan.style[styleKey] = styleVal
      })
    } else {
      const frag = range.cloneContents()
      if (isText(frag)) {
        // 无可用span，新建
        targetSpan = document.createElement('span')
        Object.entries(spanStyle).forEach(([styleKey, styleVal]) => {
          targetSpan.style[styleKey] = styleVal
        })
        // 统一设置文本内容
        targetSpan.textContent = plainText
      } else {
        // 取出选区HTML
        const div = document.createElement('div')
        div.appendChild(frag)
        const wrapFrag = wrapSpan(div.innerHTML, spanStyle)
        console.log(wrapFrag)
        return wrapFrag
      }
    }

    return targetSpan
  }

  // 处理块级样式
  const handleDivCommand = (key, value) => {
    console.log(key, value)
  }

  // 处理列表
  const handleListCommand = (key) => {
    console.log(key)
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
      const newSel = window.getSelection()
      if (!newSel.rangeCount) return
      const originRange = newSel.getRangeAt(0)
      // 记录选区原始状态
      const selectionMeta = getSelectionMeta(originRange)

      // 隔离状态下禁止格式化（仅允许撤销/重做）
      if (currentFormat.isMediaSelected && !['undo', 'redo'].includes(key)) return

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
        // 仅处理有选区的情况
        if (originRange.collapsed) return
        const plainText = selectionMeta.text
        if (!plainText.trim()) return

        // 执行DOM修改
        const newSpan = handleSpanCommand(key, value, originRange)
        originRange.deleteContents()
        originRange.insertNode(newSpan)
        mergeAdjacentSpans(newSpan.parentElement)

        // 修复选中逻辑，不再以span自身做range容器
        const newSelRange = document.createRange()
        const spanParent = newSpan.parentNode
        const spanIndex = Array.from(spanParent?.childNodes).indexOf(newSpan)
        newSelRange.setStart(spanParent, spanIndex)
        newSelRange.setEnd(spanParent, spanIndex + 1)

        const s = window.getSelection()
        s.removeAllRanges()
        s.addRange(newSelRange)
      }
      // 处理 div 样式
      if (divKeys.includes(key)) {
        handleDivCommand(key, value, originRange)
      }
      // 处理列表
      if (listKeys.includes(key)) {
        handleListCommand(key, originRange)
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
    updateCurrentFormat()

    debouncedSaveHistory()

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

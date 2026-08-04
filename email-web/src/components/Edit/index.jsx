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

  // 获取选区
  const getSelectionRange = () => {
    const sel = window.getSelection()
    if (!sel || !sel.rangeCount || sel.isCollapsed) return null
    return sel.getRangeAt(0)
  }

  // 获取所选节点
  const getSelectedBlocks = (targetTags = ['DIV', 'LI', 'P'], stopAtList = false) => {
    const range = getSelectionRange()
    if (!range) return []

    let startNode = range.startContainer
    let endNode = range.endContainer

    // 处理纯文本
    const wrapIfDirectTextChild = (node) => {
      if (node.nodeType === Node.TEXT_NODE) {
        const parent = node.parentElement
        const editorContent = editorRef.current
        if (parent && parent === editorContent) {
          const div = document.createElement('div')
          parent.insertBefore(div, node)
          div.appendChild(node)
          return div
        }
      }
      return node.nodeType === Node.TEXT_NODE ? node.parentElement : node
    }

    startNode = wrapIfDirectTextChild(startNode)
    endNode = wrapIfDirectTextChild(endNode)

    if (!startNode || !endNode) return []

    const findBlockParent = (node) => {
      let current = node
      while (current && current !== editorRef.current) {
        if (current.nodeType === Node.ELEMENT_NODE && targetTags.includes(current.tagName)) {
          if (stopAtList && ['UL', 'OL'].includes(current.tagName)) {
            current = current.parentElement
            continue
          }
          return current
        }
        current = current.parentElement
      }
      return null
    }

    const startBlock = findBlockParent(startNode)
    const endBlock = findBlockParent(endNode)

    if (!startBlock || !endBlock) return []

    const blocks = []
    if (startBlock === endBlock) {
      blocks.push(startBlock)
    } else {
      let current = startBlock
      while (current) {
        blocks.push(current)
        if (current === endBlock) break
        current = current.nextElementSibling
      }
    }
    return blocks
  }

  // 设置样式
  const setStyle = (styleObj) => {
    const nodesToStyle = getSelectedBlocks(['DIV', 'LI', 'P'])
    if (nodesToStyle.length === 0) return

    nodesToStyle.forEach((node) => {
      Object.entries(styleObj).forEach(([key, value]) => {
        const hasSameStyle = node.style[key] === value

        if (hasSameStyle) {
          node.style[key] = ''
          if (!node.getAttribute('style')) node.removeAttribute('style')
        } else {
          node.style[key] = value
        }
      })
    })
  }

  // 设置列表
  const setList = (type) => {
    const range = getSelectionRange()
    if (!range) return

    const startContainer = range.startContainer
    const startOffset = range.startOffset
    const endContainer = range.endContainer
    const endOffset = range.endOffset

    const currentListBlocks = getSelectedBlocks(['UL', 'OL'], false)
    const isInList = currentListBlocks.length > 0 && ['UL', 'OL'].includes(currentListBlocks[0].tagName)

    let blocksToConvert = []
    let anchorNode = null

    if (isInList) {
      const listToUnwrap = currentListBlocks[0]
      const fragment = document.createDocumentFragment()

      Array.from(listToUnwrap.children).forEach((li) => {
        const div = document.createElement('div')
        div.style.cssText = li.style.cssText
        while (li.firstChild) div.appendChild(li.firstChild)
        fragment.appendChild(div)
      })

      anchorNode = fragment.firstChild
      listToUnwrap.parentNode.replaceChild(fragment, listToUnwrap)
    }

    if (isInList && currentListBlocks[0].tagName === type) {
      return
    }

    if (anchorNode) {
      let current = anchorNode
      while (current) {
        blocksToConvert.push(current)
        current = current.nextElementSibling
      }
    } else {
      blocksToConvert = getSelectedBlocks(['DIV', 'LI', 'P'], true)
    }

    if (blocksToConvert.length === 0) return

    const listElement = document.createElement(type)
    listElement.style.paddingLeft = '28px'
    listElement.style.listStyleType = type === 'UL' ? 'disc' : 'decimal'

    blocksToConvert.forEach((block) => {
      const li = document.createElement('li')
      if (block.style.cssText) li.style.cssText = block.style.cssText
      while (block.firstChild) li.appendChild(block.firstChild)
      listElement.appendChild(li)
    })

    const firstBlock = blocksToConvert[0]
    firstBlock.parentNode.insertBefore(listElement, firstBlock)

    blocksToConvert.forEach((block) => {
      if (!block.hasChildNodes()) block.remove()
    })

    const sel = window.getSelection()
    const newRange = document.createRange()

    try {
      if (document.contains(startContainer) && document.contains(endContainer)) {
        newRange.setStart(startContainer, startOffset)
        newRange.setEnd(endContainer, endOffset)
      } else {
        newRange.selectNodeContents(listElement)
        newRange.collapse(false)
      }

      sel.removeAllRanges()
      sel.addRange(newRange)
    } catch (e) {
      console.error('恢复选区失败：', e)
      newRange.selectNodeContents(listElement)
      newRange.collapse(false)
      sel.removeAllRanges()
      sel.addRange(newRange)
    }
  }

  // 分割线
  const setHr = () => {
    const editor = editorRef.current
    if (!editor) return
    const range = getSelectionRange()
    const hr = document.createElement('hr')
    hr.style.margin = '16px 0px'
    hr.style.borderTop = '1px solid rgb(230, 232, 235)'
    if (range) {
      range.insertNode(hr)
    } else {
      editor.appendChild(hr)
    }
  }

  // 清除格式
  const clearFormat = () => {
    const blocks = getSelectedBlocks(['DIV', 'LI', 'UL', 'OL'])
    if (blocks.length === 0) return
    const listsToUnwrap = new Set()
    blocks.forEach((block) => {
      const tagName = block.tagName

      if (tagName === 'UL' || tagName === 'OL') {
        listsToUnwrap.add(block)
      }
      if (tagName === 'DIV' || tagName === 'LI') {
        block.removeAttribute('style')

        if (tagName === 'LI' && block.parentElement) {
          const parentList = block.parentElement
          if (parentList.tagName === 'UL' || parentList.tagName === 'OL') {
            listsToUnwrap.add(parentList)
          }
        }
      }
    })

    listsToUnwrap.forEach((listToUnwrap) => {
      const fragment = document.createDocumentFragment()

      Array.from(listToUnwrap.children).forEach((li) => {
        const div = document.createElement('div')
        while (li.firstChild) div.appendChild(li.firstChild)
        fragment.appendChild(div)
      })

      listToUnwrap.parentNode.replaceChild(fragment, listToUnwrap)
    })
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

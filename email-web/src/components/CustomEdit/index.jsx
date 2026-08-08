import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { createEditor, Editor, Element as SlateElement, Transforms } from 'slate'
import { HistoryEditor, withHistory } from 'slate-history'
import { Editable, ReactEditor, Slate, withReact } from 'slate-react'

// 样式文件
import './index.scss'

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

//  颜色选择器子组件
import ColorPicker from './ColorPicker'

// 配置数据
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

const LINE_HEIGHTS = [
  { label: '1.0', value: 1.43 },
  { label: '1.15', value: 1.64 },
  { label: '1.3', value: 1.86 },
  { label: '1.5', value: 2.15 },
  { label: '2.0', value: 2.86 },
  { label: '3.0', value: 4.29 },
]

const TEXT_ALIGN = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' },
  { label: '两端对齐', value: 'justify' },
]

// 样式映射
const MARK_TO_CSS_MAP = {
  bold: 'font-weight: bold',
  italic: 'font-style: italic',
  underline: 'text-decoration: underline',
  strike: 'text-decoration: line-through',
  color: (val) => `color: ${val}`,
  bgcolor: (val) => `background-color: ${val}`,
  lineHeight: (val) => `line-height: ${val}`,
  textAlign: (val) => `text-align: ${val}`,
  fontFamily: (val) => `font-family: ${val}`,
  fontSize: (val) => `font-size: ${val}`,
  listStyleType: (val) => `list-style-type: ${val}`,
  textIndent: (val) => `text-indent: ${val}em`,
}
const ELEMENT_TAGS = {
  DIV: () => ({ type: 'paragraph' }),
  P: () => ({ type: 'paragraph' }),
  IMG: (el) => ({ type: 'image', url: el.getAttribute('src'), children: [{ text: '' }] }),
  HR: () => ({ type: 'hr', children: [{ text: '' }] }),
}

const TEXT_TAGS = {
  SPAN: (el) => parseStyleAttribute(el),
  B: () => ({ bold: true }),
  STRONG: () => ({ bold: true }),
  I: () => ({ italic: true }),
  EM: () => ({ italic: true }),
  U: () => ({ underline: true }),
  S: () => ({ strike: true }),
  STRIKE: () => ({ strike: true }),
}

// 解析内联样式字符串
const parseStyleAttribute = (el) => {
  const styles = {}
  const styleAttr = el.getAttribute('style')
  if (styleAttr) {
    styleAttr.split(';').forEach((style) => {
      const [property, value] = style.split(':').map((s) => s && s.trim())
      if (property && value) {
        switch (property) {
          case 'font-weight':
            if (value === 'bold' || parseInt(value) >= 600) styles.bold = true
            break
          case 'font-style':
            if (value === 'italic') styles.italic = true
            break
          case 'text-decoration':
            if (value.includes('underline')) styles.underline = true
            if (value.includes('line-through')) styles.strike = true
            break
          case 'color':
            styles.color = value
            break
          case 'background-color':
            styles.backgroundColor = value
            break
          case 'font-family':
            styles.fontFamily = value
            break
          case 'font-size':
            styles.fontSize = value
            break
        }
      }
    })
  }
  return styles
}

// 工具函数
const getBlockProperties = (editor) => {
  if (!editor.selection) return {}

  const [match] = Editor.nodes(editor, {
    match: (n) => !Editor.isEditor(n) && SlateElement.isElement(n),
    universal: true,
  })
  const blockAttrs = match ? match[0] : {}

  const marks = Editor.marks(editor) || {}

  return { ...blockAttrs, ...marks }
}

// 文件转 Base64
const fileToBase64 = (file) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = (error) => reject(error)
    reader.readAsDataURL(file)
  })
}

const isMarkActive = (editor, format) => {
  const marks = Editor.marks(editor)
  return marks ? marks[format] === true : false
}

// Toolbar 配置化渲染组件
const ToolbarItem = ({ item, editor, blockProps }) => {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef(null)

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // 分割线
  if (item.type === 'divider') {
    return <div className='toolbar-divider' />
  }

  // 颜色选择器
  if (item.type === 'color') {
    const currentColor = Editor.marks(editor)?.[item.key]
    return (
      <ColorPicker
        icon={item.icon}
        color={currentColor}
        onChange={(c) => {
          Editor.addMark(editor, item.key, c)
          ReactEditor.focus(editor)
        }}
        title={item.title}
        defaultValue={item.defaultValue}
        addAfter={
          <span className='toolbar-select-arrow'>
            <DownIcon />
          </span>
        }
      />
    )
  }

  // 下拉菜单
  if (item.type === 'select') {
    const isBlock = item.target === 'block'
    const currentValue = isBlock
      ? blockProps[item.key] || item.defaultValue
      : Editor.marks(editor)?.[item.key] || item.defaultValue

    const currentLabel = item.options.find((opt) => opt.value === currentValue)?.label || item.title

    const handleSelect = (val) => {
      if (isBlock) {
        Transforms.setNodes(editor, { [item.key]: val }, { match: (n) => SlateElement.isElement(n) })
      } else {
        if (val === undefined || val === '') Editor.removeMark(editor, item.key)
        else Editor.addMark(editor, item.key, val)
      }
      setIsOpen(false)

      ReactEditor.focus(editor)
    }

    return (
      <div className='toolbar-select-wrapper' ref={dropdownRef}>
        <div className='toolbar-select-trigger' onClick={() => setIsOpen(!isOpen)} title={item.title}>
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
                {currentLabel}
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
                  onClick={() => handleSelect(opt.value)}>
                  {opt.label}
                  {opt.value === currentValue && <Checkcon className='toolbar-select-option-icon' />}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    )
  }

  // 按钮
  if (item.type === 'button' || item.type === 'toggle') {
    let isActive = false
    const handleClick = () => {
      if (typeof item.onClick === 'function') {
        item.onClick(editor)
      } else if (item.type === 'toggle') {
        if (item.target === 'mark') {
          const isActiveMark = isMarkActive(editor, item.key)
          if (isActiveMark) Editor.removeMark(editor, item.key)
          else Editor.addMark(editor, item.key, true)
        } else if (item.target === 'block') {
          const isCurrentActive = blockProps[item.key] === item.value
          Transforms.setNodes(
            editor,
            { [item.key]: isCurrentActive ? undefined : item.value },
            { match: (n) => SlateElement.isElement(n) }
          )
        }
      }
      ReactEditor.focus(editor)
    }

    // 计算激活状态
    if (item.type === 'toggle') {
      if (item.target === 'mark') isActive = isMarkActive(editor, item.key)
      if (item.target === 'block') isActive = blockProps[item.key] === item.value
    }

    return (
      <div onClick={handleClick} className={`toolbar-btn${isActive ? ' active' : ''}`} title={item.title}>
        <item.icon />
      </div>
    )
  }

  return null
}

// 编辑器主组件
const RichTextEditor = (props) => {
  const { value: htmlValue, onChange } = props
  const editor = useMemo(() => {
    const e = withHistory(withReact(createEditor()))

    e.isVoid = (element) => {
      return ['image', 'hr'].includes(element.type)
    }

    return e
  }, [])

  const DEFAULT_VALUE = [{ type: 'paragraph', children: [{ text: '' }] }]
  const [editorValue, setEditorValue] = useState(DEFAULT_VALUE)
  const [currentSelection, setCurrentSelection] = useState(null)

  const blockProps = useMemo(() => getBlockProperties(editor), [currentSelection, editor])

  // 工具栏
  const toolbarItems = [
    {
      type: 'button',
      icon: ClearIcon,
      title: '清除格式',
      onClick: (e) => {
        ;['fontFamily', 'fontSize', 'color', 'backgroundColor', 'bold', 'italic', 'underline', 'strike'].forEach((m) =>
          Editor.removeMark(e, m)
        )
        Transforms.setNodes(e, {}, { match: (n) => SlateElement.isElement(n) })
      },
    },
    { type: 'button', icon: UndoIcon, title: '撤销', onClick: (e) => HistoryEditor.undo(e) },
    { type: 'button', icon: RedoIcon, title: '重做', onClick: (e) => HistoryEditor.redo(e) },
    { type: 'divider' },
    {
      type: 'select',
      key: 'fontFamily',
      target: 'mark',
      defaultValue: 'inherit',
      options: FONT_FAMILIES,
      title: '字体',
    },
    {
      type: 'select',
      key: 'fontSize',
      target: 'mark',
      defaultValue: null,
      options: FONT_SIZES,
      title: '字号',
    },
    { type: 'divider' },
    { type: 'toggle', icon: BoldIcon, title: '加粗', key: 'bold', target: 'mark' },
    { type: 'toggle', icon: ItalicIcon, title: '斜体', key: 'italic', target: 'mark' },
    { type: 'toggle', icon: UnderlineIcon, title: '下划线', key: 'underline', target: 'mark' },
    { type: 'toggle', icon: StrikeIcon, title: '删除线', key: 'strike', target: 'mark' },
    {
      type: 'color',
      icon: ColorIcon,
      key: 'color',
      title: '字体颜色',
      defaultValue: {
        color: 'rgb(46, 48, 51)',
        text: '默认颜色',
        default: 'rgb(247, 49, 22)',
      },
    },
    {
      type: 'color',
      icon: BgColorIcon,
      key: 'backgroundColor',
      title: '背景颜色',
      defaultValue: {
        color: '',
        text: '无颜色',
        default: 'rgb(255, 217, 0)',
      },
    },
    { type: 'divider' },
    {
      type: 'toggle',
      icon: UlIcon,
      title: '项目编号',
      key: 'listStyleType',
      target: 'block',
      value: 'ul',
      onClick: (e) =>
        Transforms.setNodes(
          e,
          { listStyleType: blockProps.listStyleType === 'disc' ? undefined : 'disc' },
          { match: (n) => SlateElement.isElement(n) }
        ),
    },
    {
      type: 'toggle',
      icon: OlIcon,
      title: '数字编号',
      key: 'listStyleType',
      target: 'block',
      value: 'ol',
      onClick: (e) =>
        Transforms.setNodes(
          e,
          { listStyleType: blockProps.listStyleType === 'decimal' ? undefined : 'decimal' },
          { match: (n) => SlateElement.isElement(n) }
        ),
    },
    {
      type: 'button',
      icon: IndentPlusIcon,
      title: '增加缩进',
      onClick: (e) => {
        const c = blockProps.textIndent || 0
        Transforms.setNodes(e, { textIndent: c + 2 }, { match: (n) => SlateElement.isElement(n) })
      },
    },
    {
      type: 'button',
      icon: IndentMinusIcon,
      title: '减少缩进',
      onClick: (e) => {
        const c = blockProps.textIndent || 0
        Transforms.setNodes(e, { textIndent: Math.max(0, c - 2) || undefined }, { match: (n) => SlateElement.isElement(n) })
      },
    },
    {
      type: 'select',
      key: 'textAlign',
      target: 'block',
      defaultValue: null,
      options: TEXT_ALIGN,
      title: '对齐',
      icon: AlignIcon,
    },
    {
      type: 'select',
      key: 'lineHeight',
      target: 'block',
      defaultValue: 1.43,
      options: LINE_HEIGHTS,
      title: '行间距',
      icon: LineHeightIcon,
    },
    { type: 'divider' },
    {
      type: 'button',
      icon: HrIcon,
      title: '插入分割线',
      onClick: () => {
        Transforms.insertNodes(editor, { type: 'hr', children: [{ text: '' }] })
        const end = Editor.end(editor, [])
        if (Editor.isEnd(editor, editor.selection.anchor, end)) {
          Transforms.insertNodes(editor, { type: 'paragraph', children: [{ text: '' }] })
        }
        Transforms.select(editor, Editor.end(editor, []))
      },
    },
  ]

  const renderElement = useCallback((props) => {
    const { attributes, children, element } = props
    const style = {}

    if (element.textAlign) {
      style.textAlign = element.textAlign
    }
    if (element.lineHeight) {
      style.lineHeight = element.lineHeight
    }
    if (element.textIndent) {
      style.textIndent = `${element.textIndent}em`
    }

    // 列表样式特殊处理
    if (element.listStyleType) {
      style.display = 'list-item'
      style.marginLeft = '20px'
      style.listStyleType = element.listStyleType
    }

    if (element.type === 'hr') {
      return (
        <div {...attributes} contentEditable={false}>
          <hr style={{ border: 'none', borderTop: '1px solid #ccc', margin: '16px 0' }} />
          {children}
        </div>
      )
    }

    if (element.type === 'image') {
      return (
        <div {...attributes} contentEditable={false}>
          <img src={element.url} alt='pasted' />
          {children}
        </div>
      )
    }

    return (
      <div style={style} {...attributes}>
        {children}
      </div>
    )
  }, [])

  const renderLeaf = useCallback((props) => {
    const { attributes, children, leaf } = props
    let style = {}
    if (leaf.fontFamily) style.fontFamily = leaf.fontFamily
    if (leaf.fontSize) style.fontSize = leaf.fontSize
    if (leaf.color) style.color = leaf.color
    if (leaf.backgroundColor) style.backgroundColor = leaf.backgroundColor
    if (leaf.bold) style.fontWeight = 'bold'
    if (leaf.italic) style.fontStyle = 'italic'
    if (leaf.underline && leaf.strike) style.textDecoration = 'underline line-through'
    else if (leaf.underline) style.textDecoration = 'underline'
    else if (leaf.strike) style.textDecoration = 'line-through'

    return (
      <span style={style} {...attributes}>
        {children}
      </span>
    )
  }, [])

  // 粘贴图片
  const onPaste = async (e, editor) => {
    const clipboardData = e.clipboardData || window.clipboardData
    if (!clipboardData) return

    const items = clipboardData.items

    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.indexOf('image') !== -1) {
        e.preventDefault()
        const file = item.getAsFile()
        if (!file) continue

        try {
          const base64String = await fileToBase64(file)
          Transforms.insertNodes(editor, { type: 'image', url: base64String, children: [{ text: '' }] })
          const end = Editor.end(editor, [])
          if (Editor.isEnd(editor, editor.selection.anchor, end)) {
            Transforms.insertNodes(editor, { type: 'paragraph', children: [{ text: '' }] })
          }
          Transforms.select(editor, Editor.end(editor, []))
        } catch (error) {
          console.error('图片转换失败:', error)
        }

        break
      }
    }
  }

  // Slate JSON -> HTML 转换
  const slateToCleanHtml = (nodes) => {
    if (!nodes || !Array.isArray(nodes)) return ''

    return nodes.map((node) => {
      // 处理文本节点
      if (node.text !== undefined) {
        let text = node.text
        if (!text) return ''

        const styles = []
        Object.keys(MARK_TO_CSS_MAP).forEach((markKey) => {
          if (node[markKey]) {
            const cssValue = MARK_TO_CSS_MAP[markKey]
            if (typeof cssValue === 'function') {
              styles.push(cssValue(node[markKey]))
            } else {
              styles.push(cssValue)
            }
          }
        })

        if (styles.length > 0) {
          const styleString = styles.join('; ')
          text = `<span style="${styleString}">${text}</span>`
        }

        return text
      }

      // 处理元素节点
      if (node.type && node.children) {
        const childrenHtml = slateToCleanHtml(node.children).join('')

        switch (node.type) {
          case 'image':
            return `<img src="${node.url || ''}" alt="${node.alt || ''}" />`

          case 'hr':
            return `<hr style="border: none; border-top: 1px solid #ccc; margin: 16px 0;" />`

          default:
            if (node?.listStyleType || node?.textIndent) {
              const styles = []
              Object.keys(MARK_TO_CSS_MAP).forEach((markKey) => {
                if (node[markKey]) {
                  const cssValue = MARK_TO_CSS_MAP[markKey]
                  if (typeof cssValue === 'function') {
                    styles.push(cssValue(node[markKey]))
                  } else {
                    styles.push(cssValue)
                  }
                }
              })
              if (styles.length > 0) {
                const styleString = styles.join('; ')
                return `<div style="${styleString}">${childrenHtml}</div>`
              }
            }
            return `<div>${childrenHtml}</div>`
        }
      }

      return ''
    })
  }

  // HTML -> Slate JSON
  const htmlToSlate = (html) => {
    if (!html || typeof html !== 'string') {
      return DEFAULT_VALUE
    }

    const deserializeNode = (el) => {
      if (el.nodeType === 3) {
        return { text: el.textContent || '' }
      }

      if (el.nodeType !== 1) {
        return null
      }

      const { nodeName } = el
      let parent = el

      if (nodeName === 'PRE') {
        const text = el.textContent || ''
        parent = document.createElement('p')
        text.split('\n').forEach((line, i) => {
          if (i > 0) parent.appendChild(document.createElement('br'))
          parent.appendChild(document.createTextNode(line))
        })
      }

      if (ELEMENT_TAGS[nodeName]) {
        const props = ELEMENT_TAGS[nodeName](el)
        const children = Array.from(parent.childNodes).map(deserializeNode).filter(Boolean)
        return { ...props, children: children.length ? children : [{ text: '' }] }
      }

      if (TEXT_TAGS[nodeName]) {
        const props = TEXT_TAGS[nodeName](el)
        const children = Array.from(parent.childNodes).map(deserializeNode).flat()
        return children.map((child) => ({ ...child, ...props }))
      }

      return Array.from(parent.childNodes).map(deserializeNode).flat()
    }
    try {
      const parsed = new DOMParser().parseFromString(html, 'text/html')
      const result = Array.from(parsed.body.childNodes).map(deserializeNode).filter(Boolean)

      // 如果解析出来是空数组，也返回默认值
      return result.length > 0 ? result : DEFAULT_VALUE
    } catch (error) {
      console.error('HTML parse error:', error)
      return DEFAULT_VALUE
    }
  }

  // 监控内容改变
  const onChangeEdit = (value) => {
    setCurrentSelection(editor.selection)

    const cleanHtml = slateToCleanHtml(value)?.join('')
    console.log('cleanHtml', cleanHtml)
    onChange(cleanHtml)
  }

  // 初始化
  useEffect(() => {
    const init = () => {
      console.log('初始化', props)
      if (htmlValue && htmlValue.trim() !== '') {
        try {
          const slateValue = htmlToSlate(htmlValue)

          if (Array.isArray(slateValue) && slateValue.length > 0) {
            setEditorValue(slateValue)
          } else {
            setEditorValue(DEFAULT_VALUE)
          }
        } catch (error) {
          console.error('HTML to Slate conversion failed:', error)
          setEditorValue(DEFAULT_VALUE)
        }
      } else {
        setEditorValue(DEFAULT_VALUE)
      }
    }

    init()
  }, [htmlValue])

  return (
    <Slate editor={editor} initialValue={editorValue} value={editorValue} onChange={onChangeEdit}>
      <div className='rich-editor-toolbar'>
        {toolbarItems.map((item, index) => (
          <ToolbarItem key={index} item={item} editor={editor} blockProps={blockProps} />
        ))}
      </div>
      <div className='rich-editor-content'>
        <Editable
          renderElement={renderElement}
          renderLeaf={renderLeaf}
          placeholder='输入正文'
          onPaste={(e) => onPaste(e, editor)}
        />
      </div>
    </Slate>
  )
}

export default RichTextEditor

import { useEffect, useRef, useState } from 'react'

import Down from './icons/down.svg'

import ColorPicker from './ColorPicker'

const ToolBar = (props) => {
  const { items, onCommand } = props
  const wrapRef = useRef()
  const [openType, setOpenType] = useState(null)

  // 点击外部关闭下拉菜单
  useEffect(() => {
    const handleClickOutside = (e) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target)) {
        setOpenType(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // 点击事件
  const onItemClick = (item) => {
    if (item.children && item.children.length > 0) {
      setOpenType(openType === item.type ? null : item.type)
    } else {
      onCommand(item)
      setOpenType(null)
    }
  }

  return (
    <div ref={wrapRef} className='y-mail-toolbar' onMouseDown={(e) => e.preventDefault()}>
      {items.map((item, index) => {
        const key = `${item.type}-${index}`
        if (item.type === 'divider') {
          return <div key={key} className='toolbar-divider' />
        }

        const isOpen = openType === item.type

        // 下拉按钮
        if (item.children && item.children.length > 0) {
          return (
            <div key={key} className='toolbar-dropdown-wrap'>
              <div
                className={`toolbar-dropdown-trigger ${isOpen ? 'active' : ''}`}
                title={item.title || ''}
                onClick={() => onItemClick(item)}>
                {item.icon ? <item.icon className='toolbar-icon' /> : <span>{item.title}</span>}
                <span className='toolbar-arrow-icon'>
                  <Down />
                </span>
              </div>

              {isOpen && (
                <div className='toolbar-dropdown-panel'>
                  {/* 颜色选择器 */}
                  {item.children[0].type === 'color-picker' ? (
                    <ColorPicker
                      defaultColor={item.children[0].color}
                      onChange={(e) => onItemClick({ type: item.type, title: item.title, color: e })}
                    />
                  ) : (
                    // 下拉菜单项
                    item.children.map((child) => (
                      <div key={child.type} className='toolbar-dropdown-item' onClick={() => onItemClick(child)}>
                        {child.title}
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>
          )
        }

        // 普通按钮
        return (
          <div className='toolbar-btn' key={key} title={item.title || ''} onClick={() => onItemClick(item)}>
            {/* 渲染图标组件 */}
            {item.icon ? <item.icon className='toolbar-icon' /> : item.title}
          </div>
        )
      })}
    </div>
  )
}

export default ToolBar

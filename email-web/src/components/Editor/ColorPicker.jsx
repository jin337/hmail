import { useEffect, useRef, useState } from 'react'

const COLORS = [
  'rgb(46, 48, 51)',
  'rgb(254, 43, 35)',
  'rgb(255, 153, 0)',
  'rgb(255, 217, 0)',
  'rgb(15, 128, 36)',
  'rgb(56, 217, 240)',
  'rgb(49, 143, 250)',
  'rgb(170, 22, 208)',
  'rgb(255, 255, 255)',
  'rgb(253, 219, 214)',
  'rgb(253, 233, 208)',
  'rgb(254, 241, 207)',
  'rgb(212, 233, 214)',
  'rgb(222, 243, 243)',
  'rgb(206, 224, 239)',
  'rgb(223, 219, 236)',
  'rgb(230, 230, 230)',
  'rgb(238, 131, 126)',
  'rgb(248, 195, 135)',
  'rgb(255, 218, 92)',
  'rgb(154, 189, 158)',
  'rgb(132, 204, 211)',
  'rgb(137, 176, 206)',
  'rgb(147, 137, 177)',
  'rgb(182, 181, 180)',
  'rgb(213, 19, 40)',
  'rgb(207, 119, 12)',
  'rgb(141, 99, 74)',
  'rgb(85, 123, 92)',
  'rgb(0, 163, 176)',
  'rgb(54, 118, 166)',
  'rgb(118, 92, 131)',
  'rgb(102, 101, 100)',
  'rgb(169, 24, 20)',
  'rgb(136, 71, 2)',
  'rgb(87, 55, 37)',
  'rgb(0, 85, 46)',
  'rgb(0, 118, 122)',
  'rgb(24, 78, 119)',
  'rgb(83, 14, 111)',
]

const ColorPicker = ({ item, executeCommand, addAfter, currentFormat }) => {
  const [isOpen, setIsOpen] = useState(false)
  const pickerRef = useRef(null)

  const handleClickOutside = (event) => {
    if (pickerRef.current && !pickerRef.current.contains(event.target)) {
      setIsOpen(false)
    }
  }

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleColorChange = (newColor) => (e) => {
    e.stopPropagation()
    e.preventDefault()
    executeCommand(item.key, newColor)
    setIsOpen(false)
  }

  return (
    <div className='color-picker-wapper' title={item.title} ref={pickerRef}>
      <div className='toolbar-select-trigger' onClick={() => setIsOpen((prev) => !prev)}>
        <div className='color-picker-btn'>
          <item.icon />
          <span className='toolbar-color-icon' style={{ background: currentFormat[item.key] }}></span>
        </div>
        {addAfter}
      </div>

      {isOpen && (
        <div className='color-picker-dropdown'>
          <div className='content-default' onMouseDown={handleColorChange(item.defaultValue.default)}>
            {item.defaultValue?.text}
          </div>
          <div className='dropdown-content'>
            {COLORS.map((c) => (
              <div
                key={c}
                onMouseDown={handleColorChange(c)}
                className={`color-item ${item.defaultValue.color === c ? 'active' : ''}`}
                style={{ background: c }}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default ColorPicker

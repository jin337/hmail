import { useEffect, useRef, useState } from 'react'
import Checkcon from './icons/check.svg'

const SelectPocker = ({ key, item, currentFormat, executeCommand, addAfter }) => {
  const [isOpen, setIsOpen] = useState(false)
  const dropdownRef = useRef(null)

  const currentLabel = item?.options?.find((opt) => opt.value === currentFormat[item.key])?.label

  const handleClickOutside = (event) => {
    if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
      setIsOpen(false)
    }
  }

  useEffect(() => {
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleOptionMouseDown = (optValue) => (e) => {
    e.stopPropagation()
    e.preventDefault()
    executeCommand(item.key, optValue)
    setIsOpen(false)
  }

  const spanClassName = ['toolbar-select-value', item.key === 'fontFamily' && 'family', item.key === 'fontSize' && 'size']
    .filter(Boolean)
    .join(' ')

  return (
    <div className='toolbar-select-wrapper' title={item.title} key={key || item.key} ref={dropdownRef}>
      <div className='toolbar-select-trigger' onClick={() => setIsOpen((prev) => !prev)}>
        {item.icon ? <item.icon /> : <span className={spanClassName}>{currentLabel || item.defaultValue || item.title}</span>}
        {addAfter}
      </div>

      {isOpen && (
        <div className='toolbar-select-dropdown'>
          <div className='dropdown-content'>
            {item.options.map((opt) => (
              <div
                key={opt.value}
                className='toolbar-select-option'
                style={item.key === 'fontFamily' ? { fontFamily: opt.value } : undefined}
                onMouseDown={handleOptionMouseDown(opt.value)}>
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

export default SelectPocker

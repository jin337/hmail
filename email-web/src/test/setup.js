import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// jsdom 缺少的浏览器 API
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

window.matchMedia = window.matchMedia || ((query) => ({
  matches: false,
  media: query,
  onchange: null,
  addListener: () => {},
  removeListener: () => {},
  addEventListener: () => {},
  removeEventListener: () => {},
  dispatchEvent: () => false,
}))

window.URL.createObjectURL = window.URL.createObjectURL || (() => 'blob:mock')
window.URL.revokeObjectURL = window.URL.revokeObjectURL || (() => {})

afterEach(() => {
  cleanup()
  localStorage.clear()
})

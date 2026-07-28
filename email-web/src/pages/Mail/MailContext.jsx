import { createContext, useContext } from 'react'

const MailContext = createContext(null)

export const MailProvider = ({ children, value }) => {
  return <MailContext.Provider value={value}>{children}</MailContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useMailContext() {
  const context = useContext(MailContext)
  if (!context) {
    throw new Error('useMailContext 必须包裹在 <MailProvider /> 内部')
  }
  return context
}

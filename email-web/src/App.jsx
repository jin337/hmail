import dayjs from 'dayjs'
import { useState } from 'react'

import Home from './views/home'
import Login from './views/login'

const App = () => {
  const checkLoginStatus = () => {
    const user = localStorage.getItem('user')
    const loginTime = localStorage.getItem('loginTime')
    if (!user || !loginTime) return false
    const now = dayjs()
    const oneDay = 24 * 60 * 60 * 1000
    if (now - parseInt(loginTime) > oneDay) {
      localStorage.clear()
      return false
    }
    return true
  }

  const [isLoggedIn, setIsLoggedIn] = useState(checkLoginStatus())
  const [user, setUser] = useState(JSON.parse(localStorage.getItem('user')) || '')

  if (isLoggedIn) {
    return (
      <Login
        onLoginSuccess={() => {
          setIsLoggedIn(true)
          setUser(JSON.parse(localStorage.getItem('user')))
        }}
      />
    )
  }

  return <Home user={user} />
}
export default App

import { createBrowserRouter } from 'react-router'

import Login from 'src/views/Login'
import MailApp from 'src/views/MailApp'

export const router = createBrowserRouter([
  { path: '/', element: <MailApp /> },
  { path: '/login', element: <Login /> },
])

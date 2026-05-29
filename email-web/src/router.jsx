import { createBrowserRouter } from 'react-router'

import Admin from 'src/pages/Admin'
import Home from 'src/views/Home'
import Login from 'src/views/Login'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Home />,
  },
  {
    path: '/login',
    element: <Login />,
  },
  {
    path: '/admin',
    element: <Admin />,
  },
])

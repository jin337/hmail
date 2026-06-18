import { createBrowserRouter } from 'react-router'

import Home from 'src/views/Home'
import Login from 'src/views/Login'

import Mail from 'src/pages/Mail'
import User from 'src/pages/User'
import Preview from 'src/pages/Preview'

export const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
  },
  {
    element: <Home />,
    children: [
      {
        path: '/',
        element: <Mail />,
      },
      {
        path: '/user',
        element: <User />,
      },
      {
        path: '/preview',
        element: <Preview />,
      },
    ],
  },
])

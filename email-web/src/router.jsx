import { createBrowserRouter } from 'react-router'

import Home from 'src/views/Home'
import Login from 'src/views/Login'
import Preview from 'src/views/Preview'

import Mail from 'src/pages/Mail'
import Personal from 'src/pages/Personal'
import User from 'src/pages/User'

export const router = createBrowserRouter(
  [
    {
      path: '/login',
      element: <Login />,
    },
    {
      path: '/preview',
      element: <Preview />,
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
          path: '/personal',
          element: <Personal />,
        },
      ],
    },
  ],
  {
    basename: '/web',
  }
)

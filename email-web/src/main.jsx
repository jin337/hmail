import { createRoot } from 'react-dom/client'

// 路由
import { RouterProvider } from 'react-router'
import { router } from './router'

// 样式
import '@arco-design/web-react/dist/css/arco.css'
import '@wangeditor/editor/dist/css/style.css'
import './index.css'

import '@arco-design/web-react/es/_util/react-19-adapter'

createRoot(document.getElementById('root')).render(<RouterProvider router={router} />)

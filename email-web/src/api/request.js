import axios from 'axios'
const service = axios.create({
  baseURL:
    import.meta.env.MODE === 'development'
      ? import.meta.env.VITE_BASE_URL
      : 'http://' + window.location.hostname + ':' + window.location.port,
  timeout: 10000,
})

// 请求拦截器 带token
service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('mail_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (err) => Promise.reject(err)
)

// 响应拦截器
service.interceptors.response.use(
  (res) => res.data,
  (err) => Promise.reject(err)
)

export default service

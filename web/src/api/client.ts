/**
 * axios 实例与拦截器
 *
 * 后端存在两种响应结构：
 * 1. user/auth 接口：{ success, data, message }
 * 2. knowledge 接口：裸数组 / 裸对象，错误为 { error: string }
 * 响应拦截器在出错时兼容 message / error 两种错误字段，统一抛出 Error(后端文案)。
 */
import axios, { AxiosError } from 'axios'
import type { ApiErrorBody } from '../types'
import { clearToken, getToken } from '../utils/auth'

/** 默认走 /api/v1，可通过 VITE_API_BASE_URL 覆盖 */
export const API_BASE_URL: string = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
})

// 请求拦截器：自动附加 Bearer token
http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

/** 登录/注册接口的 401 表示凭证错误，需把后端文案交给页面提示，不能跳登录页 */
function isAuthRequest(url: string | undefined): boolean {
  return !!url && url.includes('/user/')
}

// 响应拦截器：统一错误文案提取；受保护接口 401 时清 token 并跳登录页
http.interceptors.response.use(
  (response) => response,
  (error: unknown) => {
    const axiosError = error as AxiosError<ApiErrorBody>
    const status = axiosError.response?.status
    const url = axiosError.config?.url
    const body = axiosError.response?.data
    // 兼容两种错误字段：user 接口的 message 与 knowledge 接口的 error
    const text: string =
      (body && (body.message || body.error)) || '网络异常，请稍后重试'

    if (status === 401 && !isAuthRequest(url)) {
      clearToken()
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }

    return Promise.reject(new Error(text))
  },
)

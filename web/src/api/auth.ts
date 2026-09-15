/**
 * 用户认证相关接口封装
 */
import { http } from './client'
import type {
  ApiEnvelope,
  LoginData,
  LoginRequest,
  RegisterRequest,
} from '../types'

/** 登录：成功返回含 token 的用户数据 */
export async function login(payload: LoginRequest): Promise<LoginData> {
  const res = await http.post<ApiEnvelope<LoginData>>('/user/login', payload)
  if (!res.data.success || !res.data.data) {
    throw new Error(res.data.message || '登录失败')
  }
  return res.data.data
}

/** 注册：成功无返回数据，失败抛出后端 message */
export async function register(payload: RegisterRequest): Promise<void> {
  const res = await http.post<ApiEnvelope<string>>('/user/register', payload)
  if (!res.data.success) {
    throw new Error(res.data.message || '注册失败')
  }
}

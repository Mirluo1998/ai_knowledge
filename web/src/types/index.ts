/**
 * 全局接口类型定义
 */

/** user/auth 接口统一响应包装：{ success, data, message } */
export interface ApiEnvelope<T> {
  success: boolean
  data: T | null
  message: string
}

/** 后端错误响应体：user 接口用 message，knowledge 接口用 error */
export interface ApiErrorBody {
  message?: string
  error?: string
}

/** 登录用户信息（不含 token） */
export interface UserInfo {
  id: number
  username: string
  email: string
}

/** 登录请求体 */
export interface LoginRequest {
  username: string
  password: string
}

/** 登录成功响应 data */
export interface LoginData extends UserInfo {
  token: string
}

/** 注册请求体 */
export interface RegisterRequest {
  username: string
  password: string
  email: string
}

/** 知识条目（与后端 knowledge 表字段一一对应） */
export interface KnowledgeItem {
  id: number
  type: number
  title: string
  content: string
  created_at: string
  updated_at: string
}

/** 知识列表查询参数（对应 GET /knowledge 的 type、title） */
export interface KnowledgeQuery {
  type?: number
  title?: string
}

/** 新建知识请求体（对应 PUT /knowledge，id 由后端生成） */
export interface KnowledgeCreatePayload {
  type: number
  title: string
  content: string
}

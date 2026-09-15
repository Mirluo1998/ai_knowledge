/**
 * 路由守卫：本地无 token 时重定向到登录页
 */
import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { getToken } from '../utils/auth'

export default function ProtectedRoute({ children }: { children: ReactNode }) {
  const location = useLocation()

  if (!getToken()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }

  return <>{children}</>
}

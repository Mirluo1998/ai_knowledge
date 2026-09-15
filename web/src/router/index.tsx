/**
 * 路由表：/login、/register 为公开页；其余受路由守卫保护
 */
import { createBrowserRouter, Navigate } from 'react-router-dom'
import AppLayout from '../components/AppLayout'
import ProtectedRoute from '../components/ProtectedRoute'
import KnowledgePage from '../pages/KnowledgePage'
import LoginPage from '../pages/LoginPage'
import RegisterPage from '../pages/RegisterPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <Navigate to="/knowledge" replace /> },
      { path: 'knowledge', element: <KnowledgePage /> },
    ],
  },
  { path: '*', element: <Navigate to="/knowledge" replace /> },
])

/**
 * 登录页：居中卡片表单
 */
import { App as AntdApp, Button, Card, Form, Input } from 'antd'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { login } from '../api/auth'
import { setAuth } from '../utils/auth'
import type { LoginRequest } from '../types'

export default function LoginPage() {
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()
  const [submitting, setSubmitting] = useState(false)

  const handleFinish = async (values: LoginRequest) => {
    setSubmitting(true)
    try {
      const data = await login(values)
      setAuth(data.token, {
        id: data.id,
        username: data.username,
        email: data.email,
      })
      message.success('登录成功')
      navigate('/knowledge', { replace: true })
    } catch (err) {
      message.error(err instanceof Error ? err.message : '登录失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-wrapper">
      <Card className="auth-card" title="知识管理系统 - 登录">
        <Form layout="vertical" onFinish={handleFinish} autoComplete="off">
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              block
              loading={submitting}
            >
              登录
            </Button>
          </Form.Item>
          <div>
            还没有账号？<Link to="/register">去注册</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}

/**
 * 全局布局：Header（系统名 + 当前用户 + 退出登录）+ Content
 */
import { App as AntdApp, Button, Layout, Space, Typography } from 'antd'
import { LogoutOutlined } from '@ant-design/icons'
import { Outlet, useNavigate } from 'react-router-dom'
import { clearAuth, getUser } from '../utils/auth'

const { Header, Content } = Layout

export default function AppLayout() {
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()
  const user = getUser()

  const handleLogout = () => {
    clearAuth()
    message.success('已退出登录')
    navigate('/login', { replace: true })
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 24px',
        }}
      >
        <Typography.Title level={4} style={{ color: '#fff', margin: 0 }}>
          知识管理系统
        </Typography.Title>
        <Space size="large">
          <span style={{ color: '#fff' }}>
            当前用户：{user?.username ?? '-'}
          </span>
          <Button
            type="default"
            icon={<LogoutOutlined />}
            onClick={handleLogout}
          >
            退出登录
          </Button>
        </Space>
      </Header>
      <Content style={{ padding: 24 }}>
        <Outlet />
      </Content>
    </Layout>
  )
}

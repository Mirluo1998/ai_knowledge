/**
 * 知识列表页：筛选、客户端分页、新建知识弹窗
 */
import {
  App as AntdApp,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Space,
  Table,
  Tooltip,
  Typography,
} from 'antd'
import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useCallback, useEffect, useState } from 'react'
import { createKnowledge, listKnowledge } from '../api/knowledge'
import type {
  KnowledgeCreatePayload,
  KnowledgeItem,
  KnowledgeQuery,
} from '../types'

/** 筛选表单值（InputNumber 清空时为 null） */
interface FilterFormValues {
  type?: number | null
  title?: string
}

const DATE_TIME_FORMAT = 'YYYY-MM-DD HH:mm:ss'

function formatDateTime(value: string): string {
  return value ? dayjs(value).format(DATE_TIME_FORMAT) : '-'
}

export default function KnowledgePage() {
  const { message } = AntdApp.useApp()
  const [filterForm] = Form.useForm<FilterFormValues>()
  const [createForm] = Form.useForm<KnowledgeCreatePayload>()

  const [data, setData] = useState<KnowledgeItem[]>([])
  const [loading, setLoading] = useState(false)
  const [appliedQuery, setAppliedQuery] = useState<KnowledgeQuery>({})

  const [modalOpen, setModalOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const fetchList = useCallback(
    async (query: KnowledgeQuery) => {
      setLoading(true)
      try {
        const list = await listKnowledge(query)
        setData(list)
      } catch (err) {
        message.error(err instanceof Error ? err.message : '获取知识列表失败')
      } finally {
        setLoading(false)
      }
    },
    [message],
  )

  useEffect(() => {
    fetchList({})
  }, [fetchList])

  const handleSearch = (values: FilterFormValues) => {
    const query: KnowledgeQuery = {
      type: values.type ?? undefined,
      title: values.title?.trim() || undefined,
    }
    setAppliedQuery(query)
    fetchList(query)
  }

  const handleReset = () => {
    filterForm.resetFields()
    setAppliedQuery({})
    fetchList({})
  }

  const openCreateModal = () => {
    createForm.resetFields()
    setModalOpen(true)
  }

  const handleCreate = async (values: KnowledgeCreatePayload) => {
    setSubmitting(true)
    try {
      await createKnowledge(values)
      message.success('创建成功')
      setModalOpen(false)
      fetchList(appliedQuery)
    } catch (err) {
      message.error(err instanceof Error ? err.message : '创建失败')
    } finally {
      setSubmitting(false)
    }
  }

  const columns: ColumnsType<KnowledgeItem> = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: '类型', dataIndex: 'type', width: 100 },
    { title: '标题', dataIndex: 'title' },
    {
      title: '内容',
      dataIndex: 'content',
      ellipsis: { showTitle: false },
      render: (value: string) => (
        <Tooltip placement="topLeft" title={value}>
          {value}
        </Tooltip>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
  ]

  return (
    <Card>
      <Typography.Title level={4} style={{ marginTop: 0 }}>
        知识列表
      </Typography.Title>

      <Form
        form={filterForm}
        layout="inline"
        style={{ marginBottom: 16, rowGap: 12 }}
        onFinish={handleSearch}
      >
        <Form.Item name="type" label="类型">
          <InputNumber
            min={1}
            precision={0}
            placeholder="请输入类型"
            style={{ width: 140 }}
          />
        </Form.Item>
        <Form.Item name="title" label="标题">
          <Input allowClear placeholder="标题模糊搜索" style={{ width: 200 }} />
        </Form.Item>
        <Form.Item>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              icon={<SearchOutlined />}
            >
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
            <Button
              type="primary"
              ghost
              icon={<PlusOutlined />}
              onClick={openCreateModal}
            >
              新建知识
            </Button>
          </Space>
        </Form.Item>
      </Form>

      <Table<KnowledgeItem>
        rowKey="id"
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={{
          defaultPageSize: 10,
          pageSizeOptions: [10, 20, 50],
          showSizeChanger: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
      />

      <Modal
        title="新建知识"
        open={modalOpen}
        confirmLoading={submitting}
        onOk={() => createForm.submit()}
        onCancel={() => setModalOpen(false)}
        destroyOnClose
        forceRender
      >
        <Form
          form={createForm}
          layout="vertical"
          style={{ marginTop: 16 }}
          onFinish={handleCreate}
        >
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: '请输入类型（正整数）' }]}
          >
            <InputNumber
              min={1}
              precision={0}
              placeholder="请输入正整数"
              style={{ width: '100%' }}
            />
          </Form.Item>
          <Form.Item
            name="title"
            label="标题"
            rules={[{ required: true, message: '请输入标题' }]}
          >
            <Input placeholder="请输入标题" maxLength={255} />
          </Form.Item>
          <Form.Item
            name="content"
            label="内容"
            rules={[{ required: true, message: '请输入内容' }]}
          >
            <Input.TextArea rows={5} placeholder="请输入内容" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}

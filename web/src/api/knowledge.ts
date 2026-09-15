/**
 * 知识条目相关接口封装
 *
 * 注意：knowledge 接口成功时直接返回裸数组/裸对象（无 success 包装）。
 */
import { http } from './client'
import type {
  KnowledgeCreatePayload,
  KnowledgeItem,
  KnowledgeQuery,
} from '../types'

/** 查询知识列表（后端返回全量数组，分页由前端处理） */
export async function listKnowledge(query: KnowledgeQuery): Promise<KnowledgeItem[]> {
  // 仅上送有值的筛选参数
  const params: Record<string, string | number> = {}
  if (query.type !== undefined) {
    params.type = query.type
  }
  if (query.title) {
    params.title = query.title
  }
  const res = await http.get<KnowledgeItem[]>('/knowledge', { params })
  return res.data
}

/** 新建知识（PUT /knowledge，id 由后端生成） */
export async function createKnowledge(
  payload: KnowledgeCreatePayload,
): Promise<KnowledgeItem> {
  const res = await http.put<KnowledgeItem>('/knowledge', payload)
  return res.data
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: string;
}

export interface Account {
  id: number;
  name: string;
  wechat_id: string;
  alias?: string;
  status: string;
  last_task_id?: number | null;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: number;
  account_id: number;
  status: string;
  error?: string;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  account?: Account;
}

export interface TaskLog {
  id: number;
  task_id: number;
  level: string;
  message: string;
  created_at: string;
}

export interface WechatSession {
  id: number;
  session_key: string;
  status: string;
  qr_code: string;
  expires_at?: string | null;
  last_ping?: string | null;
  created_at: string;
}

export interface WechatSearchResult {
  nickname: string;
  alias: string;
  fakeid: string;
  province?: string;
  city?: string;
}

export interface Article {
  id: number;
  account_id: number;
  title: string;
  summary: string;
  content_html: string;
  raw_url: string;
  published_at: string;
  created_at: string;
}

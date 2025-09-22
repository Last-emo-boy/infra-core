export interface User {
  id: number;
  username: string;
  email: string;
  role: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface Service {
  id: number;
  name: string;
  image: string;
  port: number;
  environment: Record<string, string>;
  command?: string[];
  args?: string[];
  status: 'running' | 'stopped' | 'error';
  created_at: string;
  updated_at: string;
}

export interface SystemInfo {
  version: string;
  environment: string;
  uptime: string;
  memory_usage: number;
  cpu_usage: number;
  services_count: number;
}

export interface MetricData {
  timestamp: string;
  value: number;
}

export interface DashboardData {
  services_running: number;
  services_total: number;
  memory_usage: number;
  cpu_usage: number;
  recent_services: Service[];
}

export interface ApiError {
  error: string;
  message?: string;
}
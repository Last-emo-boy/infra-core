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

// SSO and Portal Types
export interface RegisteredService {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  service_url: string;
  callback_url?: string;
  icon?: string;
  category: 'web' | 'api' | 'admin' | 'monitoring' | 'other';
  is_public: boolean;
  required_role: 'user' | 'admin';
  status: 'active' | 'inactive' | 'maintenance';
  health_url?: string;
  last_healthy?: string;
  is_healthy: boolean;
  created_at: string;
  updated_at: string;
}

export interface ServiceCategory {
  name: string;
  display_name: string;
  icon: string;
  services: RegisteredService[];
}

export interface SSOLoginRequest {
  service_name: string;
  redirect_url: string;
}

export interface SSOLoginResponse {
  sso_token: string;
  redirect_url: string;
  expires_at: number;
}

export interface PortalDashboard {
  user: User;
  service_categories: ServiceCategory[];
  recent_services: RegisteredService[];
  service_count: number;
  healthy_services: number;
}

export interface ServiceHealthCheck {
  id: number;
  service_id: string;
  is_healthy: boolean;
  response_time: number;
  error_message?: string;
  checked_at: string;
}

export interface UserServicePermission {
  id: number;
  user_id: number;
  service_id: string;
  can_access: boolean;
  granted_by: number;
  granted_at: string;
  expires_at?: string;
}

export interface ServicePermissionRecord {
  user_id: number;
  username: string;
  email: string;
  role: string;
  can_access: boolean;
  granted_by: number | null;
  granted_by_username?: string | null;
  granted_at?: string | null;
  expires_at?: string | null;
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
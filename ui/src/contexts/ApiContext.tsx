import React, { createContext } from 'react';
import type { ReactNode } from 'react';
import axios from 'axios';
import type { AxiosInstance, AxiosResponse } from 'axios';
import type { 
  User, 
  LoginRequest, 
  LoginResponse, 
  Service, 
  SystemInfo, 
  DashboardData,
  RegisteredService,
  ServiceCategory,
  SSOLoginRequest,
  SSOLoginResponse,
  PortalDashboard,
  ServiceHealthCheck,
  ServicePermissionRecord
} from '../types';

export interface ApiContextType {
  api: AxiosInstance;
  auth: {
    login: (credentials: LoginRequest) => Promise<LoginResponse>;
    register: (userData: { username: string; email: string; password: string }) => Promise<LoginResponse>;
  };
  users: {
    getProfile: () => Promise<User>;
    listUsers: () => Promise<User[]>;
    updateUser: (id: number, data: Partial<User>) => Promise<User>;
    deleteUser: (id: number) => Promise<void>;
  };
  services: {
    list: () => Promise<Service[]>;
    get: (id: number) => Promise<Service>;
    create: (service: Omit<Service, 'id' | 'status' | 'created_at' | 'updated_at'>) => Promise<Service>;
    update: (id: number, service: Partial<Service>) => Promise<Service>;
    delete: (id: number) => Promise<void>;
    start: (id: number) => Promise<void>;
    stop: (id: number) => Promise<void>;
    getLogs: (id: number) => Promise<string>;
  };
  sso: {
    listServices: () => Promise<RegisteredService[]>;
    getUserServices: () => Promise<RegisteredService[]>;
    registerService: (serviceData: Partial<RegisteredService>) => Promise<RegisteredService>;
    updateService: (id: string, serviceData: Partial<RegisteredService>) => Promise<RegisteredService>;
    deleteService: (id: string) => Promise<void>;
    initiateSSO: (request: SSOLoginRequest) => Promise<SSOLoginResponse>;
    validateSSO: (token: string) => Promise<any>;
    getServiceHealth: (id: string) => Promise<ServiceHealthCheck>;
    getServiceHealthHistory: (id: string, limit?: number) => Promise<ServiceHealthCheck[]>;
    grantServiceAccess: (userId: number, serviceId: string) => Promise<void>;
    revokeServiceAccess: (userId: number, serviceId: string) => Promise<void>;
    getServicePermissions: (serviceId: string) => Promise<ServicePermissionRecord[]>;
  };
  portal: {
    getDashboard: () => Promise<PortalDashboard>;
    getServicesByCategory: (category?: string) => Promise<ServiceCategory[]>;
  };
  system: {
    getInfo: () => Promise<SystemInfo>;
    getDashboard: () => Promise<DashboardData>;
    healthCheck: () => Promise<{ status: string }>;
  };
}

export const ApiContext = createContext<ApiContextType | null>(null);

interface ApiProviderProps {
  children: ReactNode;
}

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8082';

export const ApiProvider: React.FC<ApiProviderProps> = ({ children }) => {
  // Create axios instance
  const api = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Add request interceptor to include auth token
  api.interceptors.request.use((config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  });

  // Add response interceptor for error handling
  api.interceptors.response.use(
    (response: AxiosResponse) => response,
    (error) => {
      if (error.response?.status === 401) {
        // Token expired or invalid
        localStorage.removeItem('auth_token');
        window.location.href = '/login';
      }
      return Promise.reject(error);
    }
  );

  const apiMethods: ApiContextType = {
    api,
    auth: {
      login: async (credentials: LoginRequest): Promise<LoginResponse> => {
        const response = await api.post<LoginResponse>('/api/v1/auth/login', credentials);
        return response.data;
      },
      register: async (userData: { username: string; email: string; password: string }): Promise<LoginResponse> => {
        const response = await api.post<LoginResponse>('/api/v1/auth/register', userData);
        return response.data;
      },
    },
    users: {
      getProfile: async (): Promise<User> => {
        const response = await api.get<User>('/api/v1/users/profile');
        return response.data;
      },
      listUsers: async (): Promise<User[]> => {
        const response = await api.get<User[]>('/api/v1/users');
        return response.data;
      },
      updateUser: async (id: number, data: Partial<User>): Promise<User> => {
        const response = await api.put<User>(`/api/v1/users/${id}`, data);
        return response.data;
      },
      deleteUser: async (id: number): Promise<void> => {
        await api.delete(`/api/v1/users/${id}`);
      },
    },
    services: {
      list: async (): Promise<Service[]> => {
        const response = await api.get<Service[]>('/api/v1/services');
        return response.data;
      },
      get: async (id: number): Promise<Service> => {
        const response = await api.get<Service>(`/api/v1/services/${id}`);
        return response.data;
      },
      create: async (service: Omit<Service, 'id' | 'status' | 'created_at' | 'updated_at'>): Promise<Service> => {
        const response = await api.post<Service>('/api/v1/services', service);
        return response.data;
      },
      update: async (id: number, service: Partial<Service>): Promise<Service> => {
        const response = await api.put<Service>(`/api/v1/services/${id}`, service);
        return response.data;
      },
      delete: async (id: number): Promise<void> => {
        await api.delete(`/api/v1/services/${id}`);
      },
      start: async (id: number): Promise<void> => {
        await api.post(`/api/v1/services/${id}/start`);
      },
      stop: async (id: number): Promise<void> => {
        await api.post(`/api/v1/services/${id}/stop`);
      },
      getLogs: async (id: number): Promise<string> => {
        const response = await api.get<string>(`/api/v1/services/${id}/logs`);
        return response.data;
      },
    },
    sso: {
      listServices: async (): Promise<RegisteredService[]> => {
        const response = await api.get<{ services: RegisteredService[] }>('/api/v1/sso/services');
        return response.data.services;
      },
      getUserServices: async (): Promise<RegisteredService[]> => {
        const response = await api.get<{ services: RegisteredService[] }>('/api/v1/sso/user/services');
        return response.data.services;
      },
      registerService: async (serviceData: Partial<RegisteredService>): Promise<RegisteredService> => {
        const response = await api.post<RegisteredService>('/api/v1/sso/services', serviceData);
        return response.data;
      },
      updateService: async (id: string, serviceData: Partial<RegisteredService>): Promise<RegisteredService> => {
        const response = await api.put<RegisteredService>(`/api/v1/sso/services/${id}`, serviceData);
        return response.data;
      },
      deleteService: async (id: string): Promise<void> => {
        await api.delete(`/api/v1/sso/services/${id}`);
      },
      initiateSSO: async (request: SSOLoginRequest): Promise<SSOLoginResponse> => {
        const response = await api.post<SSOLoginResponse>('/api/v1/sso/login', request);
        return response.data;
      },
      validateSSO: async (token: string): Promise<any> => {
        const response = await api.get(`/api/v1/sso/validate?token=${token}`);
        return response.data;
      },
      getServiceHealth: async (id: string): Promise<ServiceHealthCheck> => {
        const response = await api.get<ServiceHealthCheck>(`/api/v1/sso/services/${id}/health`);
        return response.data;
      },
      getServiceHealthHistory: async (id: string, limit = 50): Promise<ServiceHealthCheck[]> => {
        const response = await api.get<{ checks: ServiceHealthCheck[] }>(`/api/v1/sso/services/${id}/health/history?limit=${limit}`);
        return response.data.checks;
      },
      grantServiceAccess: async (userId: number, serviceId: string): Promise<void> => {
        await api.post(`/api/v1/sso/permissions/${userId}/${serviceId}/grant`);
      },
      revokeServiceAccess: async (userId: number, serviceId: string): Promise<void> => {
        await api.post(`/api/v1/sso/permissions/${userId}/${serviceId}/revoke`);
      },
      getServicePermissions: async (serviceId: string): Promise<ServicePermissionRecord[]> => {
        const response = await api.get<{ permissions: ServicePermissionRecord[] }>(`/api/v1/sso/services/${serviceId}/permissions`);
        return response.data.permissions;
      },
    },
    portal: {
      getDashboard: async (): Promise<PortalDashboard> => {
        const response = await api.get<PortalDashboard>('/api/v1/portal/dashboard');
        return response.data;
      },
      getServicesByCategory: async (category?: string): Promise<ServiceCategory[]> => {
        const url = category ? `/api/v1/portal/services?category=${category}` : '/api/v1/portal/services';
        const response = await api.get<ServiceCategory[]>(url);
        return response.data;
      },
    },
    system: {
      getInfo: async (): Promise<SystemInfo> => {
        const response = await api.get<SystemInfo>('/api/v1/system/info');
        return response.data;
      },
      getDashboard: async (): Promise<DashboardData> => {
        const response = await api.get<DashboardData>('/api/v1/system/dashboard');
        return response.data;
      },
      healthCheck: async (): Promise<{ status: string }> => {
        const response = await api.get<{ status: string }>('/api/v1/health');
        return response.data;
      },
    },
  };

  return <ApiContext.Provider value={apiMethods}>{children}</ApiContext.Provider>;
};
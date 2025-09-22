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
  DashboardData
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
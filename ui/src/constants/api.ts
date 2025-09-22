// API endpoints and configuration
export const API_ENDPOINTS = {
  AUTH: {
    LOGIN: '/api/v1/auth/login',
    REGISTER: '/api/v1/auth/register',
    PROFILE: '/api/v1/users/profile',
  },
  USERS: {
    LIST: '/api/v1/users',
    GET: (id: number) => `/api/v1/users/${id}`,
    UPDATE: (id: number) => `/api/v1/users/${id}`,
    DELETE: (id: number) => `/api/v1/users/${id}`,
  },
  SERVICES: {
    LIST: '/api/v1/services',
    GET: (id: number) => `/api/v1/services/${id}`,
    CREATE: '/api/v1/services',
    UPDATE: (id: number) => `/api/v1/services/${id}`,
    DELETE: (id: number) => `/api/v1/services/${id}`,
    START: (id: number) => `/api/v1/services/${id}/start`,
    STOP: (id: number) => `/api/v1/services/${id}/stop`,
    LOGS: (id: number) => `/api/v1/services/${id}/logs`,
  },
  SYSTEM: {
    INFO: '/api/v1/system/info',
    METRICS: '/api/v1/system/metrics',
    DASHBOARD: '/api/v1/system/dashboard',
    HEALTH: '/api/v1/health',
  },
} as const;

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8082';
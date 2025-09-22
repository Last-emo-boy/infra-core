// Auth constants
export const TOKEN_STORAGE_KEY = 'infra-core-token';
export const USER_STORAGE_KEY = 'infra-core-user';

// Auth configuration
export const AUTH_CONFIG = {
  TOKEN_HEADER: 'Authorization',
  TOKEN_PREFIX: 'Bearer ',
  LOGIN_REDIRECT: '/dashboard',
  LOGOUT_REDIRECT: '/login',
} as const;
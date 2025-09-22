import { useContext } from 'react';
import { ApiContext, type ApiContextType } from '../contexts/ApiContext';

export const useApi = (): ApiContextType => {
  const context = useContext(ApiContext);
  if (!context) {
    throw new Error('useApi must be used within an ApiProvider');
  }
  return context;
};
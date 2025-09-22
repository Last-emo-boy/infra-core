import React, { useState, useEffect, useCallback } from 'react';
import { Activity, Server, Clock, MemoryStick } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import type { SystemInfo } from '../types';

const System: React.FC = () => {
  const api = useApi();
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadSystemInfo = useCallback(async () => {
    try {
      setLoading(true);
      const info = await api.system.getInfo();
      setSystemInfo(info);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Failed to load system information');
    } finally {
      setLoading(false);
    }
  }, [api.system]);

  useEffect(() => {
    loadSystemInfo();
  }, [loadSystemInfo]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <div className="text-sm text-red-700">{error}</div>
      </div>
    );
  }

  const infoItems = [
    {
      label: 'Version',
      value: systemInfo?.version || 'Unknown',
      icon: Server,
    },
    {
      label: 'Environment',
      value: systemInfo?.environment || 'Unknown',
      icon: Activity,
    },
    {
      label: 'Uptime',
      value: systemInfo?.uptime || 'Unknown',
      icon: Clock,
    },
    {
      label: 'Services Count',
      value: systemInfo?.services_count || 0,
      icon: Server,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">System</h1>
        <p className="mt-1 text-sm text-gray-600">
          System information and monitoring
        </p>
      </div>

      {/* System Info Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {infoItems.map((item) => (
          <div key={item.label} className="bg-white overflow-hidden shadow rounded-lg">
            <div className="p-5">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <item.icon className="h-6 w-6 text-gray-400" />
                </div>
                <div className="ml-5 w-0 flex-1">
                  <dl>
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      {item.label}
                    </dt>
                    <dd className="text-lg font-medium text-gray-900">
                      {item.value}
                    </dd>
                  </dl>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Resource Usage */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">CPU Usage</h3>
          <div className="flex items-center">
            <div className="flex-1">
              <div className="bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-blue-600 h-2 rounded-full"
                  style={{ width: `${Math.round((systemInfo?.cpu_usage || 0) * 100)}%` }}
                ></div>
              </div>
            </div>
            <div className="ml-4 text-sm font-medium text-gray-900">
              {Math.round((systemInfo?.cpu_usage || 0) * 100)}%
            </div>
          </div>
        </div>

        <div className="bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Memory Usage</h3>
          <div className="flex items-center">
            <div className="flex-1">
              <div className="bg-gray-200 rounded-full h-2">
                <div 
                  className="bg-purple-600 h-2 rounded-full"
                  style={{ width: `${Math.round((systemInfo?.memory_usage || 0) * 100)}%` }}
                ></div>
              </div>
            </div>
            <div className="ml-4 text-sm font-medium text-gray-900">
              {Math.round((systemInfo?.memory_usage || 0) * 100)}%
            </div>
          </div>
        </div>
      </div>

      {/* Health Check */}
      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Health Status
          </h3>
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <div className="h-4 w-4 bg-green-400 rounded-full"></div>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">System is healthy</p>
              <p className="text-sm text-gray-500">All services are operating normally</p>
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            System Actions
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <button 
              onClick={loadSystemInfo}
              className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
            >
              <Activity className="h-4 w-4 mr-2" />
              Refresh Status
            </button>
            <button className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500">
              <MemoryStick className="h-4 w-4 mr-2" />
              View Logs
            </button>
            <button className="inline-flex items-center px-4 py-2 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500">
              <Server className="h-4 w-4 mr-2" />
              System Settings
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default System;
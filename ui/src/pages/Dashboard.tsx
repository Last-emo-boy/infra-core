import React, { useState, useEffect, useCallback } from 'react';
import { Server, Activity, Cpu, HardDrive } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import type { DashboardData } from '../types';

const Dashboard: React.FC = () => {
  const api = useApi();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const loadDashboard = useCallback(async () => {
    try {
      setLoading(true);
      const dashboardData = await api.system.getDashboard();
      setData(dashboardData);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Failed to load dashboard');
    } finally {
      setLoading(false);
    }
  }, [api.system]);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

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

  const stats = [
    {
      name: 'Running Services',
      value: data?.services_running || 0,
      total: data?.services_total || 0,
      icon: Server,
      color: 'text-green-600',
      bgColor: 'bg-green-100',
    },
    {
      name: 'CPU Usage',
      value: Math.round((data?.cpu_usage || 0) * 100),
      unit: '%',
      icon: Cpu,
      color: 'text-blue-600',
      bgColor: 'bg-blue-100',
    },
    {
      name: 'Memory Usage',
      value: Math.round((data?.memory_usage || 0) * 100),
      unit: '%',
      icon: HardDrive,
      color: 'text-purple-600',
      bgColor: 'bg-purple-100',
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="mt-1 text-sm text-gray-600">
          Overview of your infrastructure
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {stats.map((stat) => (
          <div key={stat.name} className="bg-white overflow-hidden shadow rounded-lg">
            <div className="p-5">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <div className={`p-3 rounded-md ${stat.bgColor}`}>
                    <stat.icon className={`h-6 w-6 ${stat.color}`} />
                  </div>
                </div>
                <div className="ml-5 w-0 flex-1">
                  <dl>
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      {stat.name}
                    </dt>
                    <dd className="text-lg font-medium text-gray-900">
                      {stat.total ? `${stat.value}/${stat.total}` : `${stat.value}${stat.unit || ''}`}
                    </dd>
                  </dl>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Services */}
      {data?.recent_services && data.recent_services.length > 0 && (
        <div className="bg-white shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
              Recent Services
            </h3>
            <div className="space-y-3">
              {data.recent_services.map((service) => (
                <div key={service.id} className="flex items-center justify-between py-2 border-b border-gray-200 last:border-b-0">
                  <div className="flex items-center">
                    <div className="flex-shrink-0">
                      <div className={`h-3 w-3 rounded-full ${
                        service.status === 'running' ? 'bg-green-400' :
                        service.status === 'stopped' ? 'bg-gray-400' :
                        'bg-red-400'
                      }`}></div>
                    </div>
                    <div className="ml-3">
                      <p className="text-sm font-medium text-gray-900">{service.name}</p>
                      <p className="text-sm text-gray-500">{service.image}</p>
                    </div>
                  </div>
                  <div className="text-sm text-gray-500">
                    Port {service.port}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Quick Actions
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button className="p-4 border border-gray-300 rounded-lg hover:bg-gray-50 text-left">
              <Activity className="h-6 w-6 text-primary-600 mb-2" />
              <h4 className="font-medium text-gray-900">View System Logs</h4>
              <p className="text-sm text-gray-500">Check system health and logs</p>
            </button>
            <button className="p-4 border border-gray-300 rounded-lg hover:bg-gray-50 text-left">
              <Server className="h-6 w-6 text-primary-600 mb-2" />
              <h4 className="font-medium text-gray-900">Manage Services</h4>
              <p className="text-sm text-gray-500">Start, stop, and configure services</p>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  AlertCircle,
  ExternalLink,
  Globe2,
  Lock,
  RefreshCcw,
  Search,
  ShieldCheck,
  Users
} from 'lucide-react';
import { useApi } from '../hooks/useApi';
import { useAuth } from '../hooks/useAuth';
import type { RegisteredService } from '../types';

type CategoryKey = RegisteredService['category'];

const CATEGORY_META: Record<CategoryKey, { label: string; icon: React.ComponentType<{ className?: string }> }> = {
  web: { label: 'Web Applications', icon: Globe2 },
  api: { label: 'Service APIs', icon: Activity },
  admin: { label: 'Administration', icon: ShieldCheck },
  monitoring: { label: 'Monitoring & Ops', icon: Users },
  other: { label: 'Other Services', icon: Globe2 },
};

interface ApiErrorShape {
  response?: {
    data?: {
      error?: string;
      message?: string;
    };
  };
  message?: string;
}

const Portal: React.FC = () => {
  const api = useApi();
  const { user } = useAuth();

  const [services, setServices] = useState<RegisteredService[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [launchingServiceId, setLaunchingServiceId] = useState<string | null>(null);

  const mapError = useCallback((err: unknown): string => {
    const parsed = err as ApiErrorShape;
    return (
      parsed.response?.data?.error ||
      parsed.response?.data?.message ||
      parsed.message ||
      'Unable to process your request right now.'
    );
  }, []);

  const loadServices = useCallback(async () => {
    try {
      setError('');
      setRefreshing(true);
      const data = await api.sso.getUserServices();
      setServices(data);
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [api.sso, mapError]);

  useEffect(() => {
    loadServices();
  }, [loadServices]);

  const filteredServices = useMemo<RegisteredService[]>(() => {
    if (!search.trim()) {
      return services;
    }

    const keywords = search.toLowerCase();
    return services.filter((service: RegisteredService) => {
      const haystack = [
        service.display_name,
        service.name,
        service.description ?? '',
        service.category,
      ]
        .join(' ')
        .toLowerCase();

      return haystack.includes(keywords);
    });
  }, [services, search]);

  const groupedServices = useMemo<Record<CategoryKey, RegisteredService[]>>(() => {
    return filteredServices.reduce<Record<CategoryKey, RegisteredService[]>>(
      (acc: Record<CategoryKey, RegisteredService[]>, service: RegisteredService) => {
        const categoryKey: CategoryKey = service.category;

        acc[categoryKey] = acc[categoryKey] ?? [];
        acc[categoryKey].push(service);

        return acc;
      },
      {
        web: [],
        admin: [],
        monitoring: [],
        api: [],
        other: [],
      }
    );
  }, [filteredServices]);

  const categoryOrder: CategoryKey[] = ['web', 'admin', 'monitoring', 'api', 'other'];

  const formatStatusBadge = (service: RegisteredService): string => {
    if (service.status === 'maintenance') {
      return 'bg-yellow-100 text-yellow-700';
    }
    if (service.status === 'inactive') {
      return 'bg-gray-100 text-gray-600';
    }
    return 'bg-green-100 text-green-700';
  };

  const handleLaunch = async (service: RegisteredService) => {
    try {
      setError('');
      setLaunchingServiceId(service.id);

      const redirectTarget = (service.callback_url && service.callback_url.trim()) || service.service_url;
      const response = await api.sso.initiateSSO({
        service_name: service.name,
        redirect_url: redirectTarget,
      });

      window.open(response.redirect_url, '_blank', 'noopener');
    } catch (err) {
      setError(mapError(err));
    } finally {
      setLaunchingServiceId(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-500 mx-auto mb-4" />
          <p className="text-sm text-gray-500">Loading your portal…</p>
        </div>
      </div>
    );
  }

  const hasServices = filteredServices.length > 0;

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Unified Service Portal</h1>
          <p className="mt-1 text-sm text-gray-600">
            Welcome back {user?.username}. Access the services you&apos;re provisioned for in a single place.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={loadServices}
            disabled={refreshing}
            className="inline-flex items-center px-4 py-2 rounded-md text-sm font-medium border border-transparent text-white bg-primary-600 hover:bg-primary-700 disabled:opacity-60"
          >
            <RefreshCcw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white shadow rounded-lg p-4">
          <div className="flex items-center">
            <ShieldCheck className="h-10 w-10 text-primary-500 mr-3" />
            <div>
              <p className="text-sm text-gray-500">Accessible Services</p>
              <p className="text-2xl font-semibold text-gray-900">{services.length}</p>
            </div>
          </div>
        </div>
        <div className="bg-white shadow rounded-lg p-4">
          <div className="flex items-center">
            <Activity className="h-10 w-10 text-green-500 mr-3" />
            <div>
              <p className="text-sm text-gray-500">Healthy Today</p>
              <p className="text-2xl font-semibold text-gray-900">
                {services.filter((service) => service.is_healthy).length}
              </p>
            </div>
          </div>
        </div>
        <div className="bg-white shadow rounded-lg p-4">
          <div className="flex items-center">
            <Lock className="h-10 w-10 text-indigo-500 mr-3" />
            <div>
              <p className="text-sm text-gray-500">Role</p>
              <p className="text-xl font-semibold text-gray-900 capitalize">{user?.role ?? 'user'}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-white shadow rounded-lg p-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div className="relative w-full sm:w-72">
            <Search className="h-4 w-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="search"
              className="w-full pl-9 pr-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
              placeholder="Search services, teams, categories…"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
            />
          </div>
          <p className="text-sm text-gray-500">
            Showing <span className="font-medium text-gray-900">{filteredServices.length}</span> of{' '}
            <span className="font-medium text-gray-900">{services.length}</span> services
          </p>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-100 text-red-700 rounded-md p-4 flex items-start gap-3">
          <AlertCircle className="h-5 w-5 mt-0.5" />
          <div>
            <p className="font-medium">We ran into an issue</p>
            <p className="text-sm">{error}</p>
          </div>
        </div>
      )}

      {!hasServices ? (
        <div className="bg-white shadow rounded-lg p-10 text-center">
          <ShieldCheck className="h-10 w-10 text-gray-300 mx-auto mb-4" />
          <h2 className="text-lg font-semibold text-gray-900">No services yet</h2>
          <p className="text-sm text-gray-500 mt-2">
            Once your administrator provisions access, services will appear here automatically.
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {categoryOrder.map((categoryKey) => {
            const bucket = groupedServices[categoryKey];
            if (!bucket || bucket.length === 0) {
              return null;
            }

            const meta = CATEGORY_META[categoryKey] ?? CATEGORY_META.other;
            const Icon = meta.icon;

            return (
              <section key={categoryKey} className="space-y-4">
                <div className="flex items-center gap-2">
                  <Icon className="h-5 w-5 text-primary-500" />
                  <h2 className="text-lg font-semibold text-gray-900">{meta.label}</h2>
                  <span className="text-xs text-gray-500">({bucket.length})</span>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                  {bucket.map((service) => (
                    <article key={service.id} className="bg-white border border-gray-100 rounded-lg shadow-sm hover:border-primary-200 transition-colors">
                      <div className="p-4 space-y-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex items-start gap-3">
                            {service.icon ? (
                              <img
                                src={service.icon}
                                alt={`${service.display_name} icon`}
                                className="h-10 w-10 rounded-md object-cover border border-gray-200"
                                loading="lazy"
                              />
                            ) : (
                              <div className="h-10 w-10 rounded-md bg-primary-100 text-primary-600 flex items-center justify-center text-sm font-semibold uppercase">
                                {service.display_name.trim().slice(0, 2).toUpperCase()}
                              </div>
                            )}
                            <div>
                              <h3 className="text-lg font-semibold text-gray-900">{service.display_name}</h3>
                              <p className="text-sm text-gray-500">{service.description ?? 'No description provided.'}</p>
                            </div>
                          </div>
                          <span className={`text-xs font-medium px-2 py-1 rounded-full ${formatStatusBadge(service)}`}>
                            {service.status}
                          </span>
                        </div>

                        <div className="flex items-center justify-between text-sm text-gray-500">
                          <div className="flex items-center gap-2">
                            <span className={`h-2 w-2 rounded-full ${service.is_healthy ? 'bg-green-500' : 'bg-red-500'}`} />
                            <span>{service.is_healthy ? 'Healthy' : 'Unavailable'}</span>
                          </div>
                          <div className="text-xs text-gray-400">
                            {service.last_healthy ? `Checked ${new Date(service.last_healthy).toLocaleString()}` : 'No health data'}
                          </div>
                        </div>

                        <div className="bg-gray-50 rounded-md p-3 text-xs text-gray-600 space-y-1">
                          <div className="flex justify-between">
                            <span className="uppercase text-gray-400 tracking-wide">Access</span>
                            <span className="font-medium text-gray-700">{service.is_public ? 'Public' : service.required_role}</span>
                          </div>
                          <div className="flex justify-between">
                            <span className="uppercase text-gray-400 tracking-wide">URL</span>
                            <a
                              href={service.service_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-primary-600 hover:text-primary-700"
                            >
                              {service.service_url}
                            </a>
                          </div>
                        </div>

                        <button
                          onClick={() => handleLaunch(service)}
                          disabled={service.status !== 'active' || launchingServiceId === service.id}
                          className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-md bg-primary-600 hover:bg-primary-700 text-white disabled:opacity-60 disabled:cursor-not-allowed"
                        >
                          <ExternalLink className={`h-4 w-4 ${launchingServiceId === service.id ? 'animate-pulse' : ''}`} />
                          {launchingServiceId === service.id ? 'Opening…' : 'Launch with SSO'}
                        </button>
                      </div>
                    </article>
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default Portal;

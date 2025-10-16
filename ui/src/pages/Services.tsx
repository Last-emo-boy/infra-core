import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Play,
  Square,
  Plus,
  Edit,
  Trash2,
  Server,
  CheckCircle2,
  PauseCircle,
  AlertTriangle,
  Filter,
  Search,
  LayoutGrid,
  ListChecks,
  RefreshCcw,
  Clock,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useApi } from '../hooks/useApi';
import type { Service, ServiceSummary } from '../types';

type SummaryTone = 'primary' | 'success' | 'muted' | 'danger';
type StatusFilter = 'all' | 'running' | 'stopped' | 'error';
type ViewMode = 'list' | 'grid';

interface SummaryCardProps {
  icon: LucideIcon;
  title: string;
  value: number;
  tone: SummaryTone;
  description?: string;
}

const summaryToneStyles: Record<SummaryTone, { icon: string; badge: string }> = {
  primary: {
    icon: 'text-primary-600',
    badge: 'bg-primary-50 text-primary-600',
  },
  success: {
    icon: 'text-green-600',
    badge: 'bg-green-50 text-green-600',
  },
  muted: {
    icon: 'text-slate-600',
    badge: 'bg-slate-100 text-slate-700',
  },
  danger: {
    icon: 'text-red-600',
    badge: 'bg-red-50 text-red-600',
  },
};

const statusToneBadge: Record<Service['status'], string> = {
  running: 'bg-green-100 text-green-800',
  stopped: 'bg-gray-100 text-gray-800',
  error: 'bg-red-100 text-red-800',
};

const SummaryCard = ({ icon: Icon, title, value, tone, description }: SummaryCardProps) => (
  <div className="bg-white p-4 shadow-sm rounded-lg border border-gray-100 space-y-2">
    <div className="flex items-center justify-between">
      <div>
        <p className="text-sm text-gray-500">{title}</p>
        <p className="mt-2 text-3xl font-semibold text-gray-900">{value}</p>
      </div>
      <div className={`p-2 rounded-full ${summaryToneStyles[tone].badge}`}>
        <Icon className={`h-5 w-5 ${summaryToneStyles[tone].icon}`} />
      </div>
    </div>
    {description ? <p className="text-xs text-gray-500">{description}</p> : null}
  </div>
);

const Services: React.FC = () => {
  const { services: servicesApi } = useApi();

  const [services, setServices] = useState<Service[]>([]);
  const [summary, setSummary] = useState<ServiceSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [isRefreshing, setIsRefreshing] = useState(false);

  const formatDate = useCallback((value?: string) => {
    if (!value) {
      return '—';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '—';
    }
    return date.toLocaleString();
  }, []);

  const getStatusColor = useCallback((status: Service['status']) => statusToneBadge[status], []);

  const loadData = useCallback(
    async (withLoading = false) => {
      if (withLoading) {
        setLoading(true);
      }
      setError(null);

      try {
        const [servicesResponse, summaryResponse] = await Promise.all([
          servicesApi.list(),
          servicesApi.summary(),
        ]);

        setServices(servicesResponse);
        setSummary(summaryResponse);
      } catch (error) {
        console.error('Failed to load services data', error);
        setError('Unable to load services right now. Please try again shortly.');
      } finally {
        if (withLoading) {
          setLoading(false);
        }
      }
    },
    [servicesApi]
  );

  useEffect(() => {
    loadData(true);
  }, [loadData]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    try {
      await loadData(false);
    } finally {
      setIsRefreshing(false);
    }
  }, [loadData]);

  const handleStartService = useCallback(
    async (serviceId: string) => {
      try {
        await servicesApi.start(serviceId);
        setServices((previous) =>
          previous.map((service) =>
            service.id === serviceId ? { ...service, status: 'running' } : service
          )
        );
        await loadData(false);
      } catch (error) {
        console.error('Failed to start service', error);
        setError('Failed to start the service.');
      }
    },
    [servicesApi, loadData]
  );

  const handleStopService = useCallback(
    async (serviceId: string) => {
      try {
        await servicesApi.stop(serviceId);
        setServices((previous) =>
          previous.map((service) =>
            service.id === serviceId ? { ...service, status: 'stopped' } : service
          )
        );
        await loadData(false);
      } catch (error) {
        console.error('Failed to stop service', error);
        setError('Failed to stop the service.');
      }
    },
    [servicesApi, loadData]
  );

  const handleDeleteService = useCallback(
    async (serviceId: string) => {
      try {
        await servicesApi.delete(serviceId);
        setServices((previous) => previous.filter((service) => service.id !== serviceId));
        await loadData(false);
      } catch (error) {
        console.error('Failed to delete service', error);
        setError('Failed to delete the service.');
      }
    },
    [servicesApi, loadData]
  );

  const filteredServices = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();

    return services.filter((service) => {
      const matchesStatus = statusFilter === 'all' || service.status === statusFilter;
      const matchesSearch = normalizedQuery
        ? [service.name, service.image, String(service.port), service.id]
            .filter(Boolean)
            .some((field) => field.toLowerCase().includes(normalizedQuery))
        : true;

      return matchesStatus && matchesSearch;
    });
  }, [services, searchQuery, statusFilter]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Services</h1>
          <p className="mt-1 text-sm text-gray-600">Manage your containerized workloads</p>
        </div>
        <div className="flex flex-col sm:flex-row gap-3 sm:items-center">
          <div className="flex rounded-lg border border-gray-200 overflow-hidden">
            <button
              type="button"
              onClick={() => setViewMode('list')}
              className={`inline-flex items-center px-3 py-2 text-sm font-medium transition ${
                viewMode === 'list'
                  ? 'bg-primary-600 text-white'
                  : 'bg-white text-gray-600 hover:bg-gray-50'
              }`}
            >
              <ListChecks className="mr-2 h-4 w-4" />
              List
            </button>
            <button
              type="button"
              onClick={() => setViewMode('grid')}
              className={`inline-flex items-center px-3 py-2 text-sm font-medium transition ${
                viewMode === 'grid'
                  ? 'bg-primary-600 text-white'
                  : 'bg-white text-gray-600 hover:bg-gray-50'
              }`}
            >
              <LayoutGrid className="mr-2 h-4 w-4" />
              Grid
            </button>
          </div>
          <button
            type="button"
            onClick={handleRefresh}
            className="inline-flex items-center px-4 py-2 border border-gray-200 text-sm font-medium rounded-md shadow-sm bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          >
            <RefreshCcw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
            Refresh
          </button>
          <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500">
            <Plus className="h-4 w-4 mr-2" />
            Add Service
          </button>
        </div>
      </div>

      {error ? (
        <div className="rounded-md bg-red-50 p-4">
          <div className="text-sm text-red-700">{error}</div>
        </div>
      ) : null}

      <div className="bg-white border border-gray-200 shadow-sm rounded-lg p-4 space-y-4">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div className="flex items-center gap-3 text-sm text-gray-500">
            <Filter className="h-4 w-4" />
            <span>Quick filters to find services by status or keyword.</span>
          </div>
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1 min-w-[240px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                type="search"
                value={searchQuery}
                onChange={(event) => setSearchQuery(event.target.value)}
                placeholder="Search by name, image, or port..."
                className="w-full rounded-md border border-gray-200 pl-10 pr-3 py-2 text-sm focus:border-primary-500 focus:ring-primary-500"
              />
            </div>
            <div className="flex items-center gap-2 overflow-x-auto sm:overflow-visible">
              {[
                { label: 'All', value: 'all' as StatusFilter, tone: 'primary' as SummaryTone },
                { label: 'Running', value: 'running' as StatusFilter, tone: 'success' as SummaryTone },
                { label: 'Stopped', value: 'stopped' as StatusFilter, tone: 'muted' as SummaryTone },
                { label: 'Error', value: 'error' as StatusFilter, tone: 'danger' as SummaryTone },
              ].map((filterOption) => (
                <button
                  key={filterOption.value}
                  type="button"
                  onClick={() => setStatusFilter(filterOption.value)}
                  className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium transition ${
                    statusFilter === filterOption.value
                      ? `${
                          filterOption.tone === 'primary'
                            ? 'bg-primary-600 text-white'
                            : filterOption.tone === 'success'
                            ? 'bg-green-600 text-white'
                            : filterOption.tone === 'danger'
                            ? 'bg-red-600 text-white'
                            : 'bg-gray-800 text-white'
                        }`
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  {filterOption.label}
                </button>
              ))}
            </div>
          </div>
        </div>
        {summary?.last_updated ? (
          <p className="text-xs text-gray-500 flex items-center">
            <Clock className="mr-1 h-4 w-4" />
            Updated {formatDate(summary.last_updated)}
          </p>
        ) : null}
      </div>

      {summary ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard
            icon={Server}
            title="Total Services"
            value={summary.counts.total}
            tone="primary"
            description="All registered workloads"
          />
          <SummaryCard
            icon={CheckCircle2}
            title="Running"
            value={summary.counts.running}
            tone="success"
            description="Healthy instances right now"
          />
          <SummaryCard
            icon={PauseCircle}
            title="Stopped"
            value={summary.counts.stopped}
            tone="muted"
            description="Manually or automatically paused"
          />
          <SummaryCard
            icon={AlertTriangle}
            title="Error"
            value={summary.counts.error}
            tone="danger"
            description="Require attention"
          />
        </div>
      ) : null}

      {summary?.recent?.length ? (
        <div className="bg-white shadow overflow-hidden sm:rounded-lg">
          <div className="px-4 py-4 border-b border-gray-200">
            <h2 className="text-lg font-semibold text-gray-900">Recently Updated</h2>
            <p className="text-sm text-gray-500">Latest services that changed state</p>
          </div>
          <ul className="divide-y divide-gray-200">
            {summary.recent.map((service) => (
              <li key={`recent-${service.id}`} className="px-4 py-3 flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-900">{service.name}</p>
                  <p className="text-xs text-gray-500">{service.image}</p>
                </div>
                <div className="text-right">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                      service.status
                    )}`}
                  >
                    {service.status}
                  </span>
                  <p className="text-xs text-gray-400 mt-1">
                    {formatDate(service.updated_at || service.created_at)}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {filteredServices.length === 0 ? (
        <div className="text-center py-12">
          <div className="w-12 h-12 mx-auto bg-gray-100 rounded-full flex items-center justify-center mb-4">
            <Play className="h-6 w-6 text-gray-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">No services match your filters</h3>
          <p className="text-gray-500 mb-4">Try adjusting the status filter or search term.</p>
          <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700">
            <Plus className="h-4 w-4 mr-2" />
            Add Service
          </button>
        </div>
      ) : (
        <div
          className={
            viewMode === 'grid'
              ? 'grid gap-4 sm:grid-cols-2 xl:grid-cols-3'
              : 'bg-white shadow overflow-hidden sm:rounded-lg'
          }
        >
          {filteredServices.map((service) => (
            <ServiceRow
              key={service.id}
              service={service}
              viewMode={viewMode}
              getStatusColor={getStatusColor}
              formatDate={formatDate}
              onStart={handleStartService}
              onStop={handleStopService}
              onDelete={handleDeleteService}
            />
          ))}
        </div>
      )}
    </div>
  );
};

interface ServiceRowProps {
  service: Service;
  viewMode: ViewMode;
  getStatusColor: (status: Service['status']) => string;
  formatDate: (value?: string) => string;
  onStart: (id: string) => Promise<void>;
  onStop: (id: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

const ServiceRow = ({
  service,
  viewMode,
  getStatusColor,
  formatDate,
  onStart,
  onStop,
  onDelete,
}: ServiceRowProps) => {
  const statusDotClass =
    service.status === 'running'
      ? 'bg-green-400'
      : service.status === 'stopped'
      ? 'bg-gray-400'
      : 'bg-red-400';

  const command = (service.command ?? []).join(' ');
  const args = (service.args ?? []).join(' ');

  const body = (
    <>
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <span className={`h-3 w-3 rounded-full ${statusDotClass}`} aria-hidden />
          <div>
            <div className="flex items-center gap-3">
              <h3 className="text-lg font-semibold text-gray-900">{service.name}</h3>
              <span
                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
                  service.status
                )}`}
              >
                {service.status}
              </span>
            </div>
            <p className="mt-1 text-sm text-gray-500">
              Image: <span className="font-medium text-gray-700">{service.image}</span> • Port: {service.port}
            </p>
          </div>
        </div>
        <div className="space-x-2 flex-shrink-0">
          {service.status === 'running' ? (
            <button
              onClick={() => onStop(service.id)}
              className="inline-flex items-center p-2 border border-transparent rounded-full shadow-sm text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
            >
              <Square className="h-4 w-4" />
            </button>
          ) : (
            <button
              onClick={() => onStart(service.id)}
              className="inline-flex items-center p-2 border border-transparent rounded-full shadow-sm text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500"
            >
              <Play className="h-4 w-4" />
            </button>
          )}

          <button className="inline-flex items-center p-2 border border-gray-300 rounded-full shadow-sm text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500">
            <Edit className="h-4 w-4" />
          </button>

          <button
            onClick={() => onDelete(service.id)}
            className="inline-flex items-center p-2 border border-gray-300 rounded-full shadow-sm text-red-700 bg-white hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>

      <div className="mt-4 grid gap-3 text-sm text-gray-500 md:grid-cols-2">
        <div>
          <p className="font-medium text-gray-700">Metadata</p>
          <p>Created: {formatDate(service.created_at)}</p>
          <p>Updated: {formatDate(service.updated_at || service.created_at)}</p>
          <p>Replicas: {service.replicas ?? 1}</p>
        </div>
        <div className="space-y-1">
          {command ? (
            <p>
              <span className="font-medium text-gray-700">Command:</span> {command}
            </p>
          ) : null}
          {args ? (
            <p>
              <span className="font-medium text-gray-700">Args:</span> {args}
            </p>
          ) : null}
          {service.environment && Object.keys(service.environment).length > 0 ? (
            <div>
              <p className="font-medium text-gray-700">Environment</p>
              <div className="mt-1 flex flex-wrap gap-1">
                {Object.entries(service.environment).map(([key, value]) => (
                  <span
                    key={key}
                    className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600"
                  >
                    {key}: {value}
                  </span>
                ))}
              </div>
            </div>
          ) : (
            <p className="text-xs text-gray-400">No environment variables configured.</p>
          )}
        </div>
      </div>
    </>
  );

  if (viewMode === 'grid') {
    return (
      <div className="bg-white border border-gray-200 rounded-lg p-4 shadow-sm hover:shadow-md transition-shadow">
        {body}
      </div>
    );
  }

  return <div className="px-4 py-4 border-b last:border-b-0 border-gray-200">{body}</div>;
};

export default Services;
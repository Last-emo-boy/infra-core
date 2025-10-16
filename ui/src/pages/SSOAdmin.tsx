import React, { useCallback, useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Edit3,
  Loader2,
  Plus,
  RefreshCcw,
  Shield,
  Search,
  Trash2,
  TrendingUp,
  Users,
  X
} from 'lucide-react';
import { useApi } from '../hooks/useApi';
import type { RegisteredService, ServiceHealthCheck, ServicePermissionRecord } from '../types';

interface ApiErrorShape {
  response?: {
    data?: {
      error?: string;
      message?: string;
    };
  };
  message?: string;
}

interface FormState {
  name: string;
  display_name: string;
  description: string;
  service_url: string;
  callback_url: string;
  icon: string;
  category: RegisteredService['category'];
  is_public: boolean;
  required_role: RegisteredService['required_role'];
  health_url: string;
}

const emptyForm: FormState = {
  name: '',
  display_name: '',
  description: '',
  service_url: '',
  callback_url: '',
  icon: '',
  category: 'web',
  is_public: true,
  required_role: 'user',
  health_url: '',
};

const statusBadge = (status: RegisteredService['status']): string => {
  switch (status) {
    case 'inactive':
      return 'bg-gray-100 text-gray-600';
    case 'maintenance':
      return 'bg-yellow-100 text-yellow-700';
    default:
      return 'bg-green-100 text-green-700';
  }
};

const healthBadge = (isHealthy: boolean): string =>
  isHealthy ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-600';

const requiredRoleLabel = (service: RegisteredService): string => {
  if (service.is_public) {
    return 'Public';
  }
  return service.required_role === 'admin' ? 'Admin only' : 'Standard';
};

const SSOAdmin: React.FC = () => {
  const api = useApi();

  const [services, setServices] = useState<RegisteredService[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [refreshing, setRefreshing] = useState(false);

  const [formOpen, setFormOpen] = useState(false);
  const [formState, setFormState] = useState<FormState>(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [inspectedService, setInspectedService] = useState<RegisteredService | null>(null);
  const [healthHistory, setHealthHistory] = useState<ServiceHealthCheck[]>([]);
  const [healthLoading, setHealthLoading] = useState(false);
  const [accessModalService, setAccessModalService] = useState<RegisteredService | null>(null);
  const [permissions, setPermissions] = useState<ServicePermissionRecord[]>([]);
  const [permissionsLoading, setPermissionsLoading] = useState(false);
  const [permissionsFilter, setPermissionsFilter] = useState('');
  const [updatingPermission, setUpdatingPermission] = useState<number | null>(null);

  const mapError = useCallback((err: unknown): string => {
    const parsed = err as ApiErrorShape;
    return (
      parsed.response?.data?.error ||
      parsed.response?.data?.message ||
      parsed.message ||
      'Unable to process your request right now.'
    );
  }, []);

  const sortServices = useCallback((items: RegisteredService[]): RegisteredService[] => {
    return [...items].sort((a, b) => a.display_name.localeCompare(b.display_name));
  }, []);

  const loadServices = useCallback(async () => {
    try {
      setError('');
      setSuccess('');
      setRefreshing(true);
      const data = await api.sso.listServices();
      setServices(sortServices(data));
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [api.sso, mapError, sortServices]);

  useEffect(() => {
    loadServices();
  }, [loadServices]);

  const totalHealthy = useMemo(
    () => services.filter((service: RegisteredService) => service.is_healthy).length,
    [services]
  );

  const filteredPermissions = useMemo(() => {
    if (!permissionsFilter.trim()) {
      return permissions;
    }

    const keywords = permissionsFilter.toLowerCase();
    return permissions.filter((record) => {
      return (
        record.username.toLowerCase().includes(keywords) ||
        record.email.toLowerCase().includes(keywords)
      );
    });
  }, [permissions, permissionsFilter]);

  const handleResetForm = useCallback(() => {
    setFormState(emptyForm);
    setEditingId(null);
  }, []);

  const fetchPermissions = useCallback(async (serviceId: string) => {
    try {
      setPermissionsLoading(true);
      const data = await api.sso.getServicePermissions(serviceId);
      setPermissions(data);
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setPermissionsLoading(false);
    }
  }, [api, mapError]);

  const handleManageAccess = (service: RegisteredService) => {
    setError('');
    setSuccess('');
    setPermissions([]);
    setPermissionsFilter('');
    setAccessModalService(service);
    fetchPermissions(service.id);
  };

  const closeAccessModal = () => {
    setAccessModalService(null);
    setPermissions([]);
    setPermissionsFilter('');
    setUpdatingPermission(null);
  };

  const handleTogglePermission = async (record: ServicePermissionRecord) => {
    if (!accessModalService) {
      return;
    }

    try {
      setUpdatingPermission(record.user_id);
      if (record.can_access) {
        await api.sso.revokeServiceAccess(record.user_id, accessModalService.id);
        setSuccess(`Revoked ${record.username} from ${accessModalService.display_name}.`);
      } else {
        await api.sso.grantServiceAccess(record.user_id, accessModalService.id);
        setSuccess(`Granted ${record.username} access to ${accessModalService.display_name}.`);
      }
      await fetchPermissions(accessModalService.id);
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setUpdatingPermission(null);
    }
  };

  const handleOpenCreate = () => {
    handleResetForm();
    setFormOpen(true);
  };

  const handleEditService = (service: RegisteredService) => {
    setFormState({
      name: service.name,
      display_name: service.display_name,
      description: service.description ?? '',
      service_url: service.service_url,
      callback_url: service.callback_url ?? '',
      icon: service.icon ?? '',
      category: service.category,
      is_public: service.is_public,
      required_role: service.required_role,
      health_url: service.health_url ?? '',
    });
    setEditingId(service.id);
    setFormOpen(true);
  };

  const handleDeleteService = async (service: RegisteredService) => {
    if (!window.confirm(`Delete ${service.display_name}? This action cannot be undone.`)) {
      return;
    }

    try {
      setError('');
      await api.sso.deleteService(service.id);
      setSuccess(`Removed ${service.display_name}.`);
      await loadServices();
    } catch (err: unknown) {
      setError(mapError(err));
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    try {
      setSubmitting(true);
      setError('');
      setSuccess('');

      const payload = {
        name: formState.name,
        display_name: formState.display_name,
        description: formState.description.trim() || undefined,
        service_url: formState.service_url,
        callback_url: formState.callback_url.trim() || undefined,
        icon: formState.icon.trim() || undefined,
        category: formState.category,
        is_public: formState.is_public,
        required_role: formState.required_role,
        health_url: formState.health_url.trim() || undefined,
      };

      if (editingId) {
        await api.sso.updateService(editingId, payload);
        setSuccess(`Updated ${formState.display_name}.`);
      } else {
        await api.sso.registerService(payload);
        setSuccess(`Registered ${formState.display_name}.`);
      }

      await loadServices();
      setFormOpen(false);
      handleResetForm();
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleInspectHealth = async (service: RegisteredService) => {
    try {
      setError('');
      setInspectedService(service);
      setHealthLoading(true);

      const history = await api.sso.getServiceHealthHistory(service.id, 10);
      setHealthHistory(history);
    } catch (err: unknown) {
      setError(mapError(err));
    } finally {
      setHealthLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 text-primary-500 animate-spin" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">SSO Service Directory</h1>
          <p className="mt-1 text-sm text-gray-600">
            Register, audit, and monitor federated services connected to the Infra-Core gateway.
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
          <button
            onClick={handleOpenCreate}
            className="inline-flex items-center px-4 py-2 rounded-md text-sm font-medium border border-primary-200 text-primary-600 bg-white hover:bg-primary-50"
          >
            <Plus className="h-4 w-4 mr-2" />
            New Service
          </button>
        </div>
      </header>

      <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white shadow rounded-lg p-4 flex items-center">
          <Shield className="h-10 w-10 text-primary-500 mr-3" />
          <div>
            <p className="text-sm text-gray-500">Registered services</p>
            <p className="text-2xl font-semibold text-gray-900">{services.length}</p>
          </div>
        </div>
        <div className="bg-white shadow rounded-lg p-4 flex items-center">
          <CheckCircle2 className="h-10 w-10 text-green-500 mr-3" />
          <div>
            <p className="text-sm text-gray-500">Reporting healthy</p>
            <p className="text-2xl font-semibold text-gray-900">{totalHealthy}</p>
          </div>
        </div>
        <div className="bg-white shadow rounded-lg p-4 flex items-center">
          <TrendingUp className="h-10 w-10 text-indigo-500 mr-3" />
          <div>
            <p className="text-sm text-gray-500">Active directory entries</p>
            <p className="text-2xl font-semibold text-gray-900">
              {services.filter((service) => service.status === 'active').length}
            </p>
          </div>
        </div>
      </section>

      {(error || success) && (
        <div className="space-y-2">
          {error && (
            <div className="bg-red-50 border border-red-100 text-red-700 rounded-md p-4 flex items-start gap-3">
              <AlertCircle className="h-5 w-5 mt-0.5" />
              <div>
                <p className="font-medium">Action required</p>
                <p className="text-sm">{error}</p>
              </div>
            </div>
          )}
          {success && (
            <div className="bg-green-50 border border-green-100 text-green-700 rounded-md p-4 flex items-start gap-3">
              <CheckCircle2 className="h-5 w-5 mt-0.5" />
              <div>
                <p className="font-medium">Success</p>
                <p className="text-sm">{success}</p>
              </div>
            </div>
          )}
        </div>
      )}

      {formOpen && (
        <section className="bg-white shadow rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900">
              {editingId ? 'Update service' : 'Register new service'}
            </h2>
            <button
              onClick={() => {
                setFormOpen(false);
                handleResetForm();
              }}
              className="text-sm text-gray-500 hover:text-gray-700"
            >
              Close
            </button>
          </div>

          <form className="grid grid-cols-1 md:grid-cols-2 gap-4" onSubmit={handleSubmit}>
            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Service name</span>
              <input
                type="text"
                value={formState.name}
                onChange={(event) => setFormState((state) => ({ ...state, name: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="internal identifier"
                required
                disabled={!!editingId}
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Display name</span>
              <input
                type="text"
                value={formState.display_name}
                onChange={(event) => setFormState((state) => ({ ...state, display_name: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="Service shown to users"
                required
              />
            </label>

            <label className="flex flex-col gap-1 md:col-span-2">
              <span className="text-sm font-medium text-gray-700">Description</span>
              <textarea
                value={formState.description}
                onChange={(event) => setFormState((state) => ({ ...state, description: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                rows={2}
                placeholder="Explain what this service does"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Service URL</span>
              <input
                type="url"
                value={formState.service_url}
                onChange={(event) => setFormState((state) => ({ ...state, service_url: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="https://service.example.com"
                required
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Callback URL</span>
              <input
                type="url"
                value={formState.callback_url}
                onChange={(event) => setFormState((state) => ({ ...state, callback_url: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="SSO redirect target"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Icon URL</span>
              <input
                type="url"
                value={formState.icon}
                onChange={(event) => setFormState((state) => ({ ...state, icon: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="Optional branding icon"
              />
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Category</span>
              <select
                value={formState.category}
                onChange={(event) => setFormState((state) => ({ ...state, category: event.target.value as RegisteredService['category'] }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
              >
                <option value="web">Web</option>
                <option value="admin">Admin</option>
                <option value="monitoring">Monitoring</option>
                <option value="api">API</option>
                <option value="other">Other</option>
              </select>
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Required role</span>
              <select
                value={formState.required_role}
                onChange={(event) => setFormState((state) => ({ ...state, required_role: event.target.value as RegisteredService['required_role'] }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </label>

            <label className="flex flex-col gap-1">
              <span className="text-sm font-medium text-gray-700">Health check URL</span>
              <input
                type="url"
                value={formState.health_url}
                onChange={(event) => setFormState((state) => ({ ...state, health_url: event.target.value }))}
                className="px-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                placeholder="https://service.example.com/health"
              />
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formState.is_public}
                onChange={(event) => setFormState((state) => ({ ...state, is_public: event.target.checked }))}
                className="h-4 w-4 text-primary-600 border-gray-300 rounded"
              />
              <span className="text-sm text-gray-700">Public service (available to all authenticated users)</span>
            </label>

            <div className="md:col-span-2 flex justify-end gap-3 mt-2">
              <button
                type="button"
                onClick={() => {
                  handleResetForm();
                  setFormOpen(false);
                }}
                className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-800"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="inline-flex items-center px-4 py-2 rounded-md text-sm font-medium border border-transparent text-white bg-primary-600 hover:bg-primary-700 disabled:opacity-60"
              >
                {submitting && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
                {editingId ? 'Save changes' : 'Register service'}
              </button>
            </div>
          </form>
        </section>
      )}

      <section className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Service</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Category</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Access</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Last healthy</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {services.map((service) => (
              <tr key={service.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="text-sm font-medium text-gray-900">{service.display_name}</div>
                  <div className="text-xs text-gray-500">{service.service_url}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 capitalize">
                  {service.category}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusBadge(service.status)}`}>
                    {service.status}
                  </span>
                  <span className={`ml-2 px-2 py-1 rounded-full text-xs font-medium ${healthBadge(service.is_healthy)}`}>
                    {service.is_healthy ? 'Healthy' : 'Degraded'}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {requiredRoleLabel(service)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {service.last_healthy ? new Date(service.last_healthy).toLocaleString() : 'Never'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                  <button
                    onClick={() => handleInspectHealth(service)}
                    className="inline-flex items-center px-2 py-1 text-xs rounded-md border border-gray-200 text-gray-600 hover:bg-gray-100"
                  >
                    <Activity className="h-3 w-3 mr-1" />
                    Health
                  </button>
                  <button
                    onClick={() => handleManageAccess(service)}
                    className="inline-flex items-center px-2 py-1 text-xs rounded-md border border-gray-200 text-gray-600 hover:bg-gray-100"
                  >
                    <Users className="h-3 w-3 mr-1" />
                    Access
                  </button>
                  <button
                    onClick={() => handleEditService(service)}
                    className="inline-flex items-center px-2 py-1 text-xs rounded-md border border-gray-200 text-gray-600 hover:bg-gray-100"
                  >
                    <Edit3 className="h-3 w-3 mr-1" />
                    Edit
                  </button>
                  <button
                    onClick={() => handleDeleteService(service)}
                    className="inline-flex items-center px-2 py-1 text-xs rounded-md border border-red-200 text-red-600 hover:bg-red-50"
                  >
                    <Trash2 className="h-3 w-3 mr-1" />
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {services.length === 0 && (
          <div className="p-10 text-center text-sm text-gray-500">
            No services registered yet. Use “New Service” to add your first entry.
          </div>
        )}
      </section>

        {accessModalService && (
          <div className="fixed inset-0 z-40 flex items-center justify-center bg-gray-900/50 px-4">
            <div className="w-full max-w-3xl bg-white rounded-lg shadow-xl">
              <div className="flex items-start justify-between border-b border-gray-200 p-6">
                <div>
                  <h2 className="text-lg font-semibold text-gray-900">{accessModalService.display_name} access control</h2>
                  <p className="text-sm text-gray-500">Grant or revoke access for individual users.</p>
                </div>
                <button
                  onClick={closeAccessModal}
                  className="text-gray-400 hover:text-gray-600"
                  aria-label="Close access management dialog"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
              <div className="p-6 space-y-4">
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                  <div className="relative w-full sm:w-64">
                    <Search className="h-4 w-4 text-gray-400 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                      type="search"
                      value={permissionsFilter}
                      onChange={(event) => setPermissionsFilter(event.target.value)}
                      className="w-full pl-9 pr-3 py-2 border border-gray-200 rounded-md focus:ring-2 focus:ring-primary-200 focus:border-primary-400"
                      placeholder="Search users by name or email"
                    />
                  </div>
                  <p className="text-sm text-gray-500">
                    Showing <span className="font-medium text-gray-900">{filteredPermissions.length}</span> of{' '}
                    <span className="font-medium text-gray-900">{permissions.length}</span> users
                  </p>
                </div>

                {permissionsLoading ? (
                  <div className="flex items-center justify-center h-48">
                    <Loader2 className="h-6 w-6 text-primary-500 animate-spin" />
                  </div>
                ) : filteredPermissions.length === 0 ? (
                  <div className="h-48 flex items-center justify-center text-sm text-gray-500">
                    No users match the current filter.
                  </div>
                ) : (
                  <div className="overflow-hidden border border-gray-200 rounded-lg">
                    <table className="min-w-full divide-y divide-gray-200">
                      <thead className="bg-gray-50">
                        <tr>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Role</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Granted</th>
                          <th className="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-100">
                        {filteredPermissions.map((record) => {
                          const grantedAtLabel = record.granted_at
                            ? new Date(record.granted_at).toLocaleString()
                            : 'Never';

                          return (
                            <tr key={record.user_id}>
                              <td className="px-4 py-3">
                                <div className="text-sm font-medium text-gray-900">{record.username}</div>
                                <div className="text-xs text-gray-500">{record.email}</div>
                              </td>
                              <td className="px-4 py-3 text-sm text-gray-500 capitalize">{record.role}</td>
                              <td className="px-4 py-3">
                                <span
                                  className={`px-2 py-1 rounded-full text-xs font-medium ${
                                    record.can_access ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
                                  }`}
                                >
                                  {record.can_access ? 'Active' : 'No access'}
                                </span>
                              </td>
                              <td className="px-4 py-3 text-sm text-gray-500">
                                {record.can_access ? (
                                  <div className="space-y-1">
                                    <div>{grantedAtLabel}</div>
                                    <div className="text-xs text-gray-400">
                                      {record.granted_by_username
                                        ? `By ${record.granted_by_username}`
                                        : record.granted_by
                                          ? `By user #${record.granted_by}`
                                          : 'Awaiting audit'}
                                    </div>
                                  </div>
                                ) : (
                                  <span className="text-xs text-gray-400">Not granted</span>
                                )}
                              </td>
                              <td className="px-4 py-3 text-right">
                                <button
                                  onClick={() => handleTogglePermission(record)}
                                  disabled={updatingPermission === record.user_id}
                                  className={`inline-flex items-center px-3 py-1.5 text-xs font-medium rounded-md border transition-colors ${
                                    record.can_access
                                      ? 'border-red-200 text-red-600 hover:bg-red-50'
                                      : 'border-primary-200 text-primary-600 hover:bg-primary-50'
                                  } ${updatingPermission === record.user_id ? 'opacity-60 cursor-not-allowed' : ''}`}
                                >
                                  {updatingPermission === record.user_id ? (
                                    <Loader2 className="h-3.5 w-3.5 mr-2 animate-spin" />
                                  ) : null}
                                  {record.can_access ? 'Revoke' : 'Grant'}
                                </button>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

      {inspectedService && (
        <section className="bg-white shadow rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-900">{inspectedService.display_name} health history</h2>
              <p className="text-sm text-gray-500">Last 10 probes recorded by the SSO gateway</p>
            </div>
            <button
              onClick={() => {
                setInspectedService(null);
                setHealthHistory([]);
              }}
              className="text-sm text-gray-500 hover:text-gray-700"
            >
              Dismiss
            </button>
          </div>

          {healthLoading ? (
            <div className="flex items-center justify-center h-32">
              <Loader2 className="h-6 w-6 text-primary-500 animate-spin" />
            </div>
          ) : healthHistory.length === 0 ? (
            <div className="text-sm text-gray-500">No health data available for this service yet.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Checked at</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Response time</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Details</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-100">
                  {healthHistory.map((check) => (
                    <tr key={check.id}>
                      <td className="px-4 py-2 text-sm text-gray-600">
                        {new Date(check.checked_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-2 text-sm">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${healthBadge(check.is_healthy)}`}>
                          {check.is_healthy ? 'Healthy' : 'Failure'}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-600">
                        {check.response_time.toFixed(0)} ms
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500">
                        {check.error_message || 'OK'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      )}
    </div>
  );
};

export default SSOAdmin;

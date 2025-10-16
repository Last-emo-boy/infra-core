import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { ApiProvider } from './contexts/ApiContext';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Services from './pages/Services';
import System from './pages/System';
import Portal from './pages/Portal';
import SSOAdmin from './pages/SSOAdmin';
import Layout from './components/Layout';
import ProtectedRoute from './components/ProtectedRoute';
import RoleGuard from './components/RoleGuard';

const App: React.FC = () => {
  return (
    <ApiProvider>
      <AuthProvider>
        <Router>
          <div className="min-h-screen bg-gradient-dark">
            <Routes>
              {/* Public routes */}
              <Route path="/login" element={<Login />} />
              
              {/* Protected routes */}
              <Route path="/" element={
                <ProtectedRoute>
                  <Layout />
                </ProtectedRoute>
              }>
                <Route index element={<Navigate to="/portal" replace />} />
                <Route path="portal" element={<Portal />} />
                <Route path="dashboard" element={<Dashboard />} />
                <Route path="services" element={<Services />} />
                <Route path="system" element={<System />} />
                <Route
                  path="sso"
                  element={
                    <RoleGuard roles={["admin"]}>
                      <SSOAdmin />
                    </RoleGuard>
                  }
                />
              </Route>
              
              {/* Catch all */}
              <Route path="*" element={<Navigate to="/portal" replace />} />
            </Routes>
          </div>
        </Router>
      </AuthProvider>
    </ApiProvider>
  );
};

export default App;

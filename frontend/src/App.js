import React, { useEffect, useState } from 'react';
import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
} from 'react-router-dom';

import {
  getCurrentUser,
  logout,
} from './api/authClient';

import LoginPage from './pages/LoginPage';
import EquipmentPage from './pages/EquipmentPage';
import BorrowRequestPage from './pages/BorrowRequestPage';
import RepairTicketPage from './pages/RepairTicketPage';
import DashboardPage from './pages/DashboardPage';
import ReportPage from './pages/ReportPage';
import ImportPage from './pages/ImportPage';
import ActivityLogPage from './pages/ActivityLogPage';
import ProtectedRoute from './routes/ProtectedRoute';
import AppLayout from './layouts/AppLayout';

function App() {
  const [sessionState, setSessionState] =
    useState('checking');

  const [currentUser, setCurrentUser] =
    useState(null);

  useEffect(() => {
    let cancelled = false;

    async function restoreSession() {
      try {
        const result = await getCurrentUser();

        if (!cancelled) {
          setCurrentUser(result?.user || null);
          setSessionState('authenticated');
        }
      } catch {
        if (!cancelled) {
          setCurrentUser(null);
          setSessionState('anonymous');
        }
      }
    }

    restoreSession();

    return () => {
      cancelled = true;
    };
  }, []);

  function handleLoginSuccess(user) {
    setCurrentUser(user);
    setSessionState('authenticated');
  }

  async function handleLogout() {
    try {
      await logout();
    } finally {
      setCurrentUser(null);
      setSessionState('anonymous');
    }
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={
            sessionState === 'checking' ? (
              <div>Checking session...</div>
            ) : sessionState === 'authenticated' ? (
              <Navigate to="/equipment" replace />
            ) : (
              <LoginPage
                onLoginSuccess={handleLoginSuccess}
              />
            )
          }
        />

        <Route
          element={
            <ProtectedRoute sessionState={sessionState}>
              <AppLayout
                user={currentUser}
                onLogout={handleLogout}
              />
            </ProtectedRoute>
          }
        >
          <Route
            path="/equipment"
            element={
              <EquipmentPage user={currentUser} />
            }
          />

          <Route
            path="/borrow"
            element={
              <BorrowRequestPage user={currentUser} />
            }
          />

          <Route
            path="/repair-equipment"
            element={
              <RepairTicketPage user={currentUser} />
            }
          />
          <Route
            path="/dashboard"
            element={<DashboardPage user={currentUser} />}
          />
          <Route
            path="/report"
            element={<ReportPage user={currentUser} />}
          />
          <Route
            path="/import-data"
            element={<ImportPage user={currentUser} />}
          />
          <Route
            path="/activity-logs"
            element={<ActivityLogPage user={currentUser} />}
          />
        </Route>

        <Route
          path="/"
          element={
            sessionState === 'checking' ? (
              <div>Checking session...</div>
            ) : sessionState === 'authenticated' ? (
              <Navigate to="/equipment" replace />
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />

        <Route
          path="*"
          element={<Navigate to="/" replace />}
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App; 
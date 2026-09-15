import { Navigate } from 'react-router-dom';

function ProtectedRoute({ sessionState, children }) {
  if (sessionState === 'checking') {
    return <div>Checking session...</div>;
  }

  if (sessionState !== 'authenticated') {
    return <Navigate to="/login" replace />;
  }

  return children;
}

export default ProtectedRoute;
import React from 'react';
import ReactDOM from 'react-dom/client';
import EnhancedApp from './EnhancedApp';
import { ErrorBoundary } from './components/ErrorBoundary';
import './index.css';
import './i18n/config';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <EnhancedApp />
    </ErrorBoundary>
  </React.StrictMode>,
);

import React from 'react';
import ReactDOM from 'react-dom/client';
import EnhancedApp from './EnhancedApp';
import './index.css';
import './i18n/config';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <EnhancedApp />
  </React.StrictMode>,
);

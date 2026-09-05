import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import '@/tailwind.css';
import '@/main.less';
import { AdminLayout } from './framework/layout/layout';
import { StoreGate } from './framework/store/gate';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <StoreGate>
        <AdminLayout />
      </StoreGate>
    </BrowserRouter>
  </React.StrictMode>,
);

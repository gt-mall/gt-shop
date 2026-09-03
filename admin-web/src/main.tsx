import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import '@/main.less';
import { AdminLayout } from './framework/layout/layout';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <AdminLayout />
    </BrowserRouter>
  </React.StrictMode>,
);

import axios from 'axios';

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 65000,
  headers: {
    'Content-Type': 'application/json'
  },
  transformResponse: [function (data) {
    if (typeof data === 'string') {
      try {
        // Prevent JS int64 precision loss for Snowflake IDs in objects and arrays
        const fixedData = data.replace(/([:\[,]\s*)(\d{15,20})(?=\s*[,\]\}])/g, '$1"$2"');
        return JSON.parse(fixedData);
      } catch (e) {
        return data;
      }
    }
    return data;
  }]
});

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers['Authorization'] = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => {
    if (response.data && response.data.errCode === 1401) {
      window.dispatchEvent(new Event('auth-expired'));
      return Promise.reject(new Error(response.data.errMsg || '登录凭证已过期'));
    }
    if (response.data && response.data.errCode && response.data.errCode !== 0) {
      const msg = response.data.errDlt || response.data.errMsg || '请求错误';
      return Promise.reject(new Error(msg));
    }
    return response;
  },
  (error) => Promise.reject(error)
);

export const unwrap = (res) => {
  if (res && res.data && res.data.data !== undefined) {
    return res.data.data;
  }
  return res?.data;
};

export default api;


import axios, { AxiosRequestConfig, AxiosResponse } from 'axios';
import { ElNotification } from 'element-plus';
import {
  getEmptyResponseWithListData,
  getRequestBaseUrl,
} from '@/utils/request';
import { Router } from 'vue-router';
import { translate } from '@/utils/i18n';
import {
  LOCAL_STORAGE_KEY_REFRESH_TOKEN,
  LOCAL_STORAGE_KEY_TOKEN,
} from '@/constants/localStorage';

const t = translate;

// initialize request
export const initRequest = (router?: Router) => {
  // response interception
  let msgBoxVisible = false;
  let refreshPromise: Promise<string> | undefined;
  const clearSession = () => {
    window.localStorage.removeItem(LOCAL_STORAGE_KEY_TOKEN);
    window.localStorage.removeItem(LOCAL_STORAGE_KEY_REFRESH_TOKEN);
  };

  const refreshAccessToken = async (): Promise<string> => {
    if (refreshPromise) return refreshPromise;
    const refreshToken = window.localStorage.getItem(LOCAL_STORAGE_KEY_REFRESH_TOKEN);
    if (!refreshToken) throw new Error('refresh token is missing');

    refreshPromise = axios
      .post(
        '/refresh',
        { refresh_token: refreshToken },
        { baseURL: getRequestBaseUrl(), _jwtRefreshRequest: true } as AxiosRequestConfig
      )
      .then(res => {
        const pair = res.data?.data;
        if (!pair?.token || !pair?.refresh_token) {
          throw new Error('refresh token response is invalid');
        }
        window.localStorage.setItem(LOCAL_STORAGE_KEY_TOKEN, pair.token);
        window.localStorage.setItem(
          LOCAL_STORAGE_KEY_REFRESH_TOKEN,
          pair.refresh_token
        );
        return pair.token as string;
      })
      .finally(() => {
        refreshPromise = undefined;
      });
    return refreshPromise;
  };

  axios.interceptors.response.use(
    res => {
      return res;
    },
    async err => {
      // status code
      const status = err?.response?.status;

      const originalRequest = err.config as AxiosRequestConfig & {
        _jwtRetried?: boolean;
        _jwtRefreshRequest?: boolean;
      };
      if (
        status === 401 &&
        !originalRequest?._jwtRetried &&
        !originalRequest?._jwtRefreshRequest &&
        window.localStorage.getItem(LOCAL_STORAGE_KEY_REFRESH_TOKEN)
      ) {
        originalRequest._jwtRetried = true;
        try {
          const token = await refreshAccessToken();
          originalRequest.headers = {
            ...(originalRequest.headers || {}),
            Authorization: token,
          };
          return axios.request(originalRequest);
        } catch (_) {
          clearSession();
        }
      }

      if (status === 401) {
        // 401 error
        if (window.localStorage.getItem(LOCAL_STORAGE_KEY_TOKEN)) {
          // token expired, popup logout warning
          if (msgBoxVisible) return;
          msgBoxVisible = true;
          setTimeout(() => (msgBoxVisible = false), 5000);
          await router?.push('/login');
          ElNotification({
            title: t('common.status.unauthorized'),
            message: t('common.notification.loggedOut'),
            type: 'warning',
          });
        } else {
          // token not exists, redirect to login page
          await router?.push('/login');
        }
      } else {
        // other errors
        console.error(err);
        const { message, error } = err.response?.data || {};
        if (message === 'error' && error) {
          throw new Error(error);
        }
        throw err;
      }
    }
  );
};

const useRequest = () => {
  const getHeaders = (): any => {
    // headers
    const headers = {} as any;

    // add token to headers
    const token = localStorage.getItem(LOCAL_STORAGE_KEY_TOKEN);
    if (token) {
      headers['Authorization'] = token;
    }

    return headers;
  };

  const request = async <R = any>(opts: AxiosRequestConfig): Promise<R> => {
    // base url
    const baseURL = getRequestBaseUrl();

    // headers
    const headers = getHeaders();

    // axios response
    const res = await axios.request({
      ...opts,
      baseURL,
      headers,
    });

    // response data
    return res?.data;
  };

  const get = async <T = any, R = ResponseWithData<T>, PM = any>(
    url: string,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    opts = {
      ...opts,
      method: 'GET',
      url,
      params,
    };
    return await request<R>(opts);
  };

  const post = async <T = any, R = ResponseWithData<T>, PM = any>(
    url: string,
    data?: T,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    opts = {
      ...opts,
      method: 'POST',
      url,
      data,
      params,
    };
    return await request<R>(opts);
  };

  const put = async <T = any, R = ResponseWithData<T>, PM = any>(
    url: string,
    data?: T,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    opts = {
      ...opts,
      method: 'PUT',
      url,
      data,
      params,
    };
    return await request<R>(opts);
  };

  const del = async <T = any, R = ResponseWithData<T>, PM = any>(
    url: string,
    data?: T,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    opts = {
      ...opts,
      method: 'DELETE',
      url,
      data,
      params,
    };
    return await request<R>(opts);
  };

  const getList = async <T = any>(
    url: string,
    params?: ListRequestParams,
    opts?: AxiosRequestConfig
  ) => {
    // get request
    const res = await get<T, ResponseWithListData<T>, ListRequestParams>(
      url,
      params,
      opts
    );

    // if no response, return empty
    if (!res) {
      return getEmptyResponseWithListData();
    }

    // normalize array data
    if (!res.data) {
      res.data = [];
    }

    return res;
  };

  const postList = async <T = any, R = ResponseWithListData, PM = any>(
    url: string,
    data?: T[],
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    return await post<T[], R, PM>(url, data, params, opts);
  };

  const putList = async <T = any, R = Response, PM = any>(
    url: string,
    data?: BatchRequestPayloadWithJsonStringData,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    return await put<BatchRequestPayloadWithJsonStringData, R, PM>(
      url,
      data,
      params,
      opts
    );
  };

  const delList = async <T = any, R = Response, PM = any>(
    url: string,
    data?: BatchRequestPayload,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<R> => {
    return await del<BatchRequestPayload, R, PM>(url, data, params, opts);
  };

  const requestRaw = async <R = any>(
    opts: AxiosRequestConfig
  ): Promise<AxiosResponse> => {
    // base url
    const baseURL = getRequestBaseUrl();

    // headers
    const headers = getHeaders();

    // axios response
    return await axios.request({
      ...opts,
      baseURL,
      headers,
    });
  };

  const getRaw = async <T = any, PM = any>(
    url: string,
    params?: PM,
    opts?: AxiosRequestConfig
  ): Promise<AxiosResponse> => {
    opts = {
      ...opts,
      method: 'GET',
      url,
      params,
    };
    return await requestRaw(opts);
  };

  return {
    // public variables and methods
    request,
    get,
    post,
    put,
    del,
    getList,
    postList,
    putList,
    delList,
    requestRaw,
    getRaw,
  };
};

export default useRequest;

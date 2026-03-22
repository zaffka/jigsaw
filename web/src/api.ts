import type { User, CatalogPuzzle } from './types';

const BASE = '/api';

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  isForm = false,
): Promise<T> {
  const headers: Record<string, string> = {};
  let bodyInit: BodyInit | undefined;

  if (body !== undefined) {
    if (isForm) {
      bodyInit = body as FormData;
      // Let browser set Content-Type with boundary for multipart
    } else {
      headers['Content-Type'] = 'application/json';
      bodyInit = JSON.stringify(body);
    }
  }

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: bodyInit,
    credentials: 'same-origin',
  });

  if (res.status === 401) {
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(text || res.statusText);
  }

  const contentType = res.headers.get('Content-Type') ?? '';
  if (contentType.includes('application/json')) {
    return res.json() as Promise<T>;
  }

  return undefined as unknown as T;
}

export function get<T>(path: string): Promise<T> {
  return request<T>('GET', path);
}

export function post<T>(path: string, body: unknown): Promise<T> {
  return request<T>('POST', path, body);
}

export function postForm<T>(path: string, formData: FormData): Promise<T> {
  return request<T>('POST', path, formData, true);
}

export function put<T>(path: string, body: unknown): Promise<T> {
  return request<T>('PUT', path, body);
}

export function del(path: string): Promise<void> {
  return request<void>('DELETE', path);
}

export const api = {
  auth: {
    login: (email: string, password: string) =>
      post<User>('/auth/login', { email, password }),
    register: (email: string, password: string, locale?: string) =>
      post<User>('/auth/register', { email, password, locale }),
    logout: () => post<void>('/auth/logout', {}),
    me: () => get<User>('/auth/me'),
  },
  catalog: {
    list: () => get<CatalogPuzzle[]>('/catalog'),
    get: (id: string) => get<CatalogPuzzle>(`/catalog/${id}`),
  },
  admin: {
    catalog: {
      list: () => get<CatalogPuzzle[]>('/admin/catalog'),
      create: (formData: FormData) => postForm<CatalogPuzzle>('/admin/catalog', formData),
      update: (id: string, data: Partial<CatalogPuzzle>) =>
        put<CatalogPuzzle>(`/admin/catalog/${id}`, data),
      delete: (id: string) => del(`/admin/catalog/${id}`),
    },
    users: {
      list: () => get<User[]>('/admin/users'),
    },
  },
};

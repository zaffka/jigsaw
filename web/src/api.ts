import type { User, Category, CatalogPuzzle, GamePuzzle, ParentPuzzle, PuzzleLayer, Child, ModerationItem, Submission } from './types';

const BASE = '/api';

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  isForm = false,
  extraHeaders?: Record<string, string>,
): Promise<T> {
  const headers: Record<string, string> = { ...extraHeaders };
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

export function get<T>(path: string, childToken?: string): Promise<T> {
  return request<T>('GET', path, undefined, false, childToken ? { 'X-Child-Token': childToken } : undefined);
}

export function post<T>(path: string, body: unknown, extraHeaders?: Record<string, string>): Promise<T> {
  return request<T>('POST', path, body, false, extraHeaders);
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

export function putForm<T>(path: string, formData: FormData): Promise<T> {
  return request<T>('PUT', path, formData, true);
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
  categories: {
    list: () => get<Category[]>('/categories'),
  },
  catalog: {
    list: (filters?: { category?: string; difficulty?: string }) => {
      const params = new URLSearchParams();
      if (filters?.category) params.set('category', filters.category);
      if (filters?.difficulty) params.set('difficulty', filters.difficulty);
      const qs = params.toString();
      return get<CatalogPuzzle[]>(qs ? `/catalog?${qs}` : '/catalog');
    },
    get: (id: string) => get<GamePuzzle>(`/catalog/${id}`),
  },
  play: {
    complete: (id: string): Promise<void> => {
      const token = sessionStorage.getItem('child_token') ?? '';
      return post<void>(`/play/${id}/complete`, null, token ? { 'X-Child-Token': token } : undefined)
        .catch((e) => console.warn('play complete failed:', e));
    },
    completed: (): Promise<string[]> => {
      const token = sessionStorage.getItem('child_token') ?? '';
      return token ? get<string[]>('/play/completed', token) : Promise.resolve([]);
    },
  },
  parent: {
    listChildren: () => get<Child[]>('/parent/children'),
    createChild: (data: { name: string; pin: string; avatar_emoji?: string }) =>
      post<Child>('/parent/children', data),
    updateChild: (id: string, data: { name: string; pin?: string; avatar_emoji?: string }) =>
      put<Child>(`/parent/children/${id}`, data),
    deleteChild: (id: string) => del(`/parent/children/${id}`),

    listPuzzles: () => get<ParentPuzzle[]>('/parent/puzzles'),
    getPuzzle: (id: string) => get<ParentPuzzle>(`/parent/puzzles/${id}`),
    createPuzzle: (form: FormData) => postForm<ParentPuzzle>('/parent/puzzles', form),
    updatePuzzle: (id: string, data: { title: string; category_id?: string | null }) =>
      put<ParentPuzzle>(`/parent/puzzles/${id}`, data),
    deletePuzzle: (id: string) => del(`/parent/puzzles/${id}`),

    listLayers: (puzzleId: string) => get<PuzzleLayer[]>(`/parent/puzzles/${puzzleId}/layers`),
    createLayer: (puzzleId: string, form: FormData) =>
      postForm<PuzzleLayer>(`/parent/puzzles/${puzzleId}/layers`, form),
    updateLayer: (puzzleId: string, layerId: string, form: FormData) =>
      putForm<PuzzleLayer>(`/parent/puzzles/${puzzleId}/layers/${layerId}`, form),
    deleteLayer: (puzzleId: string, layerId: string) =>
      del(`/parent/puzzles/${puzzleId}/layers/${layerId}`),
    reorderLayers: (puzzleId: string, items: Array<{ id: string; sort_order: number }>) =>
      post<{ ok: boolean }>(`/parent/puzzles/${puzzleId}/layers/reorder`, items),

    submit: (puzzleId: string) =>
      post<Submission>(`/parent/puzzles/${puzzleId}/submit`, {}),
    listNotifications: () => get<Submission[]>('/parent/notifications'),
  },
  children: {
    auth: (child_id: string, pin: string) =>
      post<{ token: string; child_id: string; name: string; avatar_emoji: string }>(
        '/children/auth',
        { child_id, pin },
      ),
  },
  admin: {
    catalog: {
      list: () => get<CatalogPuzzle[]>('/admin/catalog/puzzles'),
      create: (formData: FormData) => postForm<CatalogPuzzle>('/admin/catalog/puzzles', formData),
      update: (id: string, data: Partial<CatalogPuzzle>) =>
        put<CatalogPuzzle>(`/admin/catalog/puzzles/${id}`, data),
      delete: (id: string) => del(`/admin/catalog/puzzles/${id}`),
      getReward: (id: string) => get<unknown>(`/admin/catalog/puzzles/${id}/reward`),
      upsertReward: (id: string, form: FormData) =>
        postForm<unknown>(`/admin/catalog/puzzles/${id}/reward`, form),
    },
    users: {
      list: () => get<User[]>('/admin/users'),
    },
    moderation: {
      list: () => get<ModerationItem[]>('/admin/moderation'),
      approve: (id: string) => post<{ ok: boolean }>(`/admin/moderation/${id}/approve`, {}),
      reject: (id: string, comment: string) =>
        post<{ ok: boolean }>(`/admin/moderation/${id}/reject`, { comment }),
    },
  },
};

export interface User {
  id: string;
  email: string;
  role: 'parent' | 'admin';
  locale: string;
}

export interface CatalogPuzzle {
  id: string;
  puzzle_id: string;
  titles: Record<string, string>;
  image_key: string;
  status: 'processing' | 'ready' | 'failed';
  config: Record<string, unknown>;
  featured: boolean;
  sort_order: number;
  created_at: string;
}

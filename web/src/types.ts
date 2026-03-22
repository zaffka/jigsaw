export interface User {
  id: string;
  email: string;
  role: 'parent' | 'admin';
  locale: string;
}

export type Locale = 'ru' | 'en' | 'es' | 'zh' | 'th';

export interface CatalogPuzzle {
  id: string;
  puzzle_id: string;
  title: string;
  locale: Locale;
  image_key: string;
  status: 'processing' | 'ready' | 'failed';
  config: Record<string, unknown>;
  featured: boolean;
  sort_order: number;
  created_at: string;
}

export interface Reward {
  id: string;
  puzzle_id: string;
  video_key: string | null;
  word: string | null;
  tts_key: string | null;
  animation: string;
}

export interface PieceBounds {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface PuzzlePiece {
  id: string;
  image_key: string;
  svg_path: string;
  grid_x: number;
  grid_y: number;
  bounds: PieceBounds;
}

export interface GamePuzzle extends CatalogPuzzle {
  pieces: PuzzlePiece[];
  reward: Reward | null;
}

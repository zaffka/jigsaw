export interface User {
  id: string;
  email: string;
  role: 'parent' | 'admin';
  locale: string;
}

export type Locale = 'ru' | 'en' | 'es' | 'zh' | 'th';

export interface Category {
  id: string;
  slug: string;
  name: Record<string, string>;
  icon: string;
  sort_order: number;
}

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
  category: string | null;
  difficulty: 'easy' | 'medium' | 'hard' | '';
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

export interface ParentPuzzle {
  id: string;
  title: string;
  locale: string;
  image_key: string;
  status: 'processing' | 'ready' | 'failed';
  config: Record<string, unknown>;
  category: string | null;
  difficulty: 'easy' | 'medium' | 'hard' | '';
  created_at: string;
}

export interface PuzzleLayer {
  id: string;
  puzzle_id: string;
  sort_order: number;
  type: 'word' | 'audio' | 'video';
  text: string | null;
  audio_key: string | null;
  tts_key: string | null;
  video_key: string | null;
  created_at: string;
}

export interface Child {
  id: string;
  name: string;
  avatar_emoji: string;
  created_at: string;
}

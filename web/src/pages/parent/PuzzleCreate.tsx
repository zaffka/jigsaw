import { useState, useEffect, useRef } from 'preact/hooks';
import { useLocation } from 'wouter';
import { api } from '../../api';
import type { Category } from '../../types';

interface DraftLayer {
  type: 'word' | 'audio' | 'video';
  text?: string;
  file?: File;
}

type GridPreset = { label: string; cols: number; rows: number };

const GRID_PRESETS: GridPreset[] = [
  { label: '2×2', cols: 2, rows: 2 },
  { label: '3×3', cols: 3, rows: 3 },
  { label: '4×4', cols: 4, rows: 4 },
  { label: '3×4', cols: 3, rows: 4 },
  { label: '4×6', cols: 4, rows: 6 },
  { label: '6×8', cols: 6, rows: 8 },
];

const SLICE_MODES = ['grid', 'puzzle', 'geometry', 'merge'] as const;
type SliceMode = typeof SLICE_MODES[number];

function difficultyLabel(cols: number, rows: number): string {
  const count = cols * rows;
  if (count <= 6) return 'Лёгкий';
  if (count <= 16) return 'Средний';
  return 'Сложный';
}

function difficultyColor(cols: number, rows: number): string {
  const count = cols * rows;
  if (count <= 6) return 'text-green-600 bg-green-50';
  if (count <= 16) return 'text-yellow-600 bg-yellow-50';
  return 'text-red-600 bg-red-50';
}

// ─── Step 1 ─────────────────────────────────────────────────────────────────

interface Step1Props {
  imageFile: File | null;
  imagePreview: string | null;
  cols: number;
  rows: number;
  mode: SliceMode;
  onImageChange: (file: File, preview: string) => void;
  onPresetChange: (cols: number, rows: number) => void;
  onModeChange: (mode: SliceMode) => void;
  onNext: () => void;
}

function Step1({
  imageFile,
  imagePreview,
  cols,
  rows,
  mode,
  onImageChange,
  onPresetChange,
  onModeChange,
  onNext,
}: Step1Props) {
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFile = (file: File) => {
    if (!file.type.startsWith('image/')) return;
    if (file.size > 10 * 1024 * 1024) {
      alert('Файл слишком большой. Максимум 10 МБ.');
      return;
    }
    const url = URL.createObjectURL(file);
    onImageChange(file, url);
  };

  const handleDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const file = e.dataTransfer?.files[0];
    if (file) handleFile(file);
  };

  const handleInputChange = (e: Event) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (file) handleFile(file);
  };

  return (
    <div class="space-y-6">
      {/* Upload zone */}
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700">Изображение</label>
        <div
          class={`relative flex min-h-48 cursor-pointer flex-col items-center justify-center rounded-xl border-2 border-dashed transition-colors ${
            dragOver ? 'border-blue-500 bg-blue-50' : 'border-gray-300 bg-gray-50 hover:border-blue-400 hover:bg-blue-50'
          }`}
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
        >
          {imagePreview ? (
            <img
              src={imagePreview}
              alt="Preview"
              class="max-h-64 max-w-full rounded-lg object-contain"
            />
          ) : (
            <div class="text-center">
              <div class="mb-2 text-4xl text-gray-400">🖼️</div>
              <p class="text-sm text-gray-500">Перетащите изображение или нажмите для выбора</p>
              <p class="mt-1 text-xs text-gray-400">JPEG, PNG · максимум 10 МБ</p>
            </div>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png"
            class="hidden"
            onChange={handleInputChange}
          />
        </div>
        {imageFile && (
          <p class="mt-1 text-xs text-gray-500">{imageFile.name}</p>
        )}
      </div>

      {/* Grid presets */}
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700">Сетка</label>
        <div class="flex flex-wrap gap-2">
          {GRID_PRESETS.map((p) => (
            <button
              key={p.label}
              type="button"
              onClick={() => onPresetChange(p.cols, p.rows)}
              class={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                cols === p.cols && rows === p.rows
                  ? 'bg-blue-600 text-white'
                  : 'bg-white text-gray-700 ring-1 ring-gray-300 hover:bg-gray-50'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>
        <div class="mt-2">
          <span class={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${difficultyColor(cols, rows)}`}>
            {difficultyLabel(cols, rows)}
          </span>
          <span class="ml-2 text-xs text-gray-500">{cols * rows} кусочков</span>
        </div>
      </div>

      {/* Slicing mode */}
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700">Режим нарезки</label>
        <div class="flex flex-wrap gap-3">
          {SLICE_MODES.map((m) => (
            <label key={m} class="flex cursor-pointer items-center gap-2">
              <input
                type="radio"
                name="mode"
                value={m}
                checked={mode === m}
                onChange={() => onModeChange(m)}
                class="accent-blue-600"
              />
              <span class="text-sm capitalize text-gray-700">{m}</span>
            </label>
          ))}
        </div>
      </div>

      <div class="flex justify-end">
        <button
          type="button"
          onClick={onNext}
          disabled={!imageFile}
          class="rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          Далее →
        </button>
      </div>
    </div>
  );
}

// ─── Step 2 ─────────────────────────────────────────────────────────────────

interface Step2Props {
  layers: DraftLayer[];
  onAdd: (layer: DraftLayer) => void;
  onRemove: (index: number) => void;
  onBack: () => void;
  onNext: () => void;
  onSkip: () => void;
}

function layerIcon(type: DraftLayer['type']): string {
  if (type === 'word') return '📝';
  if (type === 'audio') return '🔊';
  return '🎬';
}

function layerPreview(layer: DraftLayer): string {
  if (layer.text) return layer.text;
  if (layer.file) return layer.file.name;
  return '—';
}

type AddMode = 'word' | 'audio' | 'video' | null;

function Step2({ layers, onAdd, onRemove, onBack, onNext, onSkip }: Step2Props) {
  const [addMode, setAddMode] = useState<AddMode>(null);
  const [wordText, setWordText] = useState('');
  const [mediaFile, setMediaFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const submitWord = () => {
    if (!wordText.trim()) return;
    onAdd({ type: 'word', text: wordText.trim() });
    setWordText('');
    setAddMode(null);
  };

  const submitMedia = () => {
    if (!mediaFile || !addMode || addMode === 'word') return;
    onAdd({ type: addMode, file: mediaFile });
    setMediaFile(null);
    setAddMode(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const openMediaMode = (type: 'audio' | 'video') => {
    setAddMode(type);
    setMediaFile(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const mediaAccept = addMode === 'audio' ? 'audio/mpeg,audio/wav' : 'video/mp4';

  return (
    <div class="space-y-5">
      {/* Layer list */}
      {layers.length > 0 ? (
        <ul class="space-y-2">
          {layers.map((layer, i) => (
            <li key={i} class="flex items-center justify-between rounded-lg bg-gray-50 px-4 py-3">
              <div class="flex items-center gap-3">
                <span class="text-xl">{layerIcon(layer.type)}</span>
                <span class="text-sm text-gray-700">{layerPreview(layer)}</span>
              </div>
              <button
                type="button"
                onClick={() => onRemove(i)}
                class="text-xs text-red-500 hover:text-red-700"
              >
                Удалить
              </button>
            </li>
          ))}
        </ul>
      ) : (
        <p class="text-sm text-gray-500">Слои не добавлены. Можно пропустить этот шаг.</p>
      )}

      {/* Add controls */}
      <div class="space-y-3">
        {addMode === 'word' && (
          <div class="flex gap-2">
            <input
              type="text"
              value={wordText}
              onInput={(e) => setWordText((e.target as HTMLInputElement).value)}
              onKeyDown={(e) => e.key === 'Enter' && submitWord()}
              placeholder="Введите слово или фразу"
              class="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
              autoFocus
            />
            <button
              type="button"
              onClick={submitWord}
              disabled={!wordText.trim()}
              class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              Добавить
            </button>
            <button
              type="button"
              onClick={() => setAddMode(null)}
              class="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100"
            >
              Отмена
            </button>
          </div>
        )}

        {(addMode === 'audio' || addMode === 'video') && (
          <div class="flex gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept={mediaAccept}
              onChange={(e) => setMediaFile((e.target as HTMLInputElement).files?.[0] ?? null)}
              class="flex-1 text-sm text-gray-600"
            />
            <button
              type="button"
              onClick={submitMedia}
              disabled={!mediaFile}
              class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              Добавить
            </button>
            <button
              type="button"
              onClick={() => setAddMode(null)}
              class="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100"
            >
              Отмена
            </button>
          </div>
        )}

        {addMode === null && (
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => setAddMode('word')}
              class="rounded-lg bg-white px-4 py-2 text-sm font-medium text-gray-700 ring-1 ring-gray-300 hover:bg-gray-50"
            >
              + Добавить слово
            </button>
            <button
              type="button"
              onClick={() => openMediaMode('audio')}
              class="rounded-lg bg-white px-4 py-2 text-sm font-medium text-gray-700 ring-1 ring-gray-300 hover:bg-gray-50"
            >
              + Добавить аудио
            </button>
            <button
              type="button"
              onClick={() => openMediaMode('video')}
              class="rounded-lg bg-white px-4 py-2 text-sm font-medium text-gray-700 ring-1 ring-gray-300 hover:bg-gray-50"
            >
              + Добавить видео
            </button>
          </div>
        )}
      </div>

      <div class="flex justify-between pt-2">
        <button
          type="button"
          onClick={onBack}
          class="rounded-lg px-5 py-2 text-sm text-gray-600 hover:bg-gray-100"
        >
          ← Назад
        </button>
        <div class="flex gap-2">
          <button
            type="button"
            onClick={onSkip}
            class="rounded-lg px-5 py-2 text-sm text-gray-600 hover:bg-gray-100"
          >
            Пропустить
          </button>
          <button
            type="button"
            onClick={onNext}
            class="rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Далее →
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── Step 3 ─────────────────────────────────────────────────────────────────

interface Step3Props {
  title: string;
  locale: string;
  categoryId: string;
  categories: Category[];
  saving: boolean;
  error: string | null;
  onTitleChange: (v: string) => void;
  onLocaleChange: (v: string) => void;
  onCategoryChange: (v: string) => void;
  onBack: () => void;
  onSave: () => void;
}

function Step3({
  title,
  locale,
  categoryId,
  categories,
  saving,
  error,
  onTitleChange,
  onLocaleChange,
  onCategoryChange,
  onBack,
  onSave,
}: Step3Props) {
  return (
    <div class="space-y-5">
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700">
          Название <span class="text-red-500">*</span>
        </label>
        <input
          type="text"
          value={title}
          onInput={(e) => onTitleChange((e.target as HTMLInputElement).value)}
          placeholder="Например: Любимые животные"
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        />
      </div>

      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700">Категория</label>
        <select
          value={categoryId}
          onChange={(e) => onCategoryChange((e.target as HTMLSelectElement).value)}
          class="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
        >
          <option value="">— Без категории —</option>
          {categories.map((cat) => (
            <option key={cat.id} value={cat.id}>
              {cat.icon} {cat.name['ru'] ?? cat.name['en'] ?? cat.slug}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700">Язык</label>
        <div class="flex gap-4">
          {(['ru', 'en'] as const).map((loc) => (
            <label key={loc} class="flex cursor-pointer items-center gap-2">
              <input
                type="radio"
                name="locale"
                value={loc}
                checked={locale === loc}
                onChange={() => onLocaleChange(loc)}
                class="accent-blue-600"
              />
              <span class="text-sm text-gray-700">{loc === 'ru' ? 'Русский' : 'English'}</span>
            </label>
          ))}
        </div>
      </div>

      {error && (
        <p class="rounded-lg bg-red-50 px-4 py-3 text-sm text-red-600">{error}</p>
      )}

      <div class="flex justify-between pt-2">
        <button
          type="button"
          onClick={onBack}
          disabled={saving}
          class="rounded-lg px-5 py-2 text-sm text-gray-600 hover:bg-gray-100 disabled:opacity-50"
        >
          ← Назад
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={saving || !title.trim()}
          class="flex items-center gap-2 rounded-lg bg-blue-600 px-5 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {saving && (
            <svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
          )}
          {saving ? 'Сохранение…' : 'Сохранить'}
        </button>
      </div>
    </div>
  );
}

// ─── Progress indicator ──────────────────────────────────────────────────────

const STEP_LABELS = ['Картинка', 'Слои', 'Детали'];

function ProgressBar({ step }: { step: number }) {
  return (
    <div class="mb-8 flex items-center gap-0">
      {STEP_LABELS.map((label, i) => (
        <div key={i} class="flex flex-1 flex-col items-center">
          <div class="flex w-full items-center">
            {i > 0 && (
              <div class={`h-0.5 flex-1 ${i <= step ? 'bg-blue-500' : 'bg-gray-200'}`} />
            )}
            <div
              class={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold ${
                i < step
                  ? 'bg-blue-500 text-white'
                  : i === step
                  ? 'bg-blue-600 text-white ring-4 ring-blue-100'
                  : 'bg-gray-200 text-gray-500'
              }`}
            >
              {i < step ? '✓' : i + 1}
            </div>
            {i < STEP_LABELS.length - 1 && (
              <div class={`h-0.5 flex-1 ${i < step ? 'bg-blue-500' : 'bg-gray-200'}`} />
            )}
          </div>
          <span class={`mt-1 text-xs ${i === step ? 'font-medium text-blue-600' : 'text-gray-500'}`}>
            {label}
          </span>
        </div>
      ))}
    </div>
  );
}

// ─── Main wizard ─────────────────────────────────────────────────────────────

export function PuzzleCreate() {
  const [, setLocation] = useLocation();
  const [step, setStep] = useState(0);

  // Step 1 state
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [cols, setCols] = useState(3);
  const [rows, setRows] = useState(3);
  const [mode, setMode] = useState<SliceMode>('grid');

  // Step 2 state
  const [layers, setLayers] = useState<DraftLayer[]>([]);

  // Step 3 state
  const [title, setTitle] = useState('');
  const [locale, setLocale] = useState('ru');
  const [categoryId, setCategoryId] = useState('');
  const [categories, setCategories] = useState<Category[]>([]);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    api.categories.list().then(setCategories).catch(() => {});
  }, []);

  const handleImageChange = (file: File, preview: string) => {
    setImageFile(file);
    setImagePreview(preview);
  };

  const handlePresetChange = (c: number, r: number) => {
    setCols(c);
    setRows(r);
  };

  const addLayer = (layer: DraftLayer) => setLayers((prev) => [...prev, layer]);
  const removeLayer = (index: number) => setLayers((prev) => prev.filter((_, i) => i !== index));

  const handleSave = async () => {
    if (!imageFile || !title.trim()) return;
    setSaving(true);
    setSaveError(null);

    try {
      const formData = new FormData();
      formData.append('image', imageFile);
      formData.append('title', title.trim());
      formData.append('locale', locale);
      formData.append('config', JSON.stringify({ cols, rows, mode }));
      if (categoryId) formData.append('category_id', categoryId);

      const puzzle = await api.parent.createPuzzle(formData);

      for (let i = 0; i < layers.length; i++) {
        const layer = layers[i];
        const layerForm = new FormData();
        layerForm.append('type', layer.type);
        layerForm.append('sort_order', String(i));

        if (layer.type === 'word' && layer.text) {
          layerForm.append('text', layer.text);
        } else if (layer.type === 'audio' && layer.file) {
          layerForm.append('audio', layer.file);
        } else if (layer.type === 'video' && layer.file) {
          layerForm.append('video', layer.file);
        }

        await api.parent.createLayer(puzzle.id, layerForm);
      }

      setLocation(`/parent/puzzles/${puzzle.id}`);
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Ошибка при сохранении');
      setSaving(false);
    }
  };

  return (
    <div class="mx-auto max-w-lg">
      <h1 class="mb-6 text-xl font-semibold text-gray-900">Новый пазл</h1>
      <ProgressBar step={step} />

      {step === 0 && (
        <Step1
          imageFile={imageFile}
          imagePreview={imagePreview}
          cols={cols}
          rows={rows}
          mode={mode}
          onImageChange={handleImageChange}
          onPresetChange={handlePresetChange}
          onModeChange={setMode}
          onNext={() => setStep(1)}
        />
      )}

      {step === 1 && (
        <Step2
          layers={layers}
          onAdd={addLayer}
          onRemove={removeLayer}
          onBack={() => setStep(0)}
          onNext={() => setStep(2)}
          onSkip={() => setStep(2)}
        />
      )}

      {step === 2 && (
        <Step3
          title={title}
          locale={locale}
          categoryId={categoryId}
          categories={categories}
          saving={saving}
          error={saveError}
          onTitleChange={setTitle}
          onLocaleChange={setLocale}
          onCategoryChange={setCategoryId}
          onBack={() => setStep(1)}
          onSave={handleSave}
        />
      )}
    </div>
  );
}

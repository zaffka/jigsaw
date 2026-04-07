import { useEffect, useState } from 'preact/hooks';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import type { Child } from '../../types';

interface CreateForm {
  name: string;
  pin: string;
  avatar_emoji: string;
}

interface EditForm {
  name: string;
  pin: string;
  avatar_emoji: string;
}

const DEFAULT_CREATE: CreateForm = { name: '', pin: '', avatar_emoji: '🧒' };

export function ChildList() {
  const [children, setChildren] = useState<Child[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showCreate, setShowCreate] = useState(false);
  const [createForm, setCreateForm] = useState<CreateForm>(DEFAULT_CREATE);
  const [creating, setCreating] = useState(false);

  const [editId, setEditId] = useState<string | null>(null);
  const [editForm, setEditForm] = useState<EditForm>({ name: '', pin: '', avatar_emoji: '' });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.parent
      .listChildren()
      .then(setChildren)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const handleCreate = async () => {
    if (!createForm.name.trim() || !createForm.pin.trim()) return;
    setCreating(true);
    try {
      const child = await api.parent.createChild({
        name: createForm.name.trim(),
        pin: createForm.pin.trim(),
        avatar_emoji: createForm.avatar_emoji || '🧒',
      });
      setChildren((prev) => [...prev, child]);
      setCreateForm(DEFAULT_CREATE);
      setShowCreate(false);
    } catch (e: unknown) {
      alert((e as Error).message);
    } finally {
      setCreating(false);
    }
  };

  const handleStartEdit = (child: Child) => {
    setEditId(child.id);
    setEditForm({ name: child.name, pin: '', avatar_emoji: child.avatar_emoji });
  };

  const handleSaveEdit = async () => {
    if (!editId || !editForm.name.trim()) return;
    setSaving(true);
    try {
      const data: { name: string; pin?: string; avatar_emoji?: string } = {
        name: editForm.name.trim(),
        avatar_emoji: editForm.avatar_emoji || undefined,
      };
      if (editForm.pin.trim()) {
        data.pin = editForm.pin.trim();
      }
      const updated = await api.parent.updateChild(editId, data);
      setChildren((prev) => prev.map((c) => (c.id === editId ? updated : c)));
      setEditId(null);
    } catch (e: unknown) {
      alert((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`Удалить ребёнка "${name}"?`)) return;
    try {
      await api.parent.deleteChild(id);
      setChildren((prev) => prev.filter((c) => c.id !== id));
    } catch (e: unknown) {
      alert((e as Error).message);
    }
  };

  if (loading) return <Spinner />;

  return (
    <div class="max-w-xl">
      <div class="mb-6 flex items-center justify-between">
        <h1 class="text-2xl font-semibold text-gray-900">Дети</h1>
        {!showCreate && (
          <button
            onClick={() => setShowCreate(true)}
            class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            + Добавить ребёнка
          </button>
        )}
      </div>

      {error && (
        <div class="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Create form */}
      {showCreate && (
        <div class="mb-6 rounded-md border border-gray-200 bg-gray-50 p-4">
          <h2 class="mb-3 text-sm font-semibold text-gray-700">Новый ребёнок</h2>
          <div class="space-y-3">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600">Имя</label>
              <input
                class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Имя ребёнка"
                value={createForm.name}
                onInput={(e) => setCreateForm((f) => ({ ...f, name: (e.target as HTMLInputElement).value }))}
                autoFocus
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600">PIN (4 цифры)</label>
              <input
                class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="1234"
                type="text"
                inputMode="numeric"
                maxLength={4}
                value={createForm.pin}
                onInput={(e) => setCreateForm((f) => ({ ...f, pin: (e.target as HTMLInputElement).value }))}
              />
            </div>
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600">Эмодзи-аватар</label>
              <input
                class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="🧒"
                value={createForm.avatar_emoji}
                onInput={(e) => setCreateForm((f) => ({ ...f, avatar_emoji: (e.target as HTMLInputElement).value }))}
              />
            </div>
          </div>
          <div class="mt-3 flex gap-2">
            <button
              onClick={handleCreate}
              disabled={creating || !createForm.name.trim() || !createForm.pin.trim()}
              class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              Создать
            </button>
            <button
              onClick={() => { setShowCreate(false); setCreateForm(DEFAULT_CREATE); }}
              class="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-600 hover:bg-gray-100"
            >
              Отмена
            </button>
          </div>
        </div>
      )}

      {/* Children list */}
      {children.length === 0 && !showCreate ? (
        <div class="py-12 text-center text-gray-500">
          <p class="text-lg">Детей пока нет</p>
          <p class="mt-1 text-sm">Добавьте ребёнка, нажав кнопку выше</p>
        </div>
      ) : (
        <div class="space-y-3">
          {children.map((child) => (
            <div key={child.id} class="rounded-lg border border-gray-200 bg-white p-4">
              {editId === child.id ? (
                <div class="space-y-3">
                  <div class="flex items-center gap-3 mb-2">
                    <span class="text-3xl">{editForm.avatar_emoji || child.avatar_emoji}</span>
                    <span class="text-sm font-medium text-gray-500">Редактирование</span>
                  </div>
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600">Имя</label>
                    <input
                      class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      value={editForm.name}
                      onInput={(e) => setEditForm((f) => ({ ...f, name: (e.target as HTMLInputElement).value }))}
                      autoFocus
                    />
                  </div>
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600">Новый PIN (оставьте пустым, чтобы не менять)</label>
                    <input
                      class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="Новый PIN"
                      type="text"
                      inputMode="numeric"
                      maxLength={4}
                      value={editForm.pin}
                      onInput={(e) => setEditForm((f) => ({ ...f, pin: (e.target as HTMLInputElement).value }))}
                    />
                  </div>
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600">Эмодзи-аватар</label>
                    <input
                      class="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      value={editForm.avatar_emoji}
                      onInput={(e) => setEditForm((f) => ({ ...f, avatar_emoji: (e.target as HTMLInputElement).value }))}
                    />
                  </div>
                  <div class="flex gap-2">
                    <button
                      onClick={handleSaveEdit}
                      disabled={saving}
                      class="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                    >
                      Сохранить
                    </button>
                    <button
                      onClick={() => setEditId(null)}
                      class="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
                    >
                      Отмена
                    </button>
                  </div>
                </div>
              ) : (
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <span class="text-3xl">{child.avatar_emoji}</span>
                    <div>
                      <p class="font-medium text-gray-900">{child.name}</p>
                      <p class="text-xs text-gray-400">
                        Добавлен {new Date(child.created_at).toLocaleDateString('ru-RU')}
                      </p>
                    </div>
                  </div>
                  <div class="flex gap-2">
                    <button
                      onClick={() => handleStartEdit(child)}
                      class="rounded-md border border-gray-200 px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-50"
                    >
                      Изменить
                    </button>
                    <button
                      onClick={() => handleDelete(child.id, child.name)}
                      class="rounded-md border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50"
                    >
                      Удалить
                    </button>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

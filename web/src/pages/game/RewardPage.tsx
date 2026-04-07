import { useEffect, useState } from 'preact/hooks';
import { useParams, useLocation } from 'wouter';
import { api } from '../../api';
import { Spinner } from '../../components/Spinner';
import { RewardScreen } from './RewardScreen';
import type { GamePuzzle } from '../../types';

export function RewardPage() {
  const { id } = useParams<{ id: string }>();
  const [, navigate] = useLocation();
  const [puzzle, setPuzzle] = useState<GamePuzzle | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    api.catalog
      .get(id)
      .then((p) => setPuzzle(p))
      .catch((e) => console.warn('reward load failed:', e))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) {
    return (
      <div class="flex h-dvh items-center justify-center">
        <Spinner />
      </div>
    );
  }

  return (
    <RewardScreen
      layers={puzzle?.layers ?? []}
      onReplay={() => navigate(`/play/${id}`)}
    />
  );
}

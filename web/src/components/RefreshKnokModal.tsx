import { useState } from 'react';
import type { FormEvent } from 'react';
import { Modal } from './Modal';
import type { KnokDto } from '../api/types';

interface RefreshKnokModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (url: string | undefined) => Promise<void>;
  knok: KnokDto;
}

export function RefreshKnokModal({
  isOpen,
  onClose,
  onConfirm,
  knok
}: RefreshKnokModalProps) {
  const [url, setUrl] = useState(knok.url);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsRefreshing(true);
    setError('');

    try {
      // If URL unchanged, send undefined to use existing URL
      const newUrl = url.trim() === knok.url ? undefined : url.trim();
      await onConfirm(newUrl);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to refresh knok');
    } finally {
      setIsRefreshing(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Refresh Knok">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label
            htmlFor="refresh-url"
            className="block text-sm font-medium text-stone-300 mb-2"
          >
            URL (optional - leave as-is to refresh current URL)
          </label>
          <input
            id="refresh-url"
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="w-full px-4 py-2 rounded bg-stone-800 text-white border border-stone-700 focus:border-blue-500 focus:outline-none"
            placeholder="https://example.com/track"
          />
        </div>

        {error && (
          <p className="text-red-500 text-sm">{error}</p>
        )}

        <div className="flex gap-3 justify-end">
          <button
            type="button"
            onClick={onClose}
            disabled={isRefreshing}
            className="px-4 py-2 bg-stone-700 hover:bg-stone-600 text-white rounded transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isRefreshing}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors disabled:opacity-50"
          >
            {isRefreshing ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </form>
    </Modal>
  );
}

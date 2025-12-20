import { useState } from 'react';
import { Modal } from './Modal';
import type { KnokDto } from '../api/types';

interface DeleteKnokModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
  knok: KnokDto;
}

export function DeleteKnokModal({
  isOpen,
  onClose,
  onConfirm,
  knok
}: DeleteKnokModalProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState('');

  const handleConfirm = async () => {
    setIsDeleting(true);
    setError('');

    try {
      await onConfirm();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete knok');
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Delete Knok">
      <div className="space-y-4">
        <p className="text-stone-300">
          Are you sure you want to delete this knok?
        </p>

        {knok.title && (
          <div className="bg-stone-800 p-3 rounded">
            <p className="text-sm font-medium text-white">{knok.title}</p>
            <p className="text-xs text-stone-400 mt-1">{knok.url}</p>
          </div>
        )}

        {error && (
          <p className="text-red-500 text-sm">{error}</p>
        )}

        <div className="flex gap-3 justify-end">
          <button
            onClick={onClose}
            disabled={isDeleting}
            className="px-4 py-2 bg-stone-700 hover:bg-stone-600 text-white rounded transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={isDeleting}
            className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors disabled:opacity-50"
          >
            {isDeleting ? 'Deleting...' : 'Yes, Delete'}
          </button>
        </div>
      </div>
    </Modal>
  );
}
